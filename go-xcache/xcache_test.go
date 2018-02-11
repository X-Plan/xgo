// xcache_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2018-02-11
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2018-02-11

package xcache

import (
	"fmt"
	"github.com/X-Plan/xgo/go-xassert"
	"hash/fnv"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var aKidsStory = []string{
	"Somebody tell me.",
	"Why it feels more real when I dream than when I am awake.",
	"How can I know If my senses are lying?",
	"",
	"There is some fiction in your truth,",
	"and some truth in your fiction.",
	"To the truth, you must risk everything.",
	"",
	"Who are you?",
	"Am I alone?",
	"",
	"You are not alone.",
	"",
	"                               --- A Kid's Story",
}

func TestFnv32a(t *testing.T) {
	for _, str := range aKidsStory {
		xassert.Equal(t, fnv32a(str), stdFnv32a(str))
	}
}

func stdFnv32a(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func BenchmarkFnv32a(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, str := range aKidsStory {
			fnv32a(str)
		}
	}
}

func BenchmarkStdFnv32a(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, str := range aKidsStory {
			stdFnv32a(str)
		}
	}
}

type counterFinalizer struct {
	count *int64
}

func (cf *counterFinalizer) Finalize(string, interface{}) {
	atomic.AddInt64(cf.count, int64(1))
}

func TestBucket(t *testing.T) {
	var delCount int64
	b := &bucket{
		elements:  make(map[string]element),
		finalizer: &counterFinalizer{&delCount},
	}

	wg := &sync.WaitGroup{}
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			for j := 0; j < 1000; j++ {
				number := i*1000 + j
				b.set(strconv.Itoa(number), number, time.Duration(i%10)*time.Second)
			}
			wg.Done()
		}(i)
	}

	time.Sleep(5 * time.Second)
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(i int) {
			for j := 0; j < 1000; j++ {
				number := i*1000 + j
				v := b.get(strconv.Itoa(number))
				if i == 0 {
					if v.(int) != number {
						panic(fmt.Sprintf("the value of key (%d) is %d", number, v.(int)))
					}
				} else {
					if v != nil {
						panic(fmt.Sprintf("the value of key (%d) should be nil", number))
					}
				}
			}
			wg.Done()
		}(i)
	}

	wg.Wait()

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			for j := 0; j < 1000; j++ {
				number := i*1000 + j
				b.del(strconv.Itoa(number))
			}
			wg.Done()
		}(i)
	}

	wg.Wait()
	xassert.Equal(t, delCount, int64(1000*1000))
}

func TestValidate(t *testing.T) {
	configs := []struct {
		*Config
		ok bool
	}{
		{&Config{16, 30 * time.Minute, &counterFinalizer{}}, true},
		{&Config{32, 1 * time.Hour, nil}, true},
		{&Config{MinBucketNumber - 1, MinCleanInterval, nil}, false},
		{&Config{MaxBucketNumber + 1, MaxCleanInterval, nil}, false},
		{&Config{MinBucketNumber, MinCleanInterval - time.Second, nil}, false},
		{&Config{MinBucketNumber, MaxCleanInterval + time.Second, nil}, false},
	}

	for _, config := range configs {
		err := config.validate()
		if config.ok {
			xassert.IsNil(t, err)
		} else {
			xassert.NotNil(t, err)
		}
	}
}

func TestCache(t *testing.T) {
	var addCount, delCount int64
	cache, _ := New(&Config{
		BucketNumber:  16,
		CleanInterval: time.Minute,
		Finalizer:     &counterFinalizer{&delCount},
	})

	go func() {
		var begin, end = 0, 1000
		for {
			wg := &sync.WaitGroup{}
			for i := begin; i < end; i++ {
				wg.Add(1)
				go func(i int) {
					for j := 0; j < 1000; j++ {
						number := i*1000 + j
						cache.ESet(strconv.Itoa(number), number, time.Duration(i%10)*time.Second)
					}
					atomic.AddInt64(&addCount, int64(1000))
					wg.Done()
				}(i)
			}
			wg.Wait()
			time.Sleep(30 * time.Second)
			begin, end = begin+1000, end+1000
		}
	}()

	for {
		stats(atomic.LoadInt64(&addCount), atomic.LoadInt64(&delCount))
		time.Sleep(10 * time.Second)
	}
}

func stats(addCount, delCount int64) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("Add Count = %v", addCount)
	fmt.Printf("\tDel Count = %v", delCount)
	fmt.Printf("\tAlloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
