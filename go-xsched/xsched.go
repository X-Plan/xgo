// xsched.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-03-10
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-11-24

// go-xsched is a scheduler for load balancing, the implementation of it
// is based on weight round-robin algorithm, it's concurrent-safe too.
package xsched

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Scheduler is used for interface specification.
type Scheduler interface {
	Get() (string, error)
	Feedback(string, bool)
}

type XScheduler struct {
	rwmtx sync.RWMutex

	// The following fields are protected by 'rwmtx' field. Why I don't use 'mtx'
	// field?  Because the following fields only be changed by 'Update', and
	// 'Remove' methods, The number of operating them is far less than the number
	// of calling 'Get' method, I think a read-write lock is more suitable.
	addrs []*addrUnit
	addrm map[string]*addrUnit
	n     int // number of address items
	max   int // max weight
	delta int // greatest common divisor

	mtx sync.Mutex

	// The following fields are protected by 'mtx' field. They will be modified
	// by 'Get' method every time, I can only try to reduce the critical region.
	i  int
	cw int
}

// Create a new instance of XScheduler. The strs parameter is the collection
// of server address, the format of address likes 'host:port:weight'. If the
// weight field of the item is zero, this item will be ignored, but the weight
// field is empty or negative will return an error. If the multiple items share
// the common 'host:port' prefix, only the last nonzero-weight item can be used.
func New(strs []string) (*XScheduler, error) {
	var (
		xs    = &XScheduler{addrm: make(map[string]*addrUnit)}
		delta = 1
	)

	for i, str := range strs {
		u := newAddrUnit(str)
		if u == nil {
			return nil, fmt.Errorf("invalid address (%s)", str)
		}

		// Although the zero-weight item is valid, but it will be ignored.
		if u.weight == 0 {
			continue
		}

		// If the addresses are duplicate, the new item will
		// overwrite the old one.
		if _, ok := xs.addrm[u.address]; ok {
			xs.addrs = removeUnit(xs.addrs, u.address)
		}
		xs.addrs = append(xs.addrs, u)
		xs.addrm[u.address] = u

		if u.weight > xs.max {
			xs.max = u.weight
		}

		// Find the greatest common divisor of the all weights.
		if i != 0 {
			delta = gcd(delta, u.weight)
		} else {
			delta = u.weight
		}
	}

	for _, u := range xs.addrs {
		u.weight = u.weight / delta
	}
	xs.max, xs.delta = xs.max/delta, delta

	xs.n, xs.i = len(xs.addrs), -1
	return xs, nil
}

// Get the address from scheduler, this function is concurrent-safe.
func (xs *XScheduler) Get() (string, error) {
	xs.rwmtx.RLock()
	defer xs.rwmtx.RUnlock()

	var (
		i, cw   int
		u       *addrUnit
		retry   = 2 * xs.n
		address string
	)

	for retry > 0 {
		xs.mtx.Lock()
		xs.i = (xs.i + 1) % xs.n
		if xs.i == 0 {
			// We can directly use decrement operator, because we have
			// already normalize the weight field (divided by gcd).
			xs.cw--
			if xs.cw <= 0 {
				xs.cw = xs.max
			}
		}
		i, cw = xs.i, xs.cw
		xs.mtx.Unlock()

		u = xs.addrs[i]
		if u.IsAvailable() {
			if u.weight >= cw {
				return u.address, nil
			}
			address = u.address
		}
		retry--
	}

	// In high concurrent case, it's possible to can't satisfy condition
	// 'u.weight >= cw' in all of the loops, so return the last address.
	if address != "" {
		return address, nil
	} else {
		return "", errors.New("all hosts are temporarily unavailable")
	}
}

// Feedback the result of an operation on special address, true
// represent success, false represent failure.
func (xs *XScheduler) Feedback(address string, result bool) {
	xs.rwmtx.RLock()
	if u, ok := xs.addrm[address]; ok {
		u.Feedback(result)
	}
	xs.rwmtx.RUnlock()
}

func (xs *XScheduler) Update(str string) error {
	return nil
}

// Remove an existing address, the format of 'str' parameter can be
// 'ip:port', don't need the weight field.
func (xs *XScheduler) Remove(str string) error {
	tuple := strings.Split(str, ":")
	if len(tuple) < 2 {
		return fmt.Errorf("invalid format (%s)", str)
	}

	xs.rwmtx.Lock()
	defer xs.rwmtx.Unlock()

	address := tuple[0] + ":" + tuple[1]
	unit := xs.addrm[address]
	if unit == nil {
		return fmt.Errorf("address (%s) doesn't exist", address)
	}

	delete(xs.addrm, address)

	i, delta, xs.max = 0, 1, 0
	for k, u := range xs.addrs {
		if u != unit {
			u.weight *= xs.delta
			if u.weight > xs.max {
				xs.max = u.weight
			}

			if i != 0 {
				delta = gcd(delta, u.weight)
			} else {
				delta = u.weight
			}
		} else {
			i = k
		}
	}

	xs.addrs = append(xs.addrs[0:i], xs.addrs[i+1:]...)
	for _, u := range xs.addrs {
		u.weight = u.weight / delta
	}
	xs.max, xs.delta, xs.n = xs.max/delta, delta, len(xs.addrs)

	// If the deleted element in front of the index 'xs.i', shift 'xs.i'
	// for pointing to the original element. Because 'i' greater than zero
	// and 'xs.i' greater than it, so 'xs.i - 1' must greater than zero.
	if xs.n > 0 && xs.i > i {
		xs.i = (xs.i - 1) % xs.n
	}

	return nil
}

