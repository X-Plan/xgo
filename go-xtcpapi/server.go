// server.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-10-13
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-10-25

package xtcpapi

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/X-Plan/xgo/go-xlog"
	"net"
	"regexp"
	"runtime/debug"
	"sync"
	"time"
)

// This regexp is is used to detect net.errClosing error, because this
// error is hidden, so I can only use string matching.
var reErrClosing = regexp.MustCompile(`use of closed network connection`)

func IsErrClosing(err error) bool {
	return reErrClosing.MatchString(fmt.Sprint(err))
}

// If user wants to handle connections, the method should be satisfy this
// interface. The exit field is used to notify a user that the server exits.
type Handler interface {
	Handle(conn net.Conn, exit chan int)
}

type Server struct {
	Addr      string
	Handler   Handler
	Logger    *xlog.XLogger
	TLSConfig *tls.Config

	l          net.Listener
	exit       chan int
	acceptDone chan int
	once       sync.Once
	wg         sync.WaitGroup
	name       string
	timeout    time.Duration // this field only used in test.
}

func (s *Server) Serve(l net.Listener) error {
	if s.TLSConfig != nil {
		l = tls.NewListener(l, s.TLSConfig)
		s.name = "tcp/tls"
	} else {
		s.name = "tcp"
	}

	s.l, s.exit, s.acceptDone = l, make(chan int), make(chan int)
	s.Logger.Info("start %s server (listen on %s)", s.name, l.Addr())

	var (
		err   error
		conn  net.Conn
		delay time.Duration
	)
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
				s.Logger.Error("accept (%s) connection failed (retrying in %v): %s", s.name, delay, err)
				time.Sleep(delay)
				continue
			}

			// Normally, this error is caused by closing connection.
			break outer
		}

		delay = 0

		s.wg.Add(1)
		go func(conn net.Conn) {
			defer func() {
				if x := recover(); x != nil {
					s.Logger.Fatal("(panic) %s: %s", x, debug.Stack())
				}
				s.wg.Done()
			}()
			s.Handler.Handle(conn, s.exit)
		}(conn)
	}

	// Notify 'Quit' function that the accept operation has been done.
	close(s.acceptDone)

	if IsErrClosing(err) {
		err = nil
	}

	return err
}

func (s *Server) Quit() (err error) {
	s.once.Do(func() {
		var (
			timeout  = s.timeout
			exitDone = make(chan int)
		)
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
		case <-time.After(timeout):
			if err == nil {
				err = errors.New("timeout")
			}
		}

		if err == nil {
			s.Logger.Info("quit %s server", s.name)
		} else {
			s.Logger.Error("quit %s server: %s", s.name, err)
		}
	})
	return
}

func (s *Server) ListenAddr() string {
	return s.Addr
}
