// client.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-11-03
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-12-15

package xp

import (
	"fmt"
	"github.com/X-Plan/xgo/go-xconnpool"
	"github.com/X-Plan/xgo/go-xpacket"
	"github.com/X-Plan/xgo/go-xretry"
	"github.com/X-Plan/xgo/go-xsched"
	"github.com/golang/protobuf/proto"
	"net"
	"sync/atomic"
)

type Client struct {
	// Scheduler is used to tell a client instance where is destination,
	// so this field can't be empty. Its implementation must be satisfy
	// 'xscheduler.Scheduler' interface, the more detail you can get from
	// 'go-xscheduler' package.
	Scheduler xsched.Scheduler

	// The internal of a client will maintain a connection pool for each
	// address returned by 'Scheduler' field, so this field is used to
	// specify the size of each pool. If this field is zero, using 32 by
	// default, but this field can't be negative.
	PoolSize int

	seq  uint64
	xcps *xconnpool.XConnPools
}

// Send a request to a server and receive the response. If "Sequence" field in
// a request is empty, it will be filled automatically (this will change the
// request argument, so you should be careful).
func (client *Client) RoundTrip(req *Request) (*Response, error) {

	// The first time we call this function, we need to init our client instance.
	if client.xcps == nil {
		if err := client.init(); err != nil {
			return nil, err
		}
	}

	if req.Head.Sequence == 0 {
		req.Head.Sequence = atomic.AddUint64(&client.seq, uint64(1))
	}

	conn, err := client.xcps.Get()
	if err != nil {
		if fatal, ok := err.(xconnpool.GetConnError); ok {
			client.Scheduler.Feedback(fatal.Addr, false)
			return nil, fatal.Err
		} else {
			// I use 'xretry.FatalError' to wrap the raw error, this
			// characteristic can be used by 'go-xretry' package, and
			// it won't affect the raw error.
			return nil, xretry.FatalError{err}
		}
	}
	defer conn.Close()

	data, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}

	if err = xpacket.Encode(conn, data); err != nil {
		client.release(conn)
		return nil, err
	}

	if data, err = xpacket.Decode(conn); err != nil {
		client.release(conn)
		return nil, err
	}

	rsp := &Response{}
	if err = proto.Unmarshal(data, rsp); err != nil {
		client.release(conn)
		return nil, err
	}

	// If the error comes from a server end, we need to report it.
	if rsp.GetRet().GetCode() == int32(Code_SERVER_ERROR) {
		client.release(conn)
		return rsp, nil
	}

	client.Scheduler.Feedback(conn.RemoteAddr().String(), true)
	return rsp, nil
}

func (client *Client) init() error {
	if client.Scheduler == nil {
		return fmt.Errorf("invalid scheduler field")
	}

	size := client.PoolSize
	if size < 0 {
		return fmt.Errorf("pool size (%d) can't be negative", size)
	} else if size == 0 {
		size = 32
	}

	client.xcps = xconnpool.NewXConnPools(size, client.Scheduler)
	return nil
}

func (client *Client) release(conn net.Conn) {
	client.Scheduler.Feedback(conn.RemoteAddr().String(), false)
	conn.(*xconnpool.XConn).Unuse()
}
