// xcache.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2018-02-08
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2018-02-09

package xcache

import (
	"fmt"
	"sync"
	"time"
)

type Finalizer interface {
	Finalize(string, interface{})
}

const (
	MinBucketNumber  = 1
	MaxBucketNumber  = 256
	MinCleanInterval = 1 * time.Minute
	MaxCleanInterval = 24 * time.Hour
)

// This configure type is used to create Cache.
type Config struct {
	BucketNumber  int
	CleanInterval time.Duration
	Finalizer     Finalizer
}

func (cfg *Config) validate() error {
	if cfg.BucketNumber < MinBucketNumber || cfg.BucketNumber > MaxBucketNumber {
		return fmt.Errorf("the number of bucket (%d) isn't between %d and %d", cfg.BucketNumber, MinBucketNumber, MaxBucketNumber)
	}

	if cfg.CleanInterval < MinCleanInterval || cfg.CleanInterval > MaxCleanInterval {
		return fmt.Errorf("the clean interval (%s) isn't between %s and %s", cfg.CleanInterval, MinCleanInterval, MaxCleanInterval)
	}

	return nil
}

type Cache struct {
	buckets  []*bucket
	n        uint32
	stop     chan struct{}
	interval time.Duration
}

// Create a new Cache instance.
func New(cfg *Config) (*Cache, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	cache := &Cache{
		buckets:  make([]*bucket, cfg.BucketNumber),
		n:        uint32(cfg.BucketNumber),
		stop:     make(chan struct{}),
		interval: cfg.CleanInterval,
	}

	for i, _ := range cache.buckets {
		cache.buckets[i] = &bucket{
			elements:  make(map[string]element),
			finalizer: cfg.Finalizer,
		}
	}

	return cache, nil
}

// Add an element to the cache. If the element has existed, replacing it.
func (c *Cache) Set(k string, v interface{}) {
	c.buckets[fnv32a(k)%c.n].set(k, v, time.Duration(0))
}

// Add an element to the cache with an expiration. If the element has existed,
// replacing it. If the duration is zero, the effect is same as using Set method.
// Otherwise the element won't be get when it has been expired.
func (c *Cache) ESet(k string, v interface{}, d time.Duration) {
	c.buckets[fnv32a(k)%c.n].set(k, v, d)
}

// Get an element from the cache. Return nil if this element doesn't exist or
// has already expired.
func (c *Cache) Get(k string) interface{} {
	return c.buckets[fnv32a(k)%c.n].get(k)
}

// Delete an element from the cache. If the finalizer of the cache has been set,
// it will finalize that element.
func (c *Cache) Del(k string) {
	c.buckets[fnv32a(k)%c.n].del(k)
}

// Calling the clean method of each bucket to clean all expired elements periodically.
func (c *Cache) clean() {
	ticker := time.NewTicker(c.interval)
	for {
		select {
		case <-ticker.C:
			// It's not all buckets execute clean opearation simultaneously, but
			// one by one. It's too waste time when a bucket execute the clean
			// method, if all buckets do this at the same time, all user requests
			// will be blocked. So I decide clean buckets sequentially to reduce
			// this effect.
			for _, b := range c.buckets {
				b.clean()
			}
		case <-c.stop:
			ticker.Stop()
			return
		}
	}
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

	// Do this opeartion need to run in a new goroutine? I'm thinking of it. :)
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

const (
	offset32 = 0x811c9dc5
	prime32  = 0x1000193
)

// Takes a string and return a 32 bit FNV-1a. This function makes no memory allocations.
func fnv32a(s string) uint32 {
	var h uint32 = offset32
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= prime32
	}
	return h
}
