// xconnpool_test.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-10-15
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-03-02
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

func TestDuplicateClose(t *testing.T) {
	addrList, err, _ := setupServer(1, 1*time.Second, 5*time.Second)
	xassert.IsNil(t, err)
	xcp := New(10, func() (net.Conn, error) {
		return net.Dial("tcp", addrList[0])
	})
	conn, err := xcp.Get()
	xassert.IsNil(t, err)
	xassert.IsNil(t, conn.Close())
	xassert.Equal(t, conn.Close(), ErrXConnClosed)
	xassert.Equal(t, conn.(*XConn).Release(), ErrXConnClosed)

	conn, err = xcp.Get()
	xassert.IsNil(t, err)
	xassert.IsNil(t, conn.(*XConn).Release(), ErrXConnClosed)
	xassert.Equal(t, conn.Close(), ErrXConnClosed)
	xassert.Equal(t, conn.(*XConn).Release(), ErrXConnClosed)
}

func TestUnuse(t *testing.T) {
	addrList, err, wg := setupServer(1, 1*time.Second, 5*time.Second)
	xassert.IsNil(t, err)
	xcp := New(1, func() (net.Conn, error) {
		return net.Dial("tcp", addrList[0])
	})

	conn1, err := xcp.Get()
	xassert.IsNil(t, err)
	xassert.IsNil(t, conn1.Close())

	conn2, err := xcp.Get()
	xassert.IsNil(t, err)
	conn2.(*XConn).Unuse()
	xassert.IsNil(t, conn2.Close())

	conn3, err := xcp.Get()
	xassert.IsNil(t, err)
	xassert.IsNil(t, conn3.(*XConn).Release())

	wg.Wait()
}

func Test1(t *testing.T) {
	addrList, err, wg := setupServer(5, time.Second, 50*time.Second)
	xassert.IsNil(t, err)
	setupClient(10, 100, 5, 200*time.Millisecond, addrList)
	wg.Wait()
}

func Test2(t *testing.T) {
	addrList, err, wg := setupServer(3, 2*time.Second, 50*time.Second)
	xassert.IsNil(t, err)
	setupClient(10, 100, 5, 200*time.Millisecond, addrList)
	wg.Wait()
}

func Test3(t *testing.T) {
	addrList, err, wg := setupServer(5, 2*time.Second, 10*time.Second)
	xassert.IsNil(t, err)
	setupClient(10, 200, 5, 100*time.Millisecond, addrList)
	wg.Wait()
}

func Test4(t *testing.T) {
	addrList1, err, wg1 := setupServer(3, 2*time.Second, 10*time.Second)
	xassert.IsNil(t, err)
	addrList2, err, wg2 := setupServer(2, 2*time.Second, 30*time.Second)
	xassert.IsNil(t, err)
	setupClient(10, 200, 5, 100*time.Millisecond, append(addrList1, addrList2...))
	wg1.Wait()
	wg2.Wait()
}

// Connection in zero-capacity connection pool is equal to
// original connection effectively.
func TestZeroCapConnPool(t *testing.T) {
	addrList, err, wg := setupServer(1, 0, 10*time.Second)
	xassert.IsNil(t, err)
	setupClient(0, 5, 5, time.Second, addrList)
	wg.Wait()
}

func setupClient(capacity, count, n int, interval time.Duration, addrList []string) error {
	var (
		index   int64
		success int64
		wg      = &sync.WaitGroup{}
		xcp     = New(capacity, func() (net.Conn, error) {
			return net.Dial("tcp", addrList[int(atomic.AddInt64(&index, int64(1)))%len(addrList)])
		})
	)
	defer xcp.Close()

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			if clientHandle(i, xcp) == nil {
				atomic.AddInt64(&success, int64(1))
			}
			wg.Done()
		}(i)
		time.Sleep(interval)
	}

	wg.Wait()
	fmt.Printf("[client] exit [total: %d][success: %d]\n", count, success)
	return nil
}

func clientHandle(i int, xcp *XConnPool) error {
	var (
		err                             error
		data                            []byte
		conn                            net.Conn
		getRetryCount, rawGetRetryCount int
	)

get_retry:
	if getRetryCount > 0 {
		fmt.Printf("[client] get connection retry (%d)\n", getRetryCount)
	}
	if conn, err = xcp.Get(); err != nil {
		fmt.Printf("[client] get connection failed (%s)\n", err)
		if getRetryCount < 3 {
			getRetryCount++
			goto get_retry
		}
		return err
	}

again:
	if err = xpacket.Encode(conn, []byte(fmt.Sprintf("message %d", i))); err != nil {
		fmt.Printf("[client] write message %d failed, release connection (%s)\n", i, err)
		conn.(*XConn).Release()

	raw_get_retry:
		if rawGetRetryCount > 0 {
			fmt.Printf("[client] (raw) get connection retry (%d)\n", rawGetRetryCount)
		}
		if conn, err = xcp.RawGet(); err != nil {
			fmt.Printf("[client] (raw) get connection failed (%s)\n", err)
			if rawGetRetryCount < 3 {
				rawGetRetryCount++
				goto raw_get_retry
			}
			return err
		}
		goto again
	}

	if data, err = xpacket.Decode(conn); err != nil {
		fmt.Printf("[client] read message %d failed, release connection (%s)\n", i, err)
		err = conn.(*XConn).Release()
	} else {
		err = conn.Close()
	}
	fmt.Printf("[client] receive response: %s\n", string(data))

	return err
}

func setupServer(n int, interval, lifetime time.Duration) ([]string, error, *sync.WaitGroup) {
	var (
		addrList = make([]string, 0, n)
		ls       = make([]net.Listener, 0, n)
		wg       = &sync.WaitGroup{}
		i        int
		l        net.Listener
		err      error
	)

	for i := 0; i < n; i++ {
		if l, err = net.Listen("tcp", "0.0.0.0:0"); err != nil {
			return nil, err, nil
		}
		ls = append(ls, l)
		addrList = append(addrList, l.Addr().String())
	}

	for i, l = range ls {
		wg.Add(1)
		go func(l net.Listener, i int) {
			server(l, interval, lifetime, i+1)
			wg.Done()
		}(l, i)
	}

	return addrList, nil, wg
}

func server(l net.Listener, interval, lifetime time.Duration, sid int) {
	var (
		wg          = &sync.WaitGroup{}
		ctx, cancel = context.WithTimeout(context.TODO(), lifetime)
		conn        net.Conn
		err         error
		cid         int
	)

	go func(ctx context.Context) {
		select {
		case <-ctx.Done():
			l.Close()
		}
	}(ctx)

	for {
		if conn, err = l.Accept(); err != nil {
			break
		}
		cid++

		wg.Add(1)
		go func(conn net.Conn, cid int) {
			serverHandle(ctx, conn, sid, cid, interval)
			wg.Done()
		}(conn, cid)
	}

	wg.Wait()
	fmt.Printf("[server %d] timeout exit (%s)\n", sid, lifetime)
	cancel()
}

func serverHandle(ctx context.Context, conn net.Conn, sid, cid int, interval time.Duration) {
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

			time.Sleep(interval)

			back := []byte(fmt.Sprintf("%s [server: %d][connection: %d]", data, sid, cid))

			if err = xpacket.Encode(conn, back); err != nil {
				fmt.Printf("[server: %d][connection: %d] exit (%s)\n", sid, cid, err)
				return
			}

		}
	}
}
