// xconnpool.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-10-12
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-02-06

// go-xconnpool实现了一个并发安全的连接池, 该连接池
// 可以用来管理和重用连接, 由该连接池产生的连接满足
// net.Conn接口规范.
package xconnpool

import (
	"errors"
	"net"
	"sync"
)

// 版本信息
var Version = "1.2.1"

// 如果连接池已经关闭却还对其进行操作
// 会抛出该错误.
var ErrClosed = errors.New("XConnPool has been closed")

var ErrXConnClosed = errors.New("XConn has been cloesd")

var ErrXConnIsNil = errors.New("XConn is nil")

// XConn是net.Conn的一个具体实现.
type XConn struct {
	net.Conn
	xcp    *XConnPool
	unuse  bool
	closed bool
	mtx    sync.Mutex
}

// 每一个XConn都会与一个特定的XConnPool相关联.
// XConn的关闭操作不是简单的释放连接, 而是有
// 选择的将连接归还到XConnPool进行重用. 对于
// 已经成功关闭的连接再次调用Close(), Release()
// 函数都会返回ErrXConnClosed.
func (xc *XConn) Close() error {
	xc.mtx.Lock()
	defer xc.mtx.Unlock()

	if !xc.closed {
		if xc.unuse {
			if err := xc.Conn.Close(); err != nil {
				return err
			}
		} else if err := xc.xcp.put(xc.Conn); err != nil {
			return err
		}
		xc.closed = true
		return nil
	}

	return ErrXConnClosed
}

// 绕过连接池, 直接释放连接. 对于已经成功释放的连接再次
// 调用Close(), Release()函数都会返回ErrXConnClosed.
func (xc *XConn) Release() error {
	xc.mtx.Lock()
	defer xc.mtx.Unlock()

	if !xc.closed {
		if err := xc.Conn.Close(); err != nil {
			return err
		}
		xc.closed = true
		return nil
	}
	return ErrXConnClosed
}

// 将连接标记为不再使用, 在这种情况下XConn的
// Close操作不会再将该连接放回连接池, 而是真
// 的将其释放掉. 因为连接池本身的意义就是避免
// 连接的过度创建与释放, 并且多余的连接会在
// Close中被释放掉, 所以大部分情况下你不会使
// 用该函数. 但是当连接不可用的时候, 你可以
// 调用该函数, 然后再Close. 当然直接调用Release
// 是一个更好的选择.
func (xc *XConn) Unuse() {
	xc.mtx.Lock()
	xc.unuse = true
	xc.mtx.Unlock()
}

// Factory是一个用来生成连接的工厂函数类型.
// 一个十分简单但是通用的做法就是将net.Dial
// 函数进行一次封装, 使其满足Factory的定义.
type Factory func() (net.Conn, error)

// 这是一个辅助类型, 它用于Factory在创建新连接
// 失败后可以向外传递更多的信息.
// NOTE: 不强制要求Factory一定使用该类型对error
// 进行包装.
type GetConnError struct {
	Err error
	// 当建立连接失败后可以将欲建立连接的ip地址
	// 填写到该字段. 用于向调用方反馈存在问题的
	// ip地址.
	Addr string
}

func (gce GetConnError) Error() string {
	return gce.Err.Error()
}

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

	if factory == nil {
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

// 绕过已有的连接池, 直接调用factory创建新的连接.
func (xcp *XConnPool) RawGet() (net.Conn, error) {
	if conn, err := xcp.factory(); err != nil {
		return nil, err
	} else {
		return xcp.wrapConn(conn), nil
	}
}

// 将net.Conn封装成XConn.
func (xcp *XConnPool) wrapConn(conn net.Conn) net.Conn {
	xc := &XConn{Conn: conn, xcp: xcp}
	return xc
}

// 将连接放回连接池, 该功能没有直接提供给用户.
// 而是通过XConn的Close函数间接让用户使用.
func (xcp *XConnPool) put(conn net.Conn) (err error) {
	// 空连接直接拒绝.
	if conn == nil {
		err = ErrXConnIsNil
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
