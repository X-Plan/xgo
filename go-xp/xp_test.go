// xp_test.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-02-07
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-02-07

package xp

import (
	"fmt"
	"github.com/X-Plan/xgo/go-xassert"
	"github.com/X-Plan/xgo/go-xlog"
	"io"
	"io/ioutil"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

func getFreeListener(t *testing.T) (net.Listener, string) {
	l, err := net.Listen("tcp", "0.0.0.0:0")
	xassert.IsNil(t, err)
	_, port, err := net.SplitHostPort(l.Addr().String())
	xassert.IsNil(t, err)
	return l, port
}

func runServer(l net.Listener, duration time.Duration, errch chan error) {
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
}
