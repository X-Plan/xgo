// xserver.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-10-13
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-10-16

// This package is used to manage the shutdown and restart of servers.
package xserver

import (
	"fmt"
	"github.com/X-Plan/xgo/go-xtcpapi"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	isInherit = os.Getenv(xtcpapi.EnvNumber) != ""
	ppid      = os.Getppid()
)

type Server interface {
	ListenAddr() string
	Serve(net.Listener) error
	Quit() error
}

// This function will run some servers and monitor specific signals.
// SIGTERM and SIGINT are used to exit, SIGUSR2 to restart.
func Serve(servers ...Server) error {

	var (
		tcp  = &xtcpapi.TCP{}
		errs = make(chan error, 2*len(servers)+1)
		ls   = make([]net.Listener, 0, len(servers))
	)

	for _, s := range servers {
		addr, err := net.ResolveTCPAddr("tcp", s.ListenAddr())
		if err != nil {
			return err
		}

		l, err := tcp.Listen("tcp", addr)
		if err != nil {
			return err
		}

		ls = append(ls, l)
	}

	done := make(chan int)
	go notify(servers, tcp, done, errs)

	for i, s := range servers {
		go serve(s, ls[i], errs)
	}

	if isInherit && ppid != 1 {
		if err := syscall.Kill(ppid, syscall.SIGTERM); err != nil {
			return fmt.Errorf("closing parrent process failed: %s", err)
		}
	}

	select {
	case err := <-errs:
		return err
	case <-done:
	}

	return nil
}

func serve(s Server, l net.Listener, errs chan error) {
	if err := s.Serve(l); err != nil {
		errs <- err
	}
}

func notify(servers []Server, tcp *xtcpapi.TCP, done chan int, errs chan error) {
	var (
		sigch = make(chan os.Signal, 16)
		wg    = &sync.WaitGroup{}
	)

	signal.Notify(sigch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR2)
outer:
	for {
		sig := <-sigch
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			signal.Stop(sigch)

			for _, s := range servers {
				wg.Add(1)
				go func(s Server) {
					if err := s.Quit(); err != nil {
						errs <- err
					}
					wg.Done()
				}(s)
			}
			wg.Wait()

			// notify exiting to the main routine.
			close(done)
			break outer

		case syscall.SIGUSR2:
			_, err := tcp.StartProcess()
			errs <- err
			if err != nil {
				break outer
			}
		}
	}

	return
}
