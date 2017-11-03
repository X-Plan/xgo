// server.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-11-03
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-11-03

package xp

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/X-Plan/xgo/go-xlog"
	"github.com/X-Plan/xgo/go-xtcpapi"
	"net"
	"regexp"
	"sync"
	"time"
)

type Handler interface {
	Handle(net.Conn, chan int)
}

type Server struct {
	// This field is used by ListenAddr method to satisfy
	// 'xserver.Server' interface, although it looks a bit
	// redundant. When you use 'xserver.Serve' function,
	// you must init it.
	Addr      string
	Handler   Handler       // The key field of this struct, it can't be empty.
	Logger    *xlog.XLogger // This field can be empty, but I recommend you use it.
	TLSConfig *tls.Config   // optional TLS config.

	l          net.Listener
	exit       chan int
	acceptDone chan int
	once       sync.Once
	wg         sync.WaitGroup
	name       string
	timeout    time.Duration // This field is only used in debug.
}

func (s *Server) Serve(l net.Listener) error {
	if s.Handler == nil {
		return fmt.Errorf("handler is invalid")
	}

	if s.TLSConfig != nil {
		l = tls.NewListener(l, s.TLSConfig)
		s.name = "tcp/tls"
	} else {
		s.name = "tcp"
	}

	s.l, s.exit, s.acceptDone = l, make(chan int), make(chan int)
	s.info("start %s server (listen on %s)", s.name, l.Addr())
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
				s.errorf("accept (%s) connection failed (retrying in %v): %s", s.name, delay, err)
				time.Sleep(delay)
				continue
			}

			// Normally, it's caused by closing the listener.
			break outer
		}
		delay = 0

		s.wg.Add(1)
		go func(conn net.Conn) {
			s.Handler.Handle(conn, s.exit)
			s.wg.Done()
		}(conn)
	}

	// Notify 'Quit' method that the accept operation has been done.
	close(s.acceptDone)

	if xtcpapi.IsErrClosing(err) {
		err = nil
	}

	return err
}

func (s *XServer) Quit() (err error) {
	s.once.Do(func() {
		timeout, exitDone := s.timeout, make(chan int)
		if s.l != nil {
			err = s.l.Close()
		}
		<-s.acceptDone

		go func() {
			s.wg.Wait()
			exitDone <- 1
		}()

		close(s.exit)

		if timeout == 0 {
			timeout = time.Minute
		}

		select {
		case <-exitDone:
			s.info("receive exit signal")
		case <-time.After(timeout):
			if err == nil {
				err = errors.New("timeout")
			}
		}

		if err != nil {
			s.errorf("quit %s server: %s", s.name, err)
		} else {
			s.info("quit %s server", s.name)
		}
	})
	return
}

func (s *Server) ListenAddr() string {
	return s.Addr
}

func (s *Server) info(format string, args ...interface{}) {
	if s.Logger != nil {
		s.Logger.Info(format, args...)
	}
}

func (s *Server) errorf(format string, args ...interface{}) {
	if s.Logger != nil {
		s.Logger.Error(format, args...)
	}
}
