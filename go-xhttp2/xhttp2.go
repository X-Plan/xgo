// xhttp2.go
//
//      Copyright (C), blinklv. All rights reserved.
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-12-12
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2016-12-12

// go-xhttp2是net/http的一个扩展包, 提供了HTTP/1.1通过
// Upgrade的方式升级到HTTP/2的支持.
package xhttp2

import (
	"net"
	"net/http"
)

type XServer struct {
	hs  *http.Server
	h2s *xhttp2Server
}

func New(hs *http.Server) *XServer {
}

func (xs *XServer) Serve(l net.Listener) error {
	return xs.hs.Serve(l)
}

func (xs *XServer) ListenAndServe() error {
	return xs.hs.ListenAndServe()
}

func (xs *XServer) ListenAndServeTLS(pem, key string) error {
	return xs.hs.ListenAndServeTLS(pem, key)
}

func (xs *XServer) SetKeepAlivesEnabled(v bool) {
	xs.hs.SetKeepAlivesEnabled(v)
}

type xhandler struct {
	h  http.Handler
	xs *XServer
	sc *xhttp2serverConn
}

func (xh xhandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	var (
		ret int
		err error
		c   net.Conn
	)

	ret, err = xh.handshake(rw, req)
	switch ret {
	case 0:
		xh.h.ServeHTTP(rw, req)
	case 1:
		hj, ok := rw.(Hijacker)
		if !ok {
			http.Error(rw, "WebServer doesn't support hijacking", 500)
			break
		}

		if c, _, err = hj.Hijack(); err != nil {
			http.Error(rw, err.Error(), 500)
			break
		}

		opts := &xhttp2ServeConnOpts{
			Handler:    xh.h,
			BaseConfig: xh.xs.hs,
		}

		xh.sc = XNewxhttp2serverConn(c, opts)
		xh.xs.h2s.XServeConn(xh.sc, c, opts)

	default:
		http.Error(rw, err.Error(), ret)
	}
}

func (xh xhandler) handshake(rw http.ResponseWriter, req *http.Request) (ret int, err error) {
}

// 下面的函数是对h2_bundle.go中成员的扩展,
// 主要是为了提供对外对接口供go-xhttp2包
// 使用.
func (s *xhttp2Server) XServeConn(sc *xhttp2serverConn, c net.Conn, opts *xhttp2ServeConnOpts) {
	baseCtx, cancel := xhttp2serverConnBaseContext(c, opts)
	defer cancel()

	if s.NewWriteScheduler != nil {
		sc.writeSched = s.NewWriteScheduler()
	} else {
		sc.writeSched = xhttp2NewRandomWriteScheduler()
	}

	sc.flow.add(xhttp2initialWindowSize)
	sc.inflow.add(xhttp2initialWindowSize)
	sc.hpackEncoder = hpack.NewEncoder(&sc.headerWriteBuf)

	fr := xhttp2NewFramer(sc.bw, c)
	fr.ReadMetaHeaders = hpack.NewDecoder(xhttp2initialHeaderTableSize, nil)
	fr.MaxHeaderListSize = sc.maxHeaderListSize()
	fr.SetMaxReadFrameSize(s.maxReadFrameSize())
	sc.framer = fr

	if tc, ok := c.(xhttp2connectionStater); ok {
		sc.tlsState = new(tls.ConnectionState)
		*sc.tlsState = tc.ConnectionState()

		if sc.tlsState.Version < tls.VersionTLS12 {
			sc.rejectConn(xhttp2ErrCodeInadequateSecurity, "TLS version too low")
			return
		}

		if sc.tlsState.ServerName == "" {

		}

		if !s.PermitProhibitedCipherSuites && xhttp2isBadCipher(sc.tlsState.CipherSuite) {

			sc.rejectConn(xhttp2ErrCodeInadequateSecurity, fmt.Sprintf("Prohibited TLS 1.2 Cipher Suite: %x", sc.tlsState.CipherSuite))
			return
		}
	}

	if hook := xhttp2testHookGetServerConn; hook != nil {
		hook(sc)
	}
	sc.serve()
}

func (s *xhttp2Server) XNewxhttp2serverConn(c net.Conn, opts *xhttp2ServeConnOpts) *xhttp2serverConn {
	return &xhttp2serverConn{
		srv:               s,
		hs:                opts.baseConfig(),
		conn:              c,
		baseCtx:           baseCtx,
		remoteAddrStr:     c.RemoteAddr().String(),
		bw:                xhttp2newBufferedWriter(c),
		handler:           opts.handler(),
		streams:           make(map[uint32]*xhttp2stream),
		readFrameCh:       make(chan xhttp2readFrameResult),
		wantWriteFrameCh:  make(chan xhttp2FrameWriteRequest, 8),
		wantStartPushCh:   make(chan xhttp2startPushRequest, 8),
		wroteFrameCh:      make(chan xhttp2frameWriteResult, 1),
		bodyReadCh:        make(chan xhttp2bodyReadMsg),
		doneServing:       make(chan struct{}),
		clientMaxStreams:  math.MaxUint32,
		advMaxStreams:     s.maxConcurrentStreams(),
		initialWindowSize: xhttp2initialWindowSize,
		maxFrameSize:      xhttp2initialMaxFrameSize,
		headerTableSize:   xhttp2initialHeaderTableSize,
		serveG:            xhttp2newGoroutineLock(),
		pushEnabled:       true,
	}
}
