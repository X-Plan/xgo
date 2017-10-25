// router.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-02-07
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-10-25

package xp

import (
	"errors"
	"fmt"
	"github.com/X-Plan/xgo/go-xpacket"
	"github.com/golang/protobuf/proto"
	"io"
	"log"
	"net"
)

// 与cmd/subcmd相关的处理接口.
type XHandler interface {
	Handle(*Request) (*Response, error)
}

type XHandlerFunc func(*Request) (*Response, error)

func (xhf XHandlerFunc) Handle(req *Request) (*Response, error) {
	return xhf(req)
}

// 与鉴权相关的处理接口.
type XAuthHandler interface {
	Handle(*Header) error
}

type XAuthHandlerFunc func(*Header) error

func (xaf XAuthHandlerFunc) Handle(head *Header) error {
	return xaf(head)
}

type XRouter struct {
	ErrorLog *log.Logger
	mh       map[uint64]xhandlerPair
}

func NewXRouter() *XRouter {
	return &XRouter{mh: make(map[uint64]xhandlerPair)}
}

type xhandlerPair struct {
	xh   XHandler
	auth XAuthHandler
}

func (xr *XRouter) Register(cmd, subcmd uint32, xh XHandler, auth XAuthHandler) error {
	if xh == nil {
		return errors.New("XHandler can't be nil")
	}
	xr.mh[uint64(cmd)<<32+uint64(subcmd)] = xhandlerPair{xh, auth}
	return nil
}

func (xr *XRouter) Lookup(cmd, subcmd uint32) (XHandler, XAuthHandler) {
	if pair, ok := xr.mh[uint64(cmd)<<32+uint64(subcmd)]; ok {
		return pair.xh, pair.auth
	}
	return nil, nil
}

func (xr *XRouter) logf(format string, v ...interface{}) {
	if xr.ErrorLog != nil {
		xr.ErrorLog.Printf(format, v...)
	}
}

func (xr *XRouter) Handle(conn net.Conn, exit chan int) {
	xrc := &xrouterConn{
		conn:     conn,
		reqch:    make(chan *Request, 8),
		readDone: make(chan int, 1),
		exit:     exit,
		xr:       xr,
	}
	go xrc.read()
	xrc.write()
	conn.Close()
}

type xrouterConn struct {
	conn     net.Conn
	reqch    chan *Request
	readDone chan int
	exit     chan int
	xr       *XRouter
}

func (xrc *xrouterConn) read() {
	var (
		err  error
		data []byte
		req  = &Request{}
	)

	for {
		if data, err = xpacket.Decode(xrc.conn); err != nil {
			if err != io.EOF {
				xrc.xr.logf("xpacket decode failed (%s)", err)
			}
			xrc.readDone <- 1
			return
		}

		if err = proto.Unmarshal(data, req); err != nil {
			xrc.xr.logf("protobuf unmarshal failed (%s)", err)
		}

		xrc.reqch <- req
	}
}

func (xrc *xrouterConn) write() {
	var (
		code EnumRetCode
		err  error
		req  *Request
	)

outer:
	for {
		select {
		case req = <-xrc.reqch:
			if err, code = xrc.handleAndWrite(req); err != nil {
				xrc.xr.logf("%s", err)
			}
			// 只有错误信息为服务内部错误时服务端才
			// 主动断开连接.
			if code == EnumRetCode_SERVER_ERROR {
				break outer
			}
		case <-xrc.exit:
			break outer
		}
	}

	switch cc := xrc.conn.(type) {
	case *net.TCPConn:
		cc.CloseRead()
	case *net.UnixConn:
		cc.CloseRead()
	default:
		cc.Close()
	}
	<-xrc.readDone

	close(xrc.reqch)
	for req = range xrc.reqch {
		if err, _ = xrc.handleAndWrite(req); err != nil {
			xrc.xr.logf("%s", err)
		}
	}

	return
}

func (xrc *xrouterConn) handleAndWrite(req *Request) (error, EnumRetCode) {
	var (
		err      error
		data     []byte
		rsp      *Response
		head     = req.GetHead()
		xh, auth = xrc.xr.Lookup(head.GetCmd(), head.GetSubCmd())
	)

	if auth != nil {
		if err = auth.Handle(head); err != nil {
			xpacket.Encode(xrc.conn, createResponseData(head, EnumRetCode_AUTH_FAILED, err.Error()))
			return err, EnumRetCode_AUTH_FAILED
		}
	}

	if xh == nil {
		msg := fmt.Sprintf("invalid cmd/subcmd (%d/%d)", head.GetCmd(), head.GetSubCmd())
		xpacket.Encode(xrc.conn, createResponseData(head, EnumRetCode_REQUEST_ERROR, msg))
		return errors.New(msg), EnumRetCode_REQUEST_ERROR
	}

	if rsp, err = xh.Handle(req); err != nil {
		msg := fmt.Sprintf("server internal error (%s)", err)
		xpacket.Encode(xrc.conn, createResponseData(head, EnumRetCode_SERVER_ERROR, msg))
		return errors.New(msg), EnumRetCode_SERVER_ERROR
	}
	rsp.Head = req.GetHead()
	if rsp.Ret == nil {
		rsp.Ret = &Return{Msg: "ok"}
	}

	if data, err = proto.Marshal(rsp); err == nil {
		if err = xpacket.Encode(xrc.conn, data); err != nil {
			return err, EnumRetCode_SERVER_ERROR
		}
	} else {
		return err, EnumRetCode_SERVER_ERROR
	}

	return nil, EnumRetCode_OK
}

func createResponseData(head *Header, code EnumRetCode, msg string) []byte {
	data, _ := proto.Marshal(&Response{Head: head, Ret: &Return{Code: int32(code), Msg: msg}})
	return data
}
