// xconns.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-12-01
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-12-07

package xconnpool

import (
	"github.com/X-Plan/xgo/go-xsched"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// 'XConns' is also a connection pool type based on 'XConnPool' type. The objective
// of designing it is to solve the redistribution problem of backend addresses, this
// can't be detected by 'XConnPool'. I recommend you use this instead of 'XConnPool'
// when backend addresses can be changed dynamically.
type XConns struct {
	capacity  int
	scheduler xsched.Scheduler
	pools     poolsType
}

func NewXConns(capacity int, scheduler xsched.Scheduler) *XConns {
	if capacity < 0 || scheduler == nil {
		return nil
	}

	xconns := &XConns{
		capacity:  capacity,
		scheduler: scheduler,
		pools:     poolsType{&sync.RWMutex{}, make(map[string]*XConnPool)},
	}
	go xconns.clean()
	return xconns
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

	pool := xconns.pools.Get(address)
	if pool == nil {
		pool = New(xconns.capacity, func() (net.Conn, error) {
			return net.Dial("tcp", address)
		})
		xconns.pools.Set(address, pool)
	}

	atomic.AddInt64(&(pool.count), 1)
	return pool, nil
}

func (xconns *XConns) clean() {
	for {
		time.Sleep(time.Hour)

		var addresses []string
		xconns.pools.ForEach(func(address string, pool *XConnPool) {
			count := atomic.LoadInt64(&(pool.count))
			if count != 0 {
				atomic.StoreInt64(&(pool.count), 0)
			} else {
				pool.Close()
				addresses = append(addresses, address)
			}
		})

		for _, address := range addresses {
			xconns.pools.Delete(address)
		}
	}
}

type poolsType struct {
	*sync.RWMutex
	pools map[string]*XConnPool
}

func (pt poolsType) Get(address string) *XConnPool {
	pt.RLock()
	pool := pt.pools[address]
	pt.RUnlock()
	return pool
}

func (pt poolsType) Set(address string, pool *XConnPool) {
	pt.Lock()
	pt.pools[address] = pool
	pt.Unlock()
}

func (pt poolsType) Delete(address string) {
	pt.Lock()
	delete(pt.pools, address)
	pt.Unlock()
}

func (pt poolsType) ForEach(cb func(string, *XConnPool)) {
	pt.RLock()
	for address, pool := range pt.pools {
		cb(address, pool)
	}
	pt.RUnlock()
}
