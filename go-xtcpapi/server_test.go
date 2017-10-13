// server_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-10-13
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-10-13

package xtcpapi

import (
	"fmt"
	"github.com/X-Plan/xgo/go-xassert"
	"github.com/X-Plan/xgo/go-xlog"
	"github.com/X-Plan/xgo/go-xpacket"
	"io"
	"io/ioutil"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

func TestZeroClient(t *testing.T) {
	testServer(t, 3*time.Second, 0, 10)
}

func TestZeroMsgClient(t *testing.T) {
	testServer(t, 3*time.Second, 10, 0)
}

func TestServerCloseOneClient(t *testing.T) {
	testServer(t, 5*time.Second, 1, 20)
}

func TestClientCloseOneClient(t *testing.T) {
	testServer(t, 10*time.Second, 1, 3)
}

func TestServerCloseMultiClient(t *testing.T) {
	testServer(t, 5*time.Second, 10, 20)
}

func TestClientCloseMultiClient(t *testing.T) {
	testServer(t, 10*time.Second, 10, 3)
}

func TestMixCloseMultiClient(t *testing.T) {
	testServer(t, 10*time.Second, 10, 3, 10, 20)
}

func testServer(t *testing.T, duration time.Duration, values ...int) {
	var (
		wg    = &sync.WaitGroup{}
		errch = make(chan error, 16)
		port  string
		l     net.Listener
	)

	l, port = getFreeListener(t)

	wg.Add(1)
	go func() {
		runEchoServer(l, duration, errch)
		wg.Done()
	}()

	time.Sleep(time.Second)

	for i := 0; i < len(values); i += 2 {
		wg.Add(1)
		go func(i int) {
			runEchoClient(port, values[i], values[i+1], errch)
			wg.Done()
		}(i)
	}

	wg.Wait()
	close(errch)
	for err := range errch {
		xassert.IsNil(t, err)
	}
}

func getFreeListener(t *testing.T) (net.Listener, string) {
	l, err := net.Listen("tcp", "0.0.0.0:0")
	xassert.IsNil(t, err)
	_, port, err := net.SplitHostPort(l.Addr().String())
	xassert.IsNil(t, err)
	return l, port
}

func runEchoClient(port string, n int, count int, errch chan error) {
	var (
		wg = &sync.WaitGroup{}
	)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			var (
				err                     error
				conn                    net.Conn
				sendCount, receiveCount int
			)

			if conn, err = net.Dial("tcp", "127.0.0.1:"+port); err != nil {
				errch <- err
				return
			}
			defer conn.Close()

			for j := 0; j < count; j++ {
				data := []byte(fmt.Sprintf("[client %d] Tell Me: %d\n", id, j))
				if err = xpacket.Encode(conn, data); err != nil {
					break
				}
				fmt.Printf("[send]%s", string(data))
				sendCount++

				if data, err = xpacket.Decode(conn); err != nil {
					break
				}
				fmt.Printf("[receive]%s", string(data))
				receiveCount++

				time.Sleep(time.Second)
			}

			if !(sendCount == receiveCount || sendCount == receiveCount+1) {
				errch <- fmt.Errorf("send count (%d) not match receive count (%d)")
			}

			if err == nil {
				fmt.Printf("[client %d] Done\n", id)
			} else {
				fmt.Printf("[client %d] Done: %s\n", id, err)
			}
			return

		}(i)
	}

	wg.Wait()
	fmt.Printf("[all client n=%d count=%d] Done\n", n, count)
}

func runEchoServer(l net.Listener, duration time.Duration, errch chan error) {
	var (
		err    error
		logdir string
		xl     *xlog.XLogger
	)

	if logdir, err = ioutil.TempDir("/tmp", "tcpapi"); err != nil {
		errch <- err
		return
	}
	defer os.RemoveAll(logdir)

	if xl, err = xlog.New(&xlog.XConfig{Dir: logdir, Level: xlog.DEBUG}); err != nil {
		errch <- err
		return
	}
	defer xl.Close()

	s := &Server{
		Handler: &echoHandler{errch: errch},
		Logger:  xl,
		timeout: 20 * time.Second,
	}

	go func() { errch <- s.Serve(l) }()
	time.Sleep(duration)
	errch <- s.Quit()
	fmt.Printf("[server] Done\n")
}

type echoHandler struct {
	errch chan error
}

func (eh *echoHandler) Handle(conn net.Conn, exit chan int) {
	esc := &echoServerConn{
		conn:     conn,
		datach:   make(chan []byte, 128),
		readDone: make(chan int, 1),
		errch:    eh.errch,
		exit:     exit,
	}

	go esc.read()
	esc.write()
	conn.Close()
}

type echoServerConn struct {
	conn     net.Conn
	datach   chan []byte
	readDone chan int
	errch    chan error
	exit     chan int
}

func (esc *echoServerConn) read() {
	var (
		err  error
		data []byte
	)

	for {
		if data, err = xpacket.Decode(esc.conn); err != nil {
			if err == io.EOF {
				esc.readDone <- 1
				fmt.Printf("[server] Read Done: %s\n", err)
			} else {
				esc.errch <- err
			}
			return
		}
		esc.datach <- data
	}
}

func (esc *echoServerConn) write() {
	var (
		data []byte
		err  error
	)

outer:
	for {
		select {
		case data = <-esc.datach:
			if err = xpacket.Encode(esc.conn, data); err != nil {
				fmt.Printf("[server] Send Message Failed: %s\n", err)
				break outer
			}
			time.Sleep(2 * time.Second)

		case <-esc.exit:
			fmt.Printf("[server] Receive Exit Signal\n")
			break outer
		}
	}

	esc.conn.(*net.TCPConn).CloseRead()
	<-esc.readDone

	close(esc.datach)
	if len(esc.datach) > 0 {
		fmt.Printf("[server] Clean Message\n")
		for data = range esc.datach {
			if err = xpacket.Encode(esc.conn, data); err != nil {
				fmt.Printf("[server] Clean Message Failed (%s): %s\n", string(data), err)
			}
		}
	}

	return
}
