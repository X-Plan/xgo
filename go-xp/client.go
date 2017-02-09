// client.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-02-06
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-02-09

package xp

import (
	"fmt"
	"github.com/X-Plan/xgo/go-xconnpool"
	"github.com/X-Plan/xgo/go-xpacket"
	"github.com/X-Plan/xgo/go-xretry"
	"github.com/X-Plan/xgo/go-xvalid"
	"github.com/golang/protobuf/proto"
	"net"
	"sync/atomic"
	"time"
)

type XScheduler interface {
	Get() (string, error)
	Feedback(string, bool)
}

type XClientConfig struct {
	RetryCount   int           `xvalid:"min=0"`               // 重试次数.
	Interval     time.Duration `xvalid:"min=0,default=100ms"` // 重试的基础时间间隔.
	ConnPoolSize int           `xvalid:"min=0,default=16"`    // 连接池大小.
	Scheduler    XScheduler    `xvalid:"noempty"`             // 地址调度器.
}

type XClient struct {
	rc       int
	interval time.Duration
	seq      uint64
	xcp      *xconnpool.XConnPool
	sched    XScheduler
}

func NewXClient(cfg *XClientConfig) (*XClient, error) {
	err := xvalid.Validate(cfg)
	if err != nil {
		return nil, err
	}

	xcli := &XClient{
		rc:       cfg.RetryCount,
		interval: cfg.Interval,
		sched:    cfg.Scheduler,
	}

	xcli.xcp = xconnpool.New(cfg.ConnPoolSize, func() (net.Conn, error) {
		addr, err := xcli.sched.Get()
		if err != nil {
			return nil, err
		}

		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return nil, xconnpool.GetConnError{Err: err, Addr: addr}
		}

		return conn, nil
	})

	return xcli, nil
}

// Request.Header中的Sequence字段如果未填, 则会使用
// XClient自己默认的序列号.
func (xcli *XClient) Send(req *Request) (*Response, error) {
	if req.Head.Sequence == 0 {
		req.Head.Sequence = atomic.AddUint64(&xcli.seq, uint64(1))
	}
	var rsp = &Response{}

	err, _ := xretry.Retry(func() error {
		conn, err := xcli.xcp.Get()
		if err != nil {
			if getConnErr, ok := err.(xconnpool.GetConnError); ok {
				xcli.sched.Feedback(getConnErr.Addr, false)
				return getConnErr.Err
			} else {
				return xretry.FatalError{err}
			}
		}
		defer conn.Close()

		data, err := proto.Marshal(req)
		if err != nil {
			return xretry.FatalError{err}
		}

		if err = xpacket.Encode(conn, data); err != nil {
			xcli.sched.Feedback(conn.RemoteAddr().String(), false)
			conn.(*xconnpool.XConn).Unuse()
			return err
		}

		if data, err = xpacket.Decode(conn); err != nil {
			xcli.sched.Feedback(conn.RemoteAddr().String(), false)
			conn.(*xconnpool.XConn).Unuse()
			return err
		}

		if err = proto.Unmarshal(data, rsp); err != nil {
			xcli.sched.Feedback(conn.RemoteAddr().String(), false)
			conn.(*xconnpool.XConn).Unuse()
			return err
		}

		// 服务端错误需要上报.
		if rsp.GetRet().GetCode() == int32(EnumRetCode_SERVER_ERROR) {
			xcli.sched.Feedback(conn.RemoteAddr().String(), false)
			conn.(*xconnpool.XConn).Unuse()
			return fmt.Errorf("%s: %s", EnumRetCode_SERVER_ERROR, rsp.GetRet().GetMsg())
		}

		xcli.sched.Feedback(conn.RemoteAddr().String(), true)
		return nil
	}, xcli.rc, xcli.interval)

	return rsp, err
}
