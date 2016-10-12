// xconnpool.go
//
//		Copyright (C), blinklv. All right reserved.
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-10-12
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2016-10-12

// go-xconnpool实现了一个并发安全的连接池, 并且满足
// net.Conn接口. 这个包可以用来管理和重用连接.
package xconnpool

import (
	"errors"
	"net"
)

// 如果连接池已经关闭却还对其进行操作
// 会抛出如下错误.
var ErrClosed = errors.New("XConnPool has been closed")

// XConn是net.Conn的一个具体实现.XConn并不是并
// 发安全的对象, 因此你不应该将XConn置于并发环
// 境下使用.
type XConn struct {
	net.Conn
	xcp   *XConnPool
	unuse bool
}

// 每一个XConn都会与一个特定的XConnPool相关联.
// XConn的关闭操作不是简单的释放连接, 而是有
// 选择的将连接归还到XConnPool进行重用.
func (xc *XConn) Close() error {
	// 如果确定不再使用, 则释放原生连接.
	if xc.unuse {
		if xc.Conn != nil {
			return xc.Conn.Close()
		}
		return nil
	}
	// 将原生连接归还到连接池.
	return xc.xcp.put(xc.Conn)
}

// 将连接标记为不再使用, 在这种情况下XConn的
// Close操作不会再将该连接放回连接池, 而是真
// 的将其释放掉.
func (xc *XConn) Unuse() {
	xc.unuse = true
}

// Factory是一个用来生成连接的工厂函数类型.
// 一个十分简单但是通用的做法就是将net.Dial
// 函数进行一次封装, 使其满足Factory的定义.
type Factory func() (net.Conn, error)

// 连接池类型. 为了满足并发安全, 它在实现上
// 用到了GO的原生Channel类型.
type XConnPool struct {
	conns   chan net.Conn
	factory Factory
}

// 创建一个新的连接池. capacity参数用于指定
// 连接池的最大值, factory函数用于产生新的
// 连接. 调用者应该检测返回值是否为nil.
func New(capacity int, factory Factory) *XConnPool {
	if capacity <= 0 {
		return nil
	}

	xcp := &XConnPool{
		conns:   make(chan net.Conn, capacity),
		factory: factory,
	}

	return xcp
}

// 该函数从连接池中获取连接. 如果连接池
// 中存在空闲连接, 直接返回. 否则调用
// 已经注册的Factory函数创建新的连接.
func (xcp *XConnPool) Get() (net.Conn, error) {
	var (
		err  error
		conn net.Conn
	)

	select {
	case conn = <-xcp.conns:
		// 只有连接池已经关闭的情况下才会直接
		// 返回空值.
		if conn == nil {
			return nil, ErrClosed
		}
	default:
		// 没有空闲的连接, 需要创建新的连接.
		if conn, err = xcp.factory(); err != nil {
			return nil, err
		}
	}

	// 对返回的net.Conn进行一层包装, 替换成XConn.
	// 这一步非常重要, 不然连接关闭的时候会直接
	// 释放掉而不是归还到连接池.
	return xcp.wrapConn(conn), nil
}

// 将net.Conn封装成XConn.
func (xcp *XConnPool) wrapConn(conn net.Conn) net.Conn {
	xc := &XConn{xcp: xcp}
	xc.Conn = conn
	return xc
}

// 将连接放回连接池, 该功能没有直接提供给用户.
// 而是通过XConn的Close函数间接让用户使用.
func (xcp *XConnPool) put(conn net.Conn) (err error) {
	// 空连接直接拒绝.
	if conn == nil {
		err = errors.New("net.Conn is nil")
		return
	}

	// 因为XConnPool在并发环境, 所以在put操作的
	// 时候XConnPool可能已经关闭. 然而写关闭的管道
	// 不像读关闭的管道那么温和, 所抛出的异常
	// 可能直接导致程序crash(程序没有在外围对
	// 异常进行捕获). 因此这里对异常进行捕获, 同时
	// 释放连接,并返回相应的错误信息给调用方.
	defer func() {
		if tmpErr := recover(); tmpErr != nil {
			conn.Close()
			err = ErrClosed
		}
	}()

	select {
	// 连接池未关闭且存在空闲空间, 则归还成功.
	case xcp.conns <- conn:
	default:
		// 连接池未关闭但是无空闲空间, 则直接释放
		// 连接.
		err = conn.Close()
	}
	return
}

// 关闭连接池.
func (xcp *XConnPool) Close() (err error) {
	// 这里同put函数中所做的事情一样, 关闭
	// 一个已经关闭的管道可能导致程序crash.
	// 因此这里将该异常捕获.
	defer func() {
		if x := recover(); x != nil {
			err = ErrClosed
		}
	}()

	// 首先关闭管道, 这样阻止了新的连接进入
	// 连接池. 然后依次关闭连接池中的残余
	// 连接.
	close(xcp.conns)
	for conn := range xcp.conns {
		conn.Close()
	}
	return
}

// 获取连接池中连接的数目
func (xcp *XConnPool) Size() int {
	return len(xcp.conns)
}
