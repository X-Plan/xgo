// client.go
//
//		Copyright (C), blinklv. All right reserved.
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-10-13
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2016-10-13

// 该程序用于测试go-xconnpool包. 它是这个测试
// 过程中的客户端, 用来向服务端发送请求. 每个
// 客户端与服务端建立的一条连接都认为是一个次
// 会话. 每次会话客户端发送若干条消息. 消息是
// 普通的文本格式, 以换行符为分界线. 每次会话
// 均是客户端主动关闭连接.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/X-Plan/xgo/go-xconnpool"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// 连接池的容量.
	flagCapacity = flag.Int("capacity", 100, "capacity of connection pool")

	// 服务端的主机地址.
	flagHost = flag.String("host", "127.0.0.1", "host of server")

	// 服务端的tcp端口号.
	flagPort = flag.String("port", "8000", "server port")

	// 会话次数.
	flagSessionNumber = flag.Int("session-number", 100, "session number")

	// 每次会话发送消息的次数.
	flagMsgNumber = flag.Int("msg-number", 100, "message number per session")
)

// 换行符.
var newline = byte('\n')

func main() {
	flag.Parse()

	var (
		capacity = *flagCapacity
		sn       = *flagSessionNumber
		mn       = *flagMsgNumber
		wg       = &sync.WaitGroup{}
		err      error
		addr     = *flagHost + ":" + *flagPort
		// 当前的session number.
		csn int32
	)

	// 创建连接池. 工厂函数简单的使用
	// net.Dial.
	xcp := xconnpool.New(capacity, xconnpool.Factory(func() (conn net.Conn, err error) {
		conn, err = net.Dial("tcp", addr)
		return
	}))

	if xcp == nil {
		fmt.Fprintf(os.Stderr, "[ERROR]: Create XConnPool Failed\n")
		os.Exit(1)
	}

	// 创建会话.
	for i := 0; i < sn; i++ {
		wg.Add(1)
		// si(session index)为每个会话的编号.
		go func(si int) {
			defer func() {
				count := atomic.AddInt32(&csn, int32(-1))
				fmt.Fprintf(os.Stdout, "[State] Current Session Number: %d XConnPool Size: %d\n", count, xcp.Size())
				wg.Done()
			}()

			fmt.Fprintf(os.Stdout, "[Session: %d] Start\n", si)
			count := atomic.AddInt32(&csn, int32(1))

			// 从连接池中获取连接.
			conn, err := xcp.Get()
			if err != nil {
				fmt.Fprintf(os.Stdout, "[Session: %d] Error Exit (%s)\n", err)
				return
			}
			defer conn.Close()
			fmt.Fprintf(os.Stdout, "[State] Current Session Number: %d	XConnPool Size: %d\n", count, xcp.Size())

			// 发送消息.
			for i := 0; i < mn; i++ {
				fmt.Fprintf(conn, "[Session: %d] %d\n", si, i)
				r := bufio.NewReader(conn)
				line, err := r.ReadString(newline)
				if err != nil {
					fmt.Fprintf(os.Stdout, "[Session: %d] Error Exit (%s)\n", err)
					return
				}
				fmt.Fprintf(os.Stdout, "%s", line)
			}

			fmt.Fprintf(os.Stdout, "[Session: %d] End\n", si)
		}(i)
	}

	wg.Wait()
	// 这里进行延时3秒, 用来观测服务端的部分连接是
	// 在客户端调用关闭连接池后才关闭的.
	fmt.Fprintf(os.Stdout, "[INFO] Close XConnPool (But you have to wait 3s)\n")
	time.Sleep(3 * time.Second)

	// 应该在所有的会话都是释放连接后才关闭连接池.
	// 若不满足该条件, 在XConn进行Close操作的时候
	// 可能会失败(因为连接池已经关闭).
	if err = xcp.Close(); err != nil {
		fmt.Fprint(os.Stderr, "[ERROR]: %s\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "[INFO] Quit\n")
}
