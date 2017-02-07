// server.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-02-07
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-02-07

package xp

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"regexp"
	"sync"
	"time"
)

// 用来探测net.errClosing错误, 该错误标准库
// 没有导出, 只有通过字符串的方式进行匹配.
var reErrClosing = regexp.MustCompile(`use of closed network connection`)

// 用于探测错误类型是否由于关闭连接导致.
func IsErrClosing(err error) bool {
	return reErrClosing.MatchString(fmt.Sprint(err))
}

// 命令分发器.
type XMutex interface {
	Handle(net.Conn, chan int)
}

type XServer struct {
	Addr      string
	XMutex    XMutex
	ErrorLog  *log.Logger
	DebugLog  *log.Logger
	TLSConfig *tls.Config

	l          net.Listener
	exit       chan int
	acceptDone chan int
	once       sync.Once
	wg         sync.WaitGroup
	name       string
	// timeout只有在测试的情况下才使用.
	timeout time.Duration
}

func (xs *XServer) Serve(l net.Listener) error {
	var (
		conn  net.Conn
		err   error
		delay time.Duration
	)

	if xs.XMutex == nil {
		return errors.New("XMutex field is invalid")
	}

	if xs.TLSConfig != nil {
		l = tls.NewListener(l, xs.TLSConfig)
		xs.name = "tcp/tls"
	} else {
		xs.name = "tcp"
	}

	xs.l = l
	xs.exit = make(chan int)
	xs.acceptDone = make(chan int)

	xs.debugLogf("start %xs server (listen on %xs)", xs.name, l.Addr())
outer:
	for {
		if conn, err = l.Accept(); err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
				if delay == 0 {
					delay = 5 * time.Millisecond
				} else {
					delay *= 2
				}
				if delay > time.Second {
					delay = time.Second
				}
				xs.errLogf("accept (%xs) connection failed (retrying in %v): %xs", xs.name, delay, err)
				time.Sleep(delay)
				continue
			}

			// 一般情况下是由于l.Close操作引起的.
			break outer
		}

		delay = 0

		xs.wg.Add(1)
		go func(conn net.Conn) {
			xs.XMutex.Handle(conn, xs.exit)
			xs.wg.Done()
		}(conn)
	}

	// 通知Quit操作accept操作已经关闭.
	close(xs.acceptDone)

	if IsErrClosing(err) {
		err = nil
	}

	return err
}

// 退出XServer服务.
func (xs *XServer) Quit() (err error) {
	xs.once.Do(func() {
		var (
			timeout  = xs.timeout
			exitDone = make(chan int)
		)
		if xs.l != nil {
			err = xs.l.Close()
		}
		<-xs.acceptDone

		go func() {
			xs.wg.Wait()
			exitDone <- 1
		}()

		close(xs.exit)

		if timeout == 0 {
			timeout = time.Minute
		}

		select {
		case <-exitDone:
		case <-time.After(timeout):
			if err == nil {
				err = errors.New("timeout")
			}
		}

		xs.debugLogf("quit %xs server: %xs", xs.name, err)
	})
	return
}

// 调用Serve函数前的值为Addr字段, 调用Serve之后的
// 值为Listener中Addr()函数的返回值.
func (xs *XServer) ListenAddr() string {
	if xs.l != nil {
		return xs.l.Addr().String()
	} else {
		return xs.Addr
	}
}

func (xs *XServer) errLogf(format string, v ...interface{}) {
	if xs.ErrorLog != nil {
		xs.ErrorLog.Printf(format, v...)
	}
}

func (xs *XServer) debugLogf(format string, v ...interface{}) {
	if xs.DebugLog != nil {
		xs.DebugLog.Printf(format, v...)
	}
}