const (
	zeroInterval    = time.Duration(0)
	minSamplePeriod = 2 * time.Second
	maxSamplePeriod = 32 * time.Second
	minWaitInterval = 2 * time.Second
	maxWaitInterval = 128 * time.Second
)

type addrUnit struct {
	address string
	weight  int

	rwmtx        sync.RWMutex
	available    bool
	total        int
	fail         int
	samplePeriod time.Duration
	sampleTime   time.Time
	waitInterval time.Duration
	wakeupTime   time.Time
}

func (u *addrUnit) IsAvailable() bool {
	u.rwmtx.RLock()
	defer u.rwmtx.RUnlock()

	if u.available || time.Now().After(u.wakeupTime) {
		return true
	} else {
		return false
	}
}

// Call this function is similar to sample, the sampling period
// is controlled by the 'samplePeriod' field. When the duration
// of sampling exceeds the 'samplePeriod', this function will
// evaluate the fail rate (fail number divided by total number).
// If the fail rate is greater than ten percent, this address will
// be marked unavailable (affect 'Get' function). But it doesn't
// mean the address always remain unavailable. After waiting
// some time (controlled by the 'waitInterval' field), it will be
// awaked.
func (u *addrUnit) Feedback(result bool) {
	u.rwmtx.Lock()
	defer u.rwmtx.Unlock()

	u.total++
	if !result {
		u.fail++
	}

	now := time.Now()
	if now.After(u.sampleTime) {
		var failRate float64
		if u.total > 0 {
			failRate = float64(u.fail) / float64(u.total)
		}

		if failRate < 0.1 {
			u.available = true

			// 'samplePeriod' and 'waitInterval' is not static. when the
			// number of failures increases, the 'samplePeriod' will decrease
			// and 'waitInterval' will also increase. Why we do this is
			// based on the assumption: the more times it fails, the more
			// likely it will fail next time. So if we want to minimize the
			// effect of fail, we should decrease 'samplePeriod' and increase
			// 'waitInterval'.
			if u.samplePeriod < maxSamplePeriod {
				u.samplePeriod <<= 1 // Divided 2
			}

			if u.waitInterval > minWaitInterval {
				u.waitInterval >>= 1 // Times 2
			}
		} else {
			u.available = false

			if u.samplePeriod > minSamplePeriod {
				u.samplePeriod >>= 1
			}

			if u.waitInterval < maxWaitInterval {
				u.waitInterval <<= 1
			}
			// Only in the case of failure, set 'wakeupTime' field.
			u.wakeupTime = now.Add(u.waitInterval)
		}

		if u.available {
			u.sampleTime = now.Add(u.samplePeriod)
		} else {
			// If the current state is unavailable, the calculation of
			// 'sampleTime' should be based on 'wakeupTime'.
			u.sampleTime = u.wakeupTime.Add(u.samplePeriod)
		}

		u.total, u.fail = 0, 0
	}
}

func newAddrUnit(str string) *addrUnit {
	var (
		err  error
		w    int64
		strs = strings.Split(strings.TrimSpace(str), ":")
	)

	if len(strs) != 3 {
		return nil
	}

	if w, err = strconv.ParseInt(strs[2], 10, 64); err != nil {
		return nil
	}

	// Weight can't be negative, but it can be equal to zero.
	if w < 0 {
		return nil
	}

	// Just check whether the host/port field is valid.
	if _, err = strconv.ParseUint(strs[1], 10, 16); err != nil {
		return nil
	}

	if _, err = net.ResolveIPAddr("ip", strs[0]); err != nil {
		return nil
	}

	u := &addrUnit{
		address:      strs[0] + ":" + strs[1],
		weight:       int(w),
		available:    true,
		samplePeriod: maxSamplePeriod,
		waitInterval: minWaitInterval,
		sampleTime:   time.Now().Add(maxSamplePeriod),
	}

	return u
}

func removeUnit(addrs []*addrUnit, address string) []*addrUnit {
	var (
		i int
		u *addrUnit
	)

	for i, u = range addrs {
		if u.address == address {
			break
		}
	}

	return append(addrs[:i], addrs[i+1:]...)
}

// Greatest common divisor.
func gcd(a, b int) int {
	for b > 0 {
		a, b = b, a%b
	}
	return a
}
