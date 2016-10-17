// xconnpool_test.go
//
//		Copyright (C), blinklv. All rights reserved.
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-10-15
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2016-10-18
package xconnpool

import (
	"github.com/X-Plan/xgo/go-xassert"
	"net"
	"testing"
)

// 测试对panic的捕获.
func TestPanic(t *testing.T) {
	// 需要先开启server
	xcp := New(10, func() (net.Conn, error) {
		return net.Dial("tcp", "127.0.0.1:8000")
	})
	xassert.NotNil(t, xcp)

	// 测试XConn的重复关闭.
	conn, err := xcp.Get()
	xassert.IsNil(t, err)

	xassert.IsNil(t, conn.Close())
	xassert.Equal(t, ErrXConnIsNil, conn.Close())

	conn, err = xcp.Get()
	xassert.IsNil(t, err)
	xc := conn.(*XConn)
	xc.Unuse()

	xassert.IsNil(t, conn.Close())
	xassert.Equal(t, ErrXConnIsNil, conn.Close())

	// 测试XConnPool的重复关闭.
	conn, err = xcp.Get()
	xassert.IsNil(t, err)

	// 关闭连接池.
	err = xcp.Close()
	xassert.IsNil(t, err)

	// 如下的两个操作都会触发
	// panic, 但是现在转换为
	// 错误返回.

	// 重复关闭.
	err = xcp.Close()
	xassert.Equal(t, ErrClosed, err)

	// 归还连接到已经关闭的连接池.
	err = conn.Close()
	xassert.Equal(t, ErrClosed, err)
}
