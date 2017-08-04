// client.go
//
// Authro: blinklv <blinklv@icloud.com>
// Create Time: 2016-10-13
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-08-04

// This program (client) is used to test go-xconnpool package. Each connection
// is seen as a session, the client will send multiple plain text messages
// separated by newline character. Client closes each session actively.
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
	flagCapacity      = flag.Int("capacity", 10, "capacity of connection pool")
	flagHost          = flag.String("host", "127.0.0.1", "host of server")
	flagPort          = flag.String("port", "8000", "server port")
	flagSessionNumber = flag.Int("session-number", 10, "session number")
	flagMsgNumber     = flag.Int("msg-number", 10, "message number per session")
	flagTestNumber    = flag.Int("test-number", 3, "test number")
)

var newline = byte('\n')

func main() {
	flag.Parse()

	var (
		capacity = *flagCapacity
		sn       = *flagSessionNumber
		mn       = *flagMsgNumber
		tn       = *flagTestNumber
		wg       = &sync.WaitGroup{}
		err      error
		addr     = *flagHost + ":" + *flagPort
		csn      int32 // current session number
	)

	xcp := xconnpool.New(capacity, xconnpool.Factory(func() (conn net.Conn, err error) {
		conn, err = net.Dial("tcp", addr)
		return
	}))

	if xcp == nil {
		fmt.Fprintf(os.Stderr, "[ERROR]: Create XConnPool Failed\n")
		os.Exit(1)
	}

	for j := 0; j < tn; j++ {
		fmt.Fprintf(os.Stdout, "[Test: %d] Start\n", j)

		// Create sessions
		for i := 0; i < sn; i++ {
			wg.Add(1)
			// si (session index)
			go func(si int) {
				defer func() {
					count := atomic.AddInt32(&csn, int32(-1))
					fmt.Fprintf(os.Stdout, "[State] Current Session Number: %d XConnPool Size: %d\n", count, xcp.Size())
					wg.Done()
				}()

				fmt.Fprintf(os.Stdout, "[Session: %d] Start\n", si)
				count := atomic.AddInt32(&csn, int32(1))

				conn, err := xcp.Get()
				if err != nil {
					fmt.Fprintf(os.Stdout, "[Session: %d] Error Exit (%s)\n", err)
					return
				}
				defer conn.Close()
				fmt.Fprintf(os.Stdout, "[State] Current Session Number: %d	XConnPool Size: %d\n", count, xcp.Size())

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
		fmt.Fprintf(os.Stdout, "[Test: %d] End\n", j)
	}

	fmt.Fprintf(os.Stdout, "[INFO] Close XConnPool (But you have to wait 3s)\n")
	time.Sleep(3 * time.Second)

	if err = xcp.Close(); err != nil {
		fmt.Fprint(os.Stderr, "[ERROR]: %s\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "[INFO] Quit\n")
}
