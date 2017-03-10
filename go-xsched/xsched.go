// xsched.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-03-10
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-03-10

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

type XScheduler struct {
	addrs []*addrUnit
	addrm map[string]*addrUnit
	n     int
	max   int

	mtx sync.Mutex
	i   int
	cw  int
}

// Create a new instance of XScheduler. The strs parameter is the collection
// of server address, the format of address likes 'host:port:weight'. If the
// weight field of the item is zero, this item will be ignored, but the weight
// field is empty will return an error. Allow the multiple items have same
// 'host:port' prefix.
func New(strs []string) (*XScheduler, error) {
	var (
		xs    = &XScheduler{addrm: make(map[string]*addrUnit)}
		delta int
	)

	for i, str := range strs {
	}
}

const (
	zeroInterval      = time.Duration(0)
	minSampleInterval = 2 * time.Second
	maxSampleInterval = 32 * time.Second
	minWaitInterval   = 2 * time.Second
	maxWaitInterval   = 128 * time.Second
)

type addrUnit struct {
	address string
	weight  int

	rwmtx          sync.RWMutex
	available      bool
	total          int
	fail           int
	sampleInterval time.Duration
	sampleTime     time.Time
	waitInterval   time.Duration
	wakeupTime     time.Time
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
		address:        strs[0] + ":" + strs[1],
		weight:         int(w),
		available:      true,
		sampleInterval: maxSampleInterval,
		waitInterval:   minWaitInterval,
		sampleTime:     time.Now().Add(maxSampleInterval),
	}

	return u
}

// Greatest common divisor.
func gcd(a, b int) int {
	for b > 0 {
		a, b = b, a%b
	}
	return a
}
