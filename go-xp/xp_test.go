// xp_test.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-02-07
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-02-10

package xp

import (
	"fmt"
	"github.com/X-Plan/xgo/go-xassert"
	"github.com/X-Plan/xgo/go-xdebug"
	"github.com/X-Plan/xgo/go-xlog"
	"log"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

// 无客户端.
func TestZeroClient(t *testing.T) {
	testServer(t, 3*time.Second, 0, 10)
}

// 存在客户端, 但是客户端不发送消息.
func TestZeroMsgClient(t *testing.T) {
	testServer(t, 3*time.Second, 10, 0)
}

// 一个客户端, 服务端优先关闭.
func TestServerCloseOneClient(t *testing.T) {
	testServer(t, 5*time.Second, 1, 20)
}

// 一个客户端, 客户端优先关闭.
func TestClientCloseOneClient(t *testing.T) {
	testServer(t, 10*time.Second, 1, 3)
}

// 多个客户端, 服务端优先关闭.
func TestServerCloseMultiClient(t *testing.T) {
	testServer(t, 5*time.Second, 10, 20)
}

// 多个客户端, 客户端优先关闭.
func TestClientCloseMultiClient(t *testing.T) {
	testServer(t, 10*time.Second, 10, 3)
}

// 多个客户端, 混合式关闭.
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
		runServer(l, duration, errch)
		wg.Done()
	}()

	time.Sleep(time.Second)

	for i := 0; i < len(values); i += 2 {
		wg.Add(1)
		go func(i int) {
			runClient(port, values[i], values[i+1], errch)
			wg.Done()
		}(i)
	}

	wg.Wait()
	close(errch)
	for err := range errch {
		xassert.IsNil(t, err)
	}
}

type dummyScheduler struct {
	Addr string
}

func (ds dummyScheduler) Get() (string, error) {
	return ds.Addr, nil
}

func (ds dummyScheduler) Feedback(string, bool) {}

func getFreeListener(t *testing.T) (net.Listener, string) {
	l, err := net.Listen("tcp", "0.0.0.0:0")
	xassert.IsNil(t, err)
	_, port, err := net.SplitHostPort(l.Addr().String())
	xassert.IsNil(t, err)
	return l, port
}

func runClient(port string, n int, count int, errch chan error) {
	var (
		wg = &sync.WaitGroup{}
	)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			xcli, err := NewXClient(&XClientConfig{
				RetryCount: 3,
				Interval:   100 * time.Millisecond,
				Scheduler:  dummyScheduler{Addr: "127.0.0.1:" + port},
			})

			if err != nil {
				errch <- err
				return
			}

			var cmd, subcmd uint32

			for j := 0; j < count; j++ {
				switch j % 5 {
				case 0:
					cmd, subcmd = uint32(1000), uint32(1000)
				case 1:
					cmd, subcmd = uint32(1000), uint32(2000)
				case 2:
					cmd, subcmd = uint32(2000), uint32(1000)
				case 3:
					cmd, subcmd = uint32(2000), uint32(2000)
				case 4:
					cmd, subcmd = uint32(3000), uint32(1000)
				}

				rsp, err := xcli.Send(&Request{
					Head: &Header{
						Sequence: uint64(j),
						Cmd:      cmd,
						SubCmd:   subcmd,
					},
					Body: []byte(fmt.Sprintf("[client %d] Tell Me: %d\n", id, j)),
				})

				if err != nil {
					fmt.Printf("[client %d] %s\n", id, err)
				} else if rsp.GetRet().GetCode() != int32(EnumRetCode_OK) {
					fmt.Printf("[client %d] %s\n", id, rsp.GetRet().GetMsg())
				} else {
					fmt.Printf("[client %d] %s\n", id, string(rsp.GetBody()))
				}
			}

		}(i)
	}

	wg.Wait()
	fmt.Printf("[all client n=%d count=%d] Done\n", n, count)
}

func runServer(l net.Listener, duration time.Duration, errch chan error) {
	xmtx := NewXRouter()
	xmtx.ErrorLog = log.New(xlog.ErrorWrapper{nil}, "", 0)
	xmtx.Register(createXHandlerPair(1000, 1000, 0))
	xmtx.Register(createXHandlerPair(1000, 2000, -1))
	xmtx.Register(createXHandlerPair(2000, 1000, -2))
	xmtx.Register(createXHandlerPair(2000, 2000, -3))

	xs := &XServer{
		XMutex: xmtx,
		XD:     xdebug.New("server", os.Stderr),
	}

	go func() { errch <- xs.Serve(l) }()
	time.Sleep(duration)
	errch <- xs.Quit()
	fmt.Printf("[server] Done\n")
}

func createXHandlerPair(cmd, subcmd uint32, code int32) (uint32, uint32, XHandler, XAuthHandler) {
	var (
		op   XHandler
		auth XAuthHandler
	)

	switch EnumRetCode(code) {
	case EnumRetCode_OK:
		op = XHandlerFunc(func(req *Request) (*Response, error) {
			return &Response{
				Body: []byte(fmt.Sprintf("[cmd: %d][subcmd: %d][seq: %d]: You're OK!", cmd, subcmd, req.GetHead().GetSequence())),
			}, nil
		})
		auth = XAuthHandlerFunc(func(head *Header) error {
			return nil
		})
	case EnumRetCode_SERVER_ERROR:
		op = XHandlerFunc(func(req *Request) (*Response, error) {
			return nil, fmt.Errorf("[cmd: %d][subcmd: %d][seq: %d]: I'm killed.", cmd, subcmd, req.GetHead().GetSequence())
		})
	case EnumRetCode_REQUEST_ERROR:
		op = XHandlerFunc(func(req *Request) (*Response, error) {
			return &Response{
				Ret: &Return{
					Code: int32(EnumRetCode_REQUEST_ERROR),
					Msg:  fmt.Sprintf("[cmd: %d][subcmd: %d][seq: %d]: Sorry, I think you're not a good man.", cmd, subcmd, req.GetHead().GetSequence()),
				},
			}, nil
		})
	case EnumRetCode_AUTH_FAILED:
		op = XHandlerFunc(func(req *Request) (*Response, error) {
			return &Response{}, nil
		})
		auth = XAuthHandlerFunc(func(head *Header) error {
			return fmt.Errorf("[cmd: %d][subcmd: %d][seq: %d]: Hey, I have got to prohibit you!", cmd, subcmd, head.GetSequence())
		})
	}
	return cmd, subcmd, op, auth
}
