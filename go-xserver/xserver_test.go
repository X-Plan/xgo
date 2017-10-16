// xserver_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-10-16
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-10-16
package xserver

import (
	"github.com/X-Plan/xgo/go-xassert"
	"github.com/X-Plan/xgo/go-xtcpapi"
	"net"
	"os"
	"syscall"
	"testing"
	"time"
)

type dummyServer struct {
	Addr string
	l    net.Listener
}

func (ds *dummyServer) ListenAddr() string {
	return ds.Addr
}

func (ds *dummyServer) Serve(l net.Listener) error {
	ds.l = l
	_, err := l.Accept()
	if xtcpapi.IsErrClosing(err) {
		err = nil
	}
	return err
}

func (ds *dummyServer) Quit() error {
	return ds.l.Close()
}

func TestEmptyArgument(t *testing.T) {
	xassert.IsNil(t, testArgument(t, true))
}

func TestOneCorrectArgument(t *testing.T) {
	xassert.IsNil(t, testArgument(t, true, &dummyServer{Addr: "0.0.0.0:0"}))
}

func TestOneErrorArgument(t *testing.T) {
	xassert.Match(t, testArgument(t, false, &dummyServer{Addr: "0.0.0.0:-1"}), `invalid port`)
	xassert.Match(t, testArgument(t, false, &dummyServer{Addr: "0.0.0.0:80"}), `permission denied`)
}

func TestMultiErrorArgument(t *testing.T) {
	xassert.Match(t, testArgument(t, false,
		&dummyServer{Addr: "0.0.0.0:-1"},
		&dummyServer{Addr: "0,0.0.0:80"},
		&dummyServer{Addr: "0.0.0:100"},
		&dummyServer{Addr: "0.0.0:100"},
	), `invalid port`)
}

func TestMultiCorrectArgument(t *testing.T) {
	xassert.IsNil(t, testArgument(t, true,
		&dummyServer{Addr: "0.0.0.0:0"},
		&dummyServer{Addr: "0.0.0.0:0"},
		&dummyServer{Addr: "0.0.0.0:0"},
		&dummyServer{Addr: "0.0.0.0:0"},
		&dummyServer{Addr: "0.0.0.0:0"}))
}

func TestMixArgument(t *testing.T) {
	xassert.Match(t, testArgument(t, false,
		&dummyServer{Addr: "0.0.0.0:0"},
		&dummyServer{Addr: "0.0.0.0:80"},
		&dummyServer{Addr: "0.0.0.0:0"},
		&dummyServer{Addr: "0.0.0.0:-1"},
	), `permission denied`)
}

func testArgument(t *testing.T, flag bool, servers ...Server) error {
	var (
		errch = make(chan error)
	)
	go func() {
		errch <- Serve(servers...)
	}()

	if flag {
		time.Sleep(3 * time.Second)
		xassert.IsNil(t, syscall.Kill(os.Getpid(), syscall.SIGTERM))
	}

	return <-errch
}
