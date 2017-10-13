// tcp_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-10-13
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-10-13
package xtcpapi

import (
	"github.com/X-Plan/xgo/go-xassert"
	"io/ioutil"
	"net"
	"os"
	"syscall"
	"testing"
)

func TestInheritEmptyTcp(t *testing.T) {
	var tcp = &TCP{}
	os.Setenv(EnvNumber, "")
	xassert.IsNil(t, tcp.inherit())
}

func TestInheritZeroEnvNumber(t *testing.T) {
	var tcp = &TCP{}
	os.Setenv(EnvNumber, "0")
	xassert.IsNil(t, tcp.inherit())
}

func TestInheritInvalidEnvNumberFirst(t *testing.T) {
	var tcp = &TCP{}
	os.Setenv(EnvNumber, "x")
	xassert.Match(t, tcp.inherit(), `^invalid environment variable`)
}

func TestInheritInvalidSocketFirst(t *testing.T) {
	var tcp = &TCP{}
	os.Setenv(EnvNumber, "10")
	xassert.Match(t, tcp.inherit(), `^inheriting invalid socket fd`)
}

func TestInheritInvalidSocketSecond(t *testing.T) {
	var tcp = &TCP{}
	os.Setenv(EnvNumber, "1")
	tempfile, err := ioutil.TempFile("", "test")
	xassert.IsNil(t, err)
	defer os.Remove(tempfile.Name())
	tcp.start = dup(t, int(tempfile.Fd()))
	xassert.Match(t, tcp.inherit(), `^inheriting invalid socket fd`)
}

func TestInheritInvalidTcpListener(t *testing.T) {
	var tcp = &TCP{}
	os.Setenv(EnvNumber, "1")
	l, err := net.Listen("unix", "/tmp/foo.sock")
	xassert.IsNil(t, err)
	file, err := l.(*net.UnixListener).File()
	xassert.IsNil(t, err)
	xassert.IsNil(t, l.Close())
	tcp.start = dup(t, int(file.Fd()))
	xassert.Match(t, tcp.inherit(), `^invalid tcp listener$`)
}

func TestInheritOk(t *testing.T) {
	var tcp = &TCP{}
	os.Setenv(EnvNumber, "1")
	l, err := net.Listen("tcp", ":0")
	xassert.IsNil(t, err)
	file, err := l.(*net.TCPListener).File()
	xassert.IsNil(t, err)
	xassert.IsNil(t, l.Close())
	tcp.start = dup(t, int(file.Fd()))
	xassert.IsNil(t, tcp.inherit())
}

func TestListenNetTypeNotMatch(t *testing.T) {
	var (
		tcp        = &TCP{}
		laddr, err = net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	)
	xassert.IsNil(t, err)
	_, err = tcp.Listen("unix", laddr)
	xassert.NotNil(t, err)

	laddr, err = net.ResolveTCPAddr("tcp6", "[::]:0")
	xassert.IsNil(t, err)
	_, err = tcp.Listen("tcp4", laddr)
	xassert.NotNil(t, err)
}

// The follow test depends on the low-level implementation.
func TestListen(t *testing.T) {
	var (
		err                    error
		tcp                    = &TCP{}
		laddr1, laddr2, laddr3 *net.TCPAddr
		file                   *os.File
		l1, l2, l3             *net.TCPListener
	)

	laddr1, err = net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	xassert.IsNil(t, err)
	laddr2, err = net.ResolveTCPAddr("tcp4", "0.0.0.0:0")
	xassert.IsNil(t, err)
	laddr3, err = net.ResolveTCPAddr("tcp6", "[::]:0")
	xassert.IsNil(t, err)

	l1, err = net.ListenTCP("tcp", laddr1)
	xassert.IsNil(t, err)
	l2, err = net.ListenTCP("tcp4", laddr2)
	xassert.IsNil(t, err)
	l3, err = net.ListenTCP("tcp6", laddr3)
	xassert.IsNil(t, err)

	file, err = l1.File()
	xassert.IsNil(t, err)
	_, err = l2.File()
	xassert.IsNil(t, err)
	_, err = l3.File()
	xassert.IsNil(t, err)

	tcp.start = int(file.Fd())
	os.Setenv(EnvNumber, "3")

	_, err = tcp.Listen("tcp", laddr1)
	xassert.IsNil(t, err)
	_, err = tcp.Listen("tcp4", laddr2)
	xassert.IsNil(t, err)
	_, err = tcp.Listen("tcp6", laddr3)
	xassert.IsNil(t, err)
}

func dup(t *testing.T, fd int) int {
	nfd, err := syscall.Dup(fd)
	xassert.IsNil(t, err)
	return nfd
}
