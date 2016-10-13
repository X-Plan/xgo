// server.go
//
//		Copyright (C), blinklv. All right reserved.
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-10-13
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2016-10-13

// 该程序用于测试go-xconnpool包. 它是这个测试
// 过程中的服务端, 用来回显客户端的请求.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	// 服务端的处理延时, 设置该项是为了更加方便的
	// 观测连接的创建与关闭.
	flagDelay = flag.Duration("delay", time.Second, "handle delay")

	// TCP服务端口号.
	flagPort = flag.String("port", "8000", "tcp port")
)

var (
	// 换行符.
	newline = byte('\n')
)

func main() {
	flag.Parse()

	var (
		delay = *flagDelay
		port  = *flagPort
		wg    = &sync.WaitGroup{}
		// 用来记录当前的连接数.
		total int32
	)

	addr, err := net.ResolveTCPAddr("tcp", ":"+port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR]: %s\n", err)
		os.Exit(1)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR]: %s\n", err)
		os.Exit(1)
	}

	var (
		conn net.Conn
		sc   = make(chan os.Signal, 1)
		cc   = make(chan net.Conn) // 无缓冲区
	)

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				fmt.Fprintf(os.Stderr, "[ERROR]: %s\n", err)
				os.Exit(1)
			}
			cc <- conn
		}
	}()

	// 监听信号. 这样做的目的是希望
	// 当服务收到这些信号的时候能
	// 优雅的退出(将还未处理完的请求
	// 处理完).
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	fmt.Fprintf(os.Stdout, "[INFO]: Start TCP Server\n")
	// 连接处理循环.
	for {
		select {
		// 信号事件与接收事件都不应该放在
		// default范围内, 如果这样做其中
		// 任何一个可能阻塞另一个的执行.
		case sig := <-sc:
			// 收到退出信号, 退出接收循环.
			fmt.Fprintf(os.Stdout, "[INFO]: Receive %s Signal\n", sig)
			goto exit
		case conn = <-cc:
			count := atomic.AddInt32(&total, int32(1))
			fmt.Fprintf(os.Stdout, "[INFO]: Accept connection. Total connection number: %d\n", count)
		}

		wg.Add(1)
		// 处理请求. conn参数时常变动, 因此
		// 应该以参数的形式传递给处理函数.
		go func(conn net.Conn) {
			defer func() {
				conn.Close()
				wg.Done()
				count := atomic.AddInt32(&total, int32(-1))
				fmt.Fprintf(os.Stdout, "[INFO]: Close connection. Total connection number: %d\n", count)
			}()

			r := bufio.NewReader(conn)
			for {
				line, err := r.ReadString(newline)
				if err != nil && err != io.EOF {
					fmt.Fprintf(os.Stderr, "[ERROR]: %s\n", err)
					return
				} else if err == io.EOF {
					// 客户端关闭连接, 退出.
					return
				}
				// 停顿一定的时间.
				time.Sleep(delay)
				io.WriteString(conn, line)
			}
		}(conn)
	}

exit:
	// 必须在所有连接已经处理完才会
	// 退出服务.
	wg.Wait()
	fmt.Fprintf(os.Stdout, "[INFO]: Shutdown TCP Server\n")
	os.Exit(1)
}
