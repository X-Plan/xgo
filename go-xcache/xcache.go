// xcache.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2018-02-08
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2018-02-08

package xcache

import (
	"sync"
	"time"
)

type bucket struct {
	sync.RWMutex
	elements  map[string]element
	finalizer func(string, interface{})
}

// Add an element to the bucket. If the element has existed, replacing it. If the
// duration is zero, which means this element never expires.
func (b *bucket) set(k string, v interface{}, d time.Duration) {
	var expiration int64
	if d > 0 {
		expiration = time.Now().Add(d).UnixNano()
	}
	b.Lock()
	b.elements[k] = element{v, expiration}
	b.Unlock()
}

// Get an element from the bucket. Returns nil if this element doesn't exist or
// has already expired.
func (b *bucket) get(k string) interface{} {
	b.RLock()
	e, found := b.elements[k]
	b.RUnlock()

	if !found || e.expired() {
		return nil
	}
	return e.data
}

type element struct {
	data       interface{}
	expiration int64
}

// Returns true when the element has expired. Returns false directly if the
// 'expiration' field is zero, which means this element has unlimited life.
func (e element) expired() bool {
	return e.expiration != 0 && time.Now().UnixNano() > e.expiration
}
