// tcp.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-10-13
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-10-13

package xtcpapi

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

const (
	// Environment variable used to store the number of listen file descriptors.
	EnvNumber = "LISTEN_FDS_NUMBER"
	prefix    = EnvNumber + "="
)

type TCP struct {
	mtx       sync.Mutex
	once      sync.Once
	start     int
	inherited []*net.TCPListener
	active    []*net.TCPListener
}

// The network type can only be tcp, tcp4 and tcp6.
func (tcp *TCP) Listen(nt string, addr *net.TCPAddr) (*net.TCPListener, error) {
	if nt == "tcp" || nt == "tcp4" || nt == "tcp6" {
		err := tcp.inherit()
		if err != nil {
			return nil, err
		}

		tcp.mtx.Lock()
		defer tcp.mtx.Unlock()

		// At first, get the listener from the inherited listeners.
		for i, l := range tcp.inherited {
			if l == nil {
				continue
			}

			if equalAddr(addr, l.Addr().(*net.TCPAddr)) {
				tcp.inherited[i] = nil
				tcp.active = append(tcp.active, l)
				return l, nil
			}
		}

		// If not then create a new listener.
		if l, err := net.ListenTCP(nt, addr); err != nil {
			return nil, err
		} else {
			tcp.active = append(tcp.active, l)
			return l, nil
		}

	} else {
		return nil, net.UnknownNetworkError(nt)
	}
}

// Start a new process, transmit environment variables and listen file
// descriptors to the new process.
func (tcp *TCP) StartProcess() (*os.Process, error) {
	var (
		err   error
		ls    = tcp.activeListeners()
		files = make([]*os.File, len(ls))
	)

	for i, l := range ls {
		if files[i], err = l.File(); err != nil {
			return nil, err
		}
		defer files[i].Close()
	}

	var program string
	if program, err = exec.LookPath(os.Args[0]); err != nil {
		return nil, err
	}

	// All environment variables are directly copied to the new process except LISTEN_FDS_NUMBER.
	var env []string
	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, prefix) {
			env = append(env, v)
		}
	}
	env = append(env, fmt.Sprintf("%s%d", prefix, len(ls)))

	wd, _ := os.Getwd()
	process, err := os.StartProcess(program, os.Args, &os.ProcAttr{
		Dir:   wd,
		Env:   env,
		Files: append([]*os.File{os.Stdin, os.Stdout, os.Stderr}, files...),
	})
	if err != nil {
		return nil, err
	}

	return process, nil
}

func (tcp *TCP) inherit() (err error) {
	tcp.once.Do(func() {
		tcp.mtx.Lock()
		defer tcp.mtx.Unlock()

		// If this program first runs, the LISTEN_FDS_NUMBER environment variable
		// hasn't been created, so no listen-fd can be inherited, exit directly.
		env := os.Getenv(EnvNumber)
		if env == "" {
			return
		}

		var number int
		if number, err = strconv.Atoi(env); err != nil {
			err = fmt.Errorf("invalid environment variable: %s=%s", EnvNumber, env)
			return
		}

		start := tcp.start
		if start == 0 {
			start = 3
		}

		var l net.Listener

		for end := start + number; start < end; start++ {
			file := os.NewFile(uintptr(start), "listener")
			if l, err = net.FileListener(file); err == nil {
				if _, ok := l.(*net.TCPListener); !ok {
					err = fmt.Errorf("invalid tcp listener")
					return
				}
			}

			if err != nil {
				file.Close()
				err = fmt.Errorf("inheriting invalid socket fd (%d): (%s)", start, err)
				return
			}

			if err = file.Close(); err != nil {
				err = fmt.Errorf("closing inherited socket fd (%d) failed: (%s)", start, err)
				return
			}

			tcp.inherited = append(tcp.inherited, l.(*net.TCPListener))
		}
	})
	return
}

func (tcp *TCP) activeListeners() []*net.TCPListener {
	tcp.mtx.Lock()
	defer tcp.mtx.Unlock()
	return append([]*net.TCPListener{}, tcp.active...)
}

// Compare two tcp addresses, the type of an address can be tcp4 or tcp6.
func equalAddr(a1, a2 *net.TCPAddr) bool {
	return removePrefix(a1.String()) == removePrefix(a2.String())
}

func removePrefix(as string) string {
	return strings.TrimPrefix(strings.TrimPrefix(as, "0.0.0.0"), "[::]")
}
