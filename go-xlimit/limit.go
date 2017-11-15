// limit.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-11-15
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-11-15

// This package is used to control the frequency of operations.
package xlimit

import (
	"sync"
	"time"
)

type Limiter struct {
	mtx    *sync.Mutex
	limit  int
	count  int
	last   time.Time
	period time.Duration
}

// Create a new limiter, the parameters of this function mean during
// this 'duration', you operate 'limit' times at most. The 'duration'
// argument can't be less than one second, if it happens, this function
// will convert the accuracy to one second.
func New(limit int, duration time.Duration) *Limiter {
	if duration < time.Second {
		limit = int((float64(time.Second) / float64(duration)) * float64(limit))
		duration = time.Second
	}

	return &Limiter{
		mtx:    &sync.Mutex{},
		limit:  limit,
		last:   time.Now(),
		period: duration,
	}
}

// Check whether you can do an operation.
func (l *Limiter) Allow() bool {
	ok, now := false, time.Now()

	l.mtx.Lock()
	if now.Sub(l.last) >= l.period {
		l.count, l.last = 0, now
	}

	if l.count < l.limit {
		ok, l.count = true, l.count+1
	}
	l.mtx.Unlock()

	return ok
}
