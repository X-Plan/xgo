// xretry_test.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-02-02
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-02-05
package xretry

import (
	"errors"
	"fmt"
	"github.com/X-Plan/xgo/go-xassert"
	"testing"
	"time"
)

var errAlive = errors.New("I'm alive!")

func createOP(n int) func() error {
	return func() error {
		fmt.Println(time.Now())
		n--
		if n == 0 {
			return nil
		} else {
			return errAlive
		}
	}
}

func TestAllError(t *testing.T) {
	err, n := Retry(createOP(10), 3, time.Second)
	xassert.NotNil(t, err)
	xassert.Equal(t, n, 3)
}

func TestExistError(t *testing.T) {
	err, n := Retry(createOP(3), 3, time.Second)
	xassert.IsNil(t, err)
	xassert.Equal(t, n, 2)
}

func TestNoError(t *testing.T) {
	err, n := Retry(createOP(1), 3, time.Second)
	xassert.IsNil(t, err)
	xassert.Equal(t, n, 0)
}

func TestNoExecute(t *testing.T) {
	err, n := Retry(createOP(1), -1, time.Second)
	xassert.IsNil(t, err)
	xassert.Equal(t, n, 0)
}

func TestNoRetry(t *testing.T) {
	err, n := Retry(createOP(3), 0, time.Second)
	xassert.NotNil(t, err)
	xassert.Equal(t, n, 0)
}

func TestZeroInterval(t *testing.T) {
	err, n := Retry(createOP(10), 10, time.Duration(0))
	xassert.IsNil(t, err)
	xassert.Equal(t, n, 9)
}

func createFatalOP(n, threshold int) func() error {
	return func() error {
		fmt.Println(time.Now())
		n--
		if n == 0 {
			return nil
		} else if n == threshold {
			return FatalError{errAlive}
		} else {
			return errAlive
		}
	}
}

func TestFatalError(t *testing.T) {
	err, n := Retry(createFatalOP(10, 5), 10, time.Second)
	xassert.Equal(t, err, errAlive)
	xassert.Equal(t, n, 4)
}
