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

type Finalizer interface {
	Finalize(string, interface{})
}

type bucket struct {
	sync.RWMutex
	elements  map[string]element
	finalizer Finalizer
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

// Delete an element from the bucket. If the finalizer of the bucket has been set,
// it will finalize that element.
func (b *bucket) del(k string) {
	b.Lock()
	e, found := b.elements[k]
	delete(b.elements, k)
	b.Unlock()

	if found && b.finalizer != nil {
		b.finalizer.Finalize(k, e.data)
	}
	return
}

type pair struct {
	key   string
	value interface{}
}

// Clean all expired elements from the bucket.
func (b *bucket) clean() {
	var (
		pairs []pair
		now   = time.Now().UnixNano()
	)

	b.Lock()
	for k, e := range b.elements {
		// Because calling the expired method of an element every time will
		// generate a timestamp, it's too costly, so inlining this method.
		if e.expiration != 0 && now > e.expiration {
			// Deleting one element in range loop is safe, the more detial you can
			// get from StackOverflow or source codes.
			delete(b.elements, k)
			if b.finalizer != nil {
				pairs = append(pairs, pair{k, e.data})
			}
		}
	}
	b.Unlock()

	for _, pair := range pairs {
		b.finalizer.Finalize(pair.key, pair.value)
	}
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
