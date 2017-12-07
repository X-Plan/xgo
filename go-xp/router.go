// router.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-11-03
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-12-07

package xp

import (
	"fmt"
	"github.com/X-Plan/xgo/go-xlog"
	"github.com/X-Plan/xgo/go-xpacket"
	"github.com/X-Plan/xgo/go-xtcpapi"
	"github.com/golang/protobuf/proto"
	"io"
	"net"
)

// A handler responds to an X-Protocol request.
type Handler interface {
	Handle(*Request) (*Response, error)
}

// HandlerFunc type is an adapter to allow the use of ordinary functions as
// X-Protocol handlers.
type HandlerFunc func(*Request) (*Response, error)

func (f HandlerFunc) Handle(req *Request) (*Response, error) {
	return f(req)
}

// An authentication handler.
type AuthHandler interface {
	Handle(*Header) error
}

// AuthHandlerFunc type is an adapter to allow the use of ordinary functions as
// X-Protocol auth handlers.
type AuthHandlerFunc func(*Header) error

func (auth AuthHandlerFunc) Handle(header *Header) error {
	return auth(header)
}

type handlerPair struct {
	handler Handler
	auth    AuthHandler
}

// Router is an implementation of 'ConnHandler' interface. It reads the data from
// a connection and dispatches it to the corresponding handler based on command.
type Router struct {
	Logger *xlog.XLogger // This field can be empty, but I recommend you use it.
	pairs  map[uint64]handlerPair
}

// Bind a handler and an authentication handler to a command pair. If the command
// pair of a request matches it, the corresponding handler and authentication handler
// will be called to handle this request. 'handler' parameter can't be empty, but
// 'auth' can be, it means there is no authentication scheme for this command pair.
// If the command pairs of two handlers are same, the second one will overwrite the
// first one.
func (r *Router) Bind(cmd, subcmd uint32, handler Handler, auth AuthHandler) error {
	if handler == nil {
		return fmt.Errorf("handler can't be empty")
	}
	if r.pairs == nil {
		r.pairs = make(map[uint64]handlerPair)
	}
	r.pairs[uint64(cmd)<<32+uint64(subcmd)] = handlerPair{handler, auth}
	return nil
}

// Return the handler pair corresponding a command pair, this function can be used
// to check whether a command pair has been bound.
func (r *Router) Lookup(cmd, subcmd uint32) (Handler, AuthHandler) {
	if pair, ok := r.pairs[uint64(cmd)<<32+uint64(subcmd)]; ok {
		return pair.handler, pair.auth
	}
	return nil, nil
}

func (r *Router) Handle(conn net.Conn, exit chan int) {
	cw := &connWrapper{
		conn:     conn,
		queue:    make(chan *Request, 32),
		readDone: make(chan int, 1),
		exit:     exit,
		r:        r,
	}
	go cw.read()
	cw.write()
	conn.Close()
}

func (r *Router) errorf(format string, args ...interface{}) {
	if r.Logger != nil {
		r.Logger.Error(format, args...)
	}
}

type connWrapper struct {
	conn     net.Conn
	queue    chan *Request
	readDone chan int
	exit     chan int
	r        *Router
}

func (cw *connWrapper) read() {
	for {
		data, err := xpacket.Decode(cw.conn)
		if err != nil {
			if err != io.EOF && !xtcpapi.IsErrClosing(err) {
				cw.r.errorf("decode packet failed (%s)", err)
			}
			cw.readDone <- 1
			return
		}

		req := &Request{}
		if err = proto.Unmarshal(data, req); err != nil {
			cw.r.errorf("protobuf unmarshal failed (%s)", err)
		}

		cw.queue <- req
	}
}

func (cw *connWrapper) write() {
	var (
		code Code
		err  error
		req  *Request
	)

outer:
	for {
		select {
		case req = <-cw.queue:
			if err, code = cw.handle(req); err != nil {
				cw.r.errorf("handle request failed (%s)", err)
			}

			if code == Code_SERVER_ERROR {
				break outer
			}
		case <-cw.readDone:
			break outer
		case <-cw.exit:
			break outer
		}
	}

	switch c := cw.conn.(type) {
	case *net.TCPConn:
		c.CloseRead()
	case *net.UnixConn:
		c.CloseRead()
	default:
		c.Close()
	}

	close(cw.queue)
	for req = range cw.queue {
		if err, _ = cw.handle(req); err != nil {
			cw.r.errorf("handle request failed (%s)", err)
		}
	}

	return
}

func (cw *connWrapper) handle(req *Request) (err error, code Code) {
	code = Code_OK
	head := req.GetHead()
	handler, auth := cw.r.Lookup(head.GetCmd(), head.GetSubCmd())

	defer func() {
		if err != nil {
			xpacket.Encode(cw.conn, response(head, code, err.Error()))
		}
	}()

	if auth != nil {
		if err = auth.Handle(head); err != nil {
			code = Code_AUTH_FAILED
			return
		}
	}

	if handler == nil {
		err, code = fmt.Errorf("invalid cmd/subcmd (%d/%d)", head.GetCmd(), head.GetSubCmd()), Code_REQUEST_ERROR
		return
	}

	var rsp *Response
	if rsp, err = handler.Handle(req); err != nil {
		code = Code_SERVER_ERROR
		return
	}

	rsp.Head = req.GetHead()
	// If caller doesn't fill 'ret' field, fill it automatically.
	if rsp.Ret == nil {
		rsp.Ret = &Return{Msg: "ok"}
	}

	var data []byte
	if data, err = proto.Marshal(rsp); err != nil {
		code = Code_SERVER_ERROR
		return
	}

	if err = xpacket.Encode(cw.conn, data); err != nil {
		code = Code_SERVER_ERROR
	}
	return
}

func response(head *Header, code Code, msg string) []byte {
	data, _ := proto.Marshal(&Response{Head: head, Ret: &Return{Code: int32(code), Msg: msg}})
	return data
}
