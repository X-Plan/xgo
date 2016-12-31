// xhttp2.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-12-12
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2016-12-31

// go-xhttp2是net/http的一个扩展包, 提供了HTTP/1.1通过
// Upgrade的方式升级到HTTP/2的支持.
package xhttp2

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"golang.org/x/net/http2/hpack"
	"math"
	"net"
	"net/http"
	"strings"
)

type XServer struct {
	hs  *http.Server
	h2s *xhttp2Server
}

func New(hs *http.Server) *XServer {
	if hs == nil {
		return nil
	}

	if hs.Handler == nil {
		hs.Handler = http.DefaultServeMux
	}

	xs := &XServer{h2s: new(xhttp2Server)}
	hs.Handler = xhandler{
		h:  hs.Handler,
		xs: xs,
	}

	return xs
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
}

func (xh xhandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	var (
		ret           int
		err           error
		c             net.Conn
		http2settings []xhttp2Setting
	)

	ret, err, http2settings = handshake(rw, req)
	switch ret {
	case 0:
		xh.h.ServeHTTP(rw, req)
	case 1:
		hj, ok := rw.(http.Hijacker)
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

		sc := xh.xs.h2s.XNewxhttp2serverConn(c, opts)
		sc.XInitSetting(http2settings)
		xh.xs.h2s.XServeConn(sc, c, opts)

	default:
		http.Error(rw, err.Error(), ret)
	}
}

func handshake(rw http.ResponseWriter, req *http.Request) (ret int, err error, http2settings []xhttp2Setting) {
	// 只有在HTTP/1.1并且携带Upgrade: h2c的情形下才进行
	// HTTP/2的握手操作.
	if req.ProtoMajor == 1 && req.ProtoMinor == 2 &&
		strings.ToLower(req.Header.Get("Upgrade")) == "h2c" {

		// 大部分服务都没有支持OPTIONS操作,
		// 我决定随大流, 哈哈. :)
		if req.Method == "OPTIONS" {
			ret, err = 405, fmt.Errorf("Method Not Allowed")
			return
		}

		ch := strings.ToLower(req.Header.Get("Connection"))
		if strings.Replace(ch, " ", "", -1) != "upgrade,http2-settings" {
			ret, err = 400, fmt.Errorf("Invalid Connection Header")
			return
		}

		// HTTP2-Settings头部中存放了初始设置信息.
		// 参见RFC7540 (3.2.1)
		_, ok := ((map[string][]string)(req.Header))["Http2-Settings"]
		if !ok {
			ret, err = 400, fmt.Errorf("Missing HTTP2-Settings Header")
			return
		}
		raw := req.Header.Get("HTTP2-Settings")

		if http2settings, err = parseHTTP2Settings(raw); err != nil {
			ret = 400
			return
		}

		ret, err = 1, nil
	}

	return
}

func parseHTTP2Settings(raw string) ([]xhttp2Setting, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var (
		payload       []byte
		err           error
		http2settings []xhttp2Setting
	)

	if payload, err = base64.URLEncoding.DecodeString(raw); err != nil {
		goto error_exit
	}

	if len(payload)%6 != 0 {
		goto error_exit
	}

	for len(payload) > 0 {
		id := binary.BigEndian.Uint16(payload[:2])
		if id < uint16(0x1) || id > uint16(0x6) {
			err = fmt.Errorf("Invalid Setting Id - %d", id)
			goto error_exit
		}

		val := binary.BigEndian.Uint32(payload[2:6])
		if xhttp2SettingID(id) == xhttp2SettingInitialWindowSize {
			if val > math.MaxInt32 {
				err = fmt.Errorf("Initial Window Size Overflow - %d", val)
				goto error_exit
			}
		}

		http2settings = append(http2settings, xhttp2Setting{
			xhttp2SettingID(id), val,
		})
		payload = payload[6:]
	}

	return http2settings, nil

error_exit:
	return nil, fmt.Errorf("Invalid HTTP2-Settings Header (%s)", err)
}

// 下面的函数是对h2_bundle.go中成员的扩展,
// 主要是为了提供对外对接口供go-xhttp2包
// 使用.
func (s *xhttp2Server) XServeConn(sc *xhttp2serverConn, c net.Conn, opts *xhttp2ServeConnOpts) {
	baseCtx, cancel := xhttp2serverConnBaseContext(c, opts)
	defer cancel()

	sc.baseCtx = baseCtx

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

func (sc *xhttp2serverConn) XInitSetting(xhttp2settings []xhttp2Setting) {
	for _, s := range xhttp2settings {
		switch s.ID {
		case xhttp2SettingHeaderTableSize:
			sc.headerTableSize = s.Val
			sc.hpackEncoder.SetMaxDynamicTableSize(s.Val)
		case xhttp2SettingEnablePush:
			sc.pushEnabled = s.Val != 0
		case xhttp2SettingMaxConcurrentStreams:
			sc.clientMaxStreams = s.Val
		case xhttp2SettingInitialWindowSize:
			sc.initialWindowSize = int32(s.Val)
		case xhttp2SettingMaxFrameSize:
			sc.maxFrameSize = int32(s.Val)
		case xhttp2SettingMaxHeaderListSize:
			sc.peerMaxHeaderListSize = s.Val
		default:
			// 其它情况忽略.
		}
	}
}
