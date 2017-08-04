// server.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2016-10-13
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-08-04

// This program (server) is used to test go-xconnpool package.
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
	// The delay of handling requests in the server.
	flagDelay = flag.Duration("delay", time.Second, "handle delay")

	// TCP listen port
	flagPort = flag.String("port", "8000", "listen port")
)

var (
	newline = byte('\n')
)

func main() {
	flag.Parse()

	var (
		delay = *flagDelay
		port  = *flagPort
		wg    = &sync.WaitGroup{}
		total int32 // number of current connections
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
		cc   = make(chan net.Conn)
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

	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	fmt.Fprintf(os.Stdout, "[INFO]: Start TCP Server\n")
	for {
		select {
		case sig := <-sc:
			fmt.Fprintf(os.Stdout, "[INFO]: Receive %s Signal\n", sig)
			goto exit
		case conn = <-cc:
			count := atomic.AddInt32(&total, int32(1))
			fmt.Fprintf(os.Stdout, "[INFO]: Accept connection. Total connection number: %d\n", count)
		}

		wg.Add(1)
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
					// Client close the connection, so exit.
					return
				}
				time.Sleep(delay)
				io.WriteString(conn, line)
			}
		}(conn)
	}

exit:
	// Exit after all connections have been finished.
	wg.Wait()
	fmt.Fprintf(os.Stdout, "[INFO]: Shutdown TCP Server\n")
	os.Exit(1)
}
