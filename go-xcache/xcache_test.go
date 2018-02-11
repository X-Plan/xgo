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
	count int64
}

func (cf *counterFinalizer) Finalize(string, interface{}) {
	atomic.AddInt64(&(cf.count), int64(1))
}

func TestBucket(t *testing.T) {
	b := &bucket{
		elements:  make(map[string]element),
		finalizer: &counterFinalizer{},
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
	xassert.Equal(t, b.finalizer.(*counterFinalizer).count, int64(1000*1000))
}
