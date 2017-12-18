// xconnpools_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-12-15
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-12-18

package xconnpool

import (
	"fmt"
	"github.com/X-Plan/xgo/go-xassert"
	"github.com/X-Plan/xgo/go-xsched"
	"net"
	"testing"
	"time"
)

var conns = make(map[string]int)

type debugConn struct {
	address string // remote address.
}

func (dc debugConn) Read(b []byte) (n int, err error) {
	return len(b), nil
}

func (dc debugConn) Write(b []byte) (n int, err error) {
	return len(b), nil
}

func (dc debugConn) Close() error {
	fmt.Printf("release connection (%s)\n", dc.address)
	return nil
}

func (dc debugConn) LocalAddr() net.Addr {
	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:8181")
	return addr
}

func (dc debugConn) RemoteAddr() net.Addr {
	addr, _ := net.ResolveTCPAddr("tcp", dc.address)
	return addr
}

func (dc debugConn) SetDeadline(t time.Time) error {
	return nil
}

func (dc debugConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (dc debugConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func dial(_, address string) (net.Conn, error) {
	fmt.Printf("create connection (%s)\n", address)
	return debugConn{address: address}, nil
}

func TestClean(t *testing.T) {
	cleanPeriod = time.Minute
	addrs := []string{
		"192.168.1.1:100:20",
		"192.168.1.1:200:30",
		"192.168.1.1:300:40",
		"192.168.1.1:400:10",
		"192.168.1.1:500:20",
		"192.168.1.1:600:50",
		"192.168.1.1:700:10",
		"192.168.1.1:800:20",
	}
	scheduler, err := xsched.New(addrs)
	xassert.IsNil(t, err)

	xcps := NewXConnPools(32, scheduler, dial)
	xassert.NotNil(t, xcps)

	for i := 0; i < 10; i++ {
		go func() {
			for {
				conn, err := xcps.Get()
				if err != nil {
					fmt.Printf("get connection failed (%s)\n", err)
					break
				}
				time.Sleep(time.Second)
				scheduler.Feedback(conn.RemoteAddr().String(), true)
				conn.Close()
			}
		}()
	}

	for _, addr := range addrs {
		scheduler.Remove(addr)
		time.Sleep(time.Minute)
	}

	xassert.IsNil(t, xcps.Close())
}
