// xconnpool.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2016-10-12
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2018-03-30

// go-xconnpool implement a concurrent safe connection pool, this connection
// pool is used to manage and reuse connections. The connection created by
// this pool meets net.Conn interface.
package xconnpool

import (
	"errors"
	"net"
	"sync"
)

// If you operate a closed pool, returns this error.
var ErrClosed = errors.New("XConnPool has been closed")

// If you operate a closed connection, returns this error.
var ErrXConnClosed = errors.New("XConn has been cloesd")

var ErrXConnIsNil = errors.New("XConn is nil")

// XConn is a implementation of net.Conn
type XConn struct {
	net.Conn
	xcp    *XConnPool
	unuse  bool
	closed bool
	mtx    sync.Mutex
}

// Each XConn is associated with a XConnPool, close a XConn is not just
// releasing a connection instead of returning the XConn to the XConnPool.
// If a XConn has been closed, calling Close() will return ErrXConnClosed.
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

// Release the connection directly. If a XConn has been closed, calling
// Release() will return ErrXConnClosed.
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

// Mark a XConn as unused. Release the an unused connection directly when
// you calling Close() function.
func (xc *XConn) Unuse() {
	xc.mtx.Lock()
	xc.unuse = true
	xc.mtx.Unlock()
}

// A factory type to generate connection. The simplest way is wrapping net.Dial().
type Factory func() (net.Conn, error)

// This is an auxiliary type, it's used to transmit more information of error
// to caller.
type GetConnError struct {
	Err error
	// When you create a connection failed, you can store the IP address
	// in this field.
	Addr string
}

func (gce GetConnError) Error() string {
	return gce.Err.Error()
}

// A concurrent safe connection pool type.
type XConnPool struct {
	conns   chan net.Conn
	factory Factory

	// This field is used by 'XConns' to know how many times
	// this pool is used.
	count int64
}

// Create a new pool. capacity parameter represents the capacity of the pool,
// factory parameter is used to generate new connections. If capacity is
// less than zero or factory is nil, returns nil.
func New(capacity int, factory Factory) *XConnPool {
	if capacity < 0 {
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

// Get a connection from the pool. If there is a free connection in the pool,
// returns it directly. Otherwise, creates a new connection by calling the
// factory function.
func (xcp *XConnPool) Get() (net.Conn, error) {
	var (
		err  error
		conn net.Conn
	)

	select {
	case conn = <-xcp.conns:
		// conn is nil only when the pool is closed.
		if conn == nil {
			return nil, ErrClosed
		}
	default:
		if conn, err = xcp.factory(); err != nil {
			return nil, err
		}
	}

	// Using XConn type to wrap the underlying connection.
	return xcp.wrapConn(conn), nil
}

// Get a new connection by calling the factory function, ignore existing free connections.
func (xcp *XConnPool) RawGet() (net.Conn, error) {
	if conn, err := xcp.factory(); err != nil {
		return nil, err
	} else {
		return xcp.wrapConn(conn), nil
	}
}

func (xcp *XConnPool) wrapConn(conn net.Conn) net.Conn {
	xc := &XConn{Conn: conn, xcp: xcp}
	return xc
}

// Put a connection to the pool.
func (xcp *XConnPool) put(conn net.Conn) (err error) {
	// Skipping empty connections.
	if conn == nil {
		err = ErrXConnIsNil
		return
	}

	// Put a connection to the closed pool will generate a panic, so captures
	// it to prevent program crashing, and returns ErrClosed to the caller.
	defer func() {
		if tmpErr := recover(); tmpErr != nil {
			conn.Close()
			err = ErrClosed
		}
	}()

	select {
	case xcp.conns <- conn:
	default:
		err = conn.Close()
	}
	return
}

// Close the connection pool
func (xcp *XConnPool) Close() (err error) {
	// Close a closed pool will generate a panic, so captures it to prevent
	// program crashing, and returns ErrClosed to the caller.
	defer func() {
		if x := recover(); x != nil {
			err = ErrClosed
		}
	}()

	close(xcp.conns)
	for conn := range xcp.conns {
		conn.Close()
	}
	return
}

// Get the number of connections in the pool.
func (xcp *XConnPool) Size() int {
	return len(xcp.conns)
}
