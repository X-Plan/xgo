// xconns.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-12-01
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-12-01

package xconnpool

import (
	"github.com/X-Plan/xgo/go-xsched"
	"net"
	"sync"
	"sync/atomic"
)

// 'XConns' is also a connection pool type based on 'XConnPool' type. The objective
// of designing it is to solve the redistribution problem of backend addresses, this
// can't be detected by 'XConnPool'. I recommend you use this instead of 'XConnPool'
// when backend addresses can be changed dynamically.
type XConns struct {
	capacity  int
	scheduler xsched.Scheduler
	pools     map[string]*XConnPool
}

func NewXConns(capacity int, scheduler xsched.Scheduler) *XConns {
	if capacity < 0 || scheduler == nil {
		return nil
	}

	return &XConns{
		capacity:  capacity,
		scheduler: scheduler,
		pools:     make(map[string]*XConnPool),
	}
}

func (xconns *XConns) Get() (net.Conn, error) {
	if pool, err := xconns.selectPool(); err == nil {
		return pool.Get()
	} else {
		return nil, err
	}
}

func (xconns *XConns) RawGet() (net.Conn, error) {
	if pool, err := xconns.selectPool(); err == nil {
		return pool.RawGet()
	} else {
		return nil, err
	}
}

func (xconns *XConns) selectPool() (*XConnPool, error) {
	address, err := xconns.scheduler.Get()
	if err != nil {
		return nil, err
	}

	pool := xconns.pools[address]
	if pool == nil {
		pool = New(xconns.capacity, func() (net.Conn, error) {
			return net.Dial("tcp", address)
		})
		xconns.pools[address] = pool
	}

	atomic.AddInt64(&(pool.count), 1)
	return pool, nil
}

func (xconns *XConns) clean() {
}
