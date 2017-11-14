// pid_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-11-14
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-11-14

package xpid

import (
	"github.com/X-Plan/xgo/go-xassert"
	"os"
	"testing"
)

func TestSetAndGet(t *testing.T) {
	var (
		pid     int
		err     error
		pidfile = "run/test.pid"
	)

	err = Set(pidfile)
	xassert.IsNil(t, err)

	pid, err = Get(pidfile)
	xassert.IsNil(t, err)
	xassert.Equal(t, pid, os.Getpid())
}
