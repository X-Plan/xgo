// xconnpools.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-12-01
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-12-28

package xconnpool

import (
	"github.com/X-Plan/xgo/go-xsched"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// 'XConnPools' is also a connection pool type based on 'XConnPool' type. The objective
// of designing it is to solve the redistribution problem of backend addresses, this
// can't be detected by 'XConnPool'. I recommend you use this instead of 'XConnPool'
// when backend addresses can be changed dynamically.
type XConnPools struct {
	capacity  int
	exit      chan struct{}
	scheduler xsched.Scheduler
	pools     poolsType
	dial      Dial
}

type Dial func(network, address string) (net.Conn, error)

// Create a new 'XConnPools'. 'XConnPools' will create a single connection pool for each
// address returned by 'scheduler' parameter. 'capacity' means the capacity of each pool
// instead of total capacity. If 'capacity' is less than or equal to zero or 'scheduler'
// is nil, returns nil. 'dial' is used to create new connections.
func NewXConnPools(capacity int, scheduler xsched.Scheduler, dial Dial) *XConnPools {
	if capacity <= 0 || scheduler == nil {
		return nil
	}

	xcps := &XConnPools{
		capacity:  capacity,
		exit:      make(chan struct{}),
		scheduler: scheduler,
		pools:     poolsType{&sync.RWMutex{}, make(map[string]*XConnPool)},
		dial:      dial,
	}
	go xcps.clean()
	return xcps
}

// Get a connection from the pools.
func (xcps *XConnPools) Get() (net.Conn, error) {
	if pool, err := xcps.selectPool(); err == nil {
		return pool.Get()
	} else {
		return nil, err
	}
}

// Get a new connection, ignore existing free connections.
func (xcps *XConnPools) RawGet() (net.Conn, error) {
	if pool, err := xcps.selectPool(); err == nil {
		return pool.RawGet()
	} else {
		return nil, err
	}
}

// Close the pools.
func (xcps *XConnPools) Close() (err error) {
	close(xcps.exit) // Notify 'clean' routine to exit.
	return xcps.pools.ForEach(func(_ string, pool *XConnPool) error {
		// Ignoring an error returned by 'Close' function of each pool.
		pool.Close()
		return nil
	})
}

// Select a pool from pools based on the address returned by scheduler.
// If there exists the pool corresponding to an address, return it directly.
// Otherwise, create a new one and return it.
func (xcps *XConnPools) selectPool() (*XConnPool, error) {
	address, err := xcps.scheduler.Get()
	if err != nil {
		return nil, err
	}

	pool := xcps.pools.Get(address)
	if pool == nil {
		pool = New(xcps.capacity, func() (net.Conn, error) {
			return xcps.dial("tcp", address)
		})
		xcps.pools.Set(address, pool)
	}

	atomic.AddInt64(&(pool.count), 1)
	return pool, nil
}

var cleanPeriod time.Duration // This variable is only used to debug.

// If a pool has never be used in the last one hour, it will be closed and
// removed from the pools.
func (xcps *XConnPools) clean() {
	var ticker *time.Ticker
	if cleanPeriod == time.Duration(0) {
		ticker = time.NewTicker(time.Hour)
	} else {
		ticker = time.NewTicker(cleanPeriod)
	}

	for {
		select {
		case <-ticker.C:
			var addresses []string
			xcps.pools.ForEach(func(address string, pool *XConnPool) error {
				count := atomic.LoadInt64(&(pool.count))
				if count != 0 {
					atomic.StoreInt64(&(pool.count), 0)
				} else {
					// We can close a pool at this callback function, but we can't
					// delete it from the pools, because this will lead to deadlock.
					pool.Close()
					addresses = append(addresses, address)
				}
				return nil
			})

			// Out of ForEach loop, we can delete deprecated pools safely.
			for _, address := range addresses {
				xcps.pools.Delete(address)
			}

		case <-xcps.exit:
			ticker.Stop()
			return
		}
	}
}

// An auxiliary type to guarantee access 'pools' field is concurrent security.
type poolsType struct {
	*sync.RWMutex
	pools map[string]*XConnPool
}

func (pt *poolsType) Get(address string) *XConnPool {
	pt.RLock()
	pool := pt.pools[address]
	pt.RUnlock()
	return pool
}

func (pt *poolsType) Set(address string, pool *XConnPool) {
	pt.Lock()
	pt.pools[address] = pool
	pt.Unlock()
}

func (pt *poolsType) Delete(address string) {
	pt.Lock()
	delete(pt.pools, address)
	pt.Unlock()
}

// Iterate each item in 'pools' and handle it by using a user-defined callback function.
// NOTE: The implementation of a callback function shouldn't contain any operation of
// pools itself (Get, Set, Delete and ForEach), this will lead to deadlock.
func (pt *poolsType) ForEach(cb func(string, *XConnPool) error) error {
	var err error
	pt.RLock()
	for address, pool := range pt.pools {
		if err = cb(address, pool); err != nil {
			break
		}
	}
	pt.RUnlock()
	return err
}
