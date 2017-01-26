// xconnpool_test.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-10-15
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-01-25
package xconnpool

import (
	"context"
	"fmt"
	"github.com/X-Plan/xgo/go-xassert"
	"github.com/X-Plan/xgo/go-xpacket"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func setupClient(capacity, count, n int, addrList []string) error {
	var (
		err   error
		conn  net.Conn
		index int64
		xcp   = New(capacity, func() (net.Conn, error) {
			return net.Dial("tcp", addrList[atomic.AddInt64(&index, 1)%len(addrList)])
		})
	)
	defer xcp.Close()

	for i := 0; i < count; i++ {
		if n == 0 {
			fmt.Printf("[client] Game Over?\n")
			break
		}

		if conn, err = xcp.Get(); err != nil {
			fmt.Printf("[client] get connection failed (%s)\n", err)
			n--
			continue
		}

	again:
		if err = xpacket.Encode(conn, []byte(fmt.Sprintf("message %d", i))); err != nil {
			err = conn.(*xconnpool.XConn).Release()
			fmt.Printf("[client] write message %d failed, release connection (%s)\n", err)
			if conn, err = xcp.RawGet(); err != nil {
				fmt.Printf("[client] (raw) get connection failed (%s)\n", err)
				n--
				continue
			}
			goto again
		}
		conn.Close()
	}
}

func setupServer(n int, lifetime time.Duration) ([]string, error) {
	var (
		addrList = make([]string, 0, n)
		ls       = make([]net.Listener, 0, n)
		l        net.Listener
		err      error
	)

	for i := 0; i < n; i++ {
		if l, err = net.Listen("0.0.0.0:0"); err != nil {
			return nil, err
		}
		ls = append(ls, l)
		addrList = append(addrList, l.Addr().String())
	}

	for i, l = range ls {
		go server(l, lifetime, i+1)
	}

	return addrList
}

func server(l net.Listener, lifetime time.Duration, sid int) {
	var (
		wg          = &sync.WaitGroup{}
		ctx, cancel = context.WithTimeout(context.TODO(), lifetime)
		cid         int
	)

	go func(ctx context.Context) {
		select {
		case <-ctx.Done():
			l.Close()
		}
	}(ctx)

	for {
		if conn, err := l.Accept(); err != nil {
			break
		}
		cid++

		wg.Add(1)
		go func() {
			handle(ctx, conn, sid, cid)
			wg.Done()
		}()
	}

	wg.Wait()
	fmt.Printf("[server %d] timeout exit (%s)", sid, lifetime)
	cancel()
}

func handle(ctx context.Context, conn net.Conn, sid, cid int) {
	var (
		data []byte
		err  error
	)
	fmt.Printf("[server: %d][connection: %d] accept connection\n", sid, cid)

	for {
		select {
		case <-ctx.Done():
			conn.Close()
			fmt.Printf("[server: %d][connection: %d] timeout exit\n", sid, cid)
			return
		default:
			if data, err = xpacket.Decode(conn); err != nil {
				fmt.Printf("[server: %d][connection: %d] exit (%s)\n", sid, cid, err)
				return
			}

			if err = xpacket.Encode(conn, data); err != nil {
				fmt.Printf("[server: %d][connection: %d] exit (%s)\n", sid, cid, err)
				return
			}
		}
	}
}
