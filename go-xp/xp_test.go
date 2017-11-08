// xp_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-11-06
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-11-08

package xp

import (
	"fmt"
	"github.com/X-Plan/xgo/go-xassert"
	"net"
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
	l, port := freeListener(t)
	wg, errs := &sync.WaitGroup{}, make(chan error, 16)

	wg.Add(1)
	go func() {
		runServer(l, duration, errs)
		wg.Done()
	}()

	time.Sleep(time.Second)

	for i := 0; i < len(values); i += 2 {
		wg.Add(1)
		go func(i int) {
			runClient(port, values[i], values[i+1], errs)
			wg.Done()
		}(i)
	}

	wg.Wait()
	close(errs)
	for err := range errs {
		xassert.IsNil(t, err)
	}
}

func runClient(port string, n int, count int, errs chan error) {
	wg := &sync.WaitGroup{}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			client := &Client{
				Scheduler: dummyScheduler{Addr: "127.0.0.1:" + port},
				PoolSize:  32,
			}

			for j := 0; j < count; j++ {
				// Generate five command pair types, the last two (cmd=3000) are
				// invalid command pair.
				subcmd, cmd := uint32((j%2+1)*1000), uint32(((j/2)%3+1)*1000)
				rsp, err := client.RoundTrip(&Request{
					Head: &Header{Cmd: cmd, SubCmd: subcmd},
					Body: []byte(fmt.Sprintf("[client %d] Tell Me: %d\n", id, j)),
				})

				if err != nil {
					fmt.Printf("[client %d] %s\n", id, err)
				} else if rsp.GetRet().GetCode() != int32(Code_OK) {
					fmt.Printf("[client %d] %s\n", id, rsp.GetRet().GetMsg())
				} else {
					fmt.Printf("[client %d] %s\n", id, string(rsp.GetBody()))
				}

				time.Sleep(time.Second)
			}
		}(i)
	}

	wg.Wait()
	fmt.Printf("[all client n=%d count=%d] Done\n", n, count)
}

func runServer(l net.Listener, duration time.Duration, errs chan error) {
	router := &Router{}
	router.Bind(genHandlerPair(1000, 1000, Code_OK))
	router.Bind(genHandlerPair(1000, 2000, Code_REQUEST_ERROR))
	router.Bind(genHandlerPair(2000, 1000, Code_SERVER_ERROR))
	router.Bind(genHandlerPair(2000, 2000, Code_AUTH_FAILED))

	s := &Server{Handler: router}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		errs <- s.Serve(l)
		wg.Done()
	}()
	time.Sleep(duration)
	errs <- s.Quit()
	wg.Wait()
	fmt.Printf("[server] Done\n")
}

func genHandlerPair(cmd, subcmd uint32, code Code) (uint32, uint32, Handler, AuthHandler) {
	var (
		handler Handler
		auth    AuthHandler
	)

	switch code {
	case Code_OK:
		handler, auth = genHandler(code, "You're OK!"), genAuthHandler("")
	case Code_SERVER_ERROR:
		handler, auth = genHandler(code, "Oh! I'm killed!"), nil
	case Code_REQUEST_ERROR:
		handler, auth = genHandler(code, "Sorry, I don't think you are a good man."), nil
	case Code_AUTH_FAILED:
		handler, auth = genHandler(Code_OK, "Don't come here!"), genAuthHandler("Hey, I have got to prohibit you!")
	}

	return cmd, subcmd, handler, auth
}

func genHandler(code Code, msg string) Handler {
	return HandlerFunc(func(req *Request) (*Response, error) {
		head := req.GetHead()
		prefix := fmt.Sprintf("[cmd: %d][subcmd: %d][seq: %d]: ", head.GetCmd(), head.GetSubCmd(), head.GetSequence())
		if code == Code_OK {
			return &Response{Body: []byte(prefix + msg)}, nil
		} else {
			return nil, fmt.Errorf("%s%s", prefix, msg)
		}
	})
}

func genAuthHandler(msg string) AuthHandler {
	return AuthHandlerFunc(func(head *Header) error {
		prefix := fmt.Sprintf("[cmd: %d][subcmd: %d][seq: %d]: ", head.GetCmd(), head.GetSubCmd(), head.GetSequence())
		if len(msg) > 0 {
			return fmt.Errorf("%s%s", prefix, msg)
		}
		return nil
	})
}

func freeListener(t *testing.T) (net.Listener, string) {
	l, err := net.Listen("tcp", "0.0.0.0:0")
	xassert.IsNil(t, err)
	_, port, err := net.SplitHostPort(l.Addr().String())
	xassert.IsNil(t, err)
	return l, port
}

type dummyScheduler struct {
	Addr string
}

func (ds dummyScheduler) Get() (string, error) {
	return ds.Addr, nil
}

func (ds dummyScheduler) Feedback(string, bool) {}
