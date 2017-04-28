// xretry.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-02-02
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-04-28

// go-xretry provides the retry mechanism based on Binary Exponential Backoff.
package xretry

import "time"

const Version = "1.1.0"

// This is an auxiliary struct, you can use it to
// to wrap an error returned by the 'op' function.
// When 'Retry' function meet this error, it will
// directly exit, but not retry.
// NOTE: 'Retry' also returns the original error.
type FatalError struct {
	Err error
}

func (fe FatalError) Error() string {
	return fe.Err.Error()
}

// 'Retry' function will repeat executing the 'op' function when 'op' returns
// a non-nil error. The retry number limit equals to 'n', the retry interval
// is based on Binary Exponential Backoff algorithm, the base interval equals
// to the 'interval' argument, then the interval equals to 2 * interval, 4 * interval,
// 8 * interval. It returns the error of the last retry and the retry number.
// NOTE:
// 1. n represent the retry number, if n equals to 3 and all execution failed, the
// execution number equal to 4, excluding the initial execution.
// 2. If n is less than zero, the 'op' function won't be invoked.
// 3. If 'interval' argument equals to zero, all retry interval will equal to zero.
func Retry(op func() error, n int, interval time.Duration) (error, int) {
	var (
		i   int
		err error
	)

	// I hope this loop doesn't make you feel strange. :)
	for i = 0; n >= 0; i++ {
		if err = op(); err == nil || i >= n {
			break
		}

		if fatalError, ok := err.(FatalError); ok {
			err = fatalError.Err
			break
		}

		time.Sleep(interval)
		interval = 2 * interval
	}

	return err, i
}
