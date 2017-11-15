// limit_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-11-15
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-11-15
package xlimit

import (
	"github.com/X-Plan/xgo/go-xassert"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestProhibitWrite(t *testing.T) {
	var (
		l     = New(0, time.Second)
		count = 0
	)

	for i := 0; i < 100000; i++ {
		if l.Allow() {
			count++
		}
	}

	xassert.Equal(t, count, 0)
}

func TestOneWriterOneSecond(t *testing.T) {
	testLimiter(t, 100, time.Second, 0, 1)
	testLimiter(t, 100, time.Second, 1000, 1)
	testLimiter(t, 1000, time.Second, 10000, 1)
}

func TestMultiWriterOneSecond(t *testing.T) {
	testLimiter(t, 100, time.Second, 0, 10)
	testLimiter(t, 100, time.Second, 100, 10)
	testLimiter(t, 1000, time.Second, 1000, 10)
}

func testLimiter(t *testing.T, limit int, period time.Duration, count, n int) {
	var (
		wg = &sync.WaitGroup{}
		in = (n*count)/limit + 16
		ia = make([]uint64, in)

		// NOTE: the order of these two variables can't be exchanged.
		start = time.Now()
		l     = New(limit, period)
	)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			for j := 0; j < count; j++ {
				// If can't be allowed, dead loop.
				for !l.Allow() {
				}
				offset := int(time.Now().Sub(start) / period)
				atomic.AddUint64(&(ia[offset]), 1)
			}
			wg.Done()
		}()
	}

	wg.Wait()

	for _, v := range ia {
		xassert.IsTrue(t, int(v) <= limit)
	}
}
