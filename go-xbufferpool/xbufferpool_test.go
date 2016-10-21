// xbufferpool_test.go
//
//		Copyright (C), blinklv. All rights reserved.
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-10-16
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2016-10-22
package xbufferpool

import (
	"bytes"
	"github.com/X-Plan/xgo/go-xassert"
	"runtime"
	"strconv"
	"sync"
	"testing"
)

// 测试缓冲池的正确性.
func TestBufferPool(t *testing.T) {
	var (
		xbp *XBufferPool
		err error
	)
	xbp = New(-1, -1)
	xassert.IsNil(t, xbp)
	xbp = New(-1, 0)
	xassert.IsNil(t, xbp)
	xbp = New(1000, 0)
	xassert.NotNil(t, xbp)

	var xbs = make([]*XBuffer, 0, 1000)
	// 先写入1000个数字标示这些缓冲区.
	for i := 0; i < 1000; i++ {
		xb, err := xbp.Get()
		xassert.IsNil(t, err)
		xassert.Equal(t, 0, xb.Len())
		_, err = xb.WriteString(strconv.Itoa(i))
		xassert.IsNil(t, err)
		xbs = append(xbs, xb)

		xassert.Equal(t, 0, xbp.Size())
	}

	// 释放1000个缓冲区.
	for _, xb := range xbs {
		err = xb.Close()
		xassert.IsNil(t, err)
	}

	// 再次释放应该返回对应的错误.
	for _, xb := range xbs {
		err = xb.Close()
		xassert.Equal(t, ErrXBufferIsNil, err)
	}

	xbs = make([]*XBuffer, 0, 1000)

	// 再次获取1000次, 这1000次得到缓冲区
	// 还是之前的缓冲区, 但是内容被重置.
	for i := 0; i < 1000; i++ {
		xassert.Equal(t, 1000-i, xbp.Size())
		xb, err := xbp.Get()
		xassert.IsNil(t, err)
		xassert.Equal(t, 0, xb.Len())
		xbs = append(xbs, xb)
	}

	// 再次获取500次
	for i := 0; i < 1000; i++ {
		xb, err := xbp.Get()
		xassert.IsNil(t, err)
		xassert.Equal(t, 0, xbp.Size())
		xbs = append(xbs, xb)
	}

	for i, xb := range xbs {
		if i < 1000 {
			xassert.Equal(t, i, xbp.Size())
		} else {
			xassert.Equal(t, 1000, xbp.Size())
		}
		xassert.IsNil(t, xb.Close())
	}
}

// 测试性能. 主要是和不用缓冲池做对比.
func benchmarkBufferPool(dataSize int, b *testing.B) {
	// 制造一份数据用于写入.
	var (
		data = make([]byte, dataSize)
	)
	for i := 0; i < len(data); i++ {
		data[i] = byte('1')
	}

	// 先进行一次垃圾回收.
	runtime.GC()

	var (
		xbp = New(b.N, 0)
		wg  = &sync.WaitGroup{}
	)

	b.StartTimer()
	// 100个go-routine同时获取, 每个go-routine
	// 获取b.N次缓存.
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			var (
				xb *XBuffer
			)
			for i := 0; i < b.N; i++ {
				xb, _ = xbp.Get()
				xb.Write(data)
				xb.Close()
			}
			wg.Done()
		}()
	}
	wg.Wait()
	b.StopTimer()
}

// 这里构造一个鸡肋缓存池用作对比.
type dummyBuffer struct {
	*bytes.Buffer
}

func (db *dummyBuffer) Close() error {
	return nil
}

type dummyBufferPool struct{}

func (dbp *dummyBufferPool) Get() (*dummyBuffer, error) {
	db := &dummyBuffer{}
	db.Buffer = new(bytes.Buffer)
	return db, nil
}

func (dbp *dummyBufferPool) Close() error {
	return nil
}

// 性能测试, 使用傀儡缓存池.
func benchmarkDummyBufferPool(dataSize int, b *testing.B) {
	// 制造一份数据用于写入.
	var (
		data = make([]byte, dataSize)
	)
	for i := 0; i < len(data); i++ {
		data[i] = byte('1')
	}

	// 先进行一次垃圾回收.
	runtime.GC()

	var (
		dbp = &dummyBufferPool{}
		wg  = &sync.WaitGroup{}
	)

	b.StartTimer()
	// 100个go-routine同时获取, 每个go-routine
	// 获取b.N次缓存.
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			var (
				db *dummyBuffer
			)
			for i := 0; i < b.N; i++ {
				db, _ = dbp.Get()
				db.Write(data)
				db.Close()
			}
			wg.Done()
		}()
	}
	wg.Wait()
	b.StopTimer()
}

func BenchmarkBufferPool128(b *testing.B)  { benchmarkBufferPool(128, b) }
func BenchmarkBufferPool256(b *testing.B)  { benchmarkBufferPool(256, b) }
func BenchmarkBufferPool512(b *testing.B)  { benchmarkBufferPool(512, b) }
func BenchmarkBufferPool1024(b *testing.B) { benchmarkBufferPool(1024, b) }
func BenchmarkBufferPool2048(b *testing.B) { benchmarkBufferPool(2048, b) }
func BenchmarkBufferPool4096(b *testing.B) { benchmarkBufferPool(4096, b) }
func BenchmarkBufferPool8192(b *testing.B) { benchmarkBufferPool(8192, b) }

func BenchmarkDummyBufferPool128(b *testing.B)  { benchmarkDummyBufferPool(128, b) }
func BenchmarkDummyBufferPool256(b *testing.B)  { benchmarkDummyBufferPool(256, b) }
func BenchmarkDummyBufferPool512(b *testing.B)  { benchmarkDummyBufferPool(512, b) }
func BenchmarkDummyBufferPool1024(b *testing.B) { benchmarkDummyBufferPool(1024, b) }
func BenchmarkDummyBufferPool2048(b *testing.B) { benchmarkDummyBufferPool(2048, b) }
func BenchmarkDummyBufferPool4096(b *testing.B) { benchmarkDummyBufferPool(4096, b) }
func BenchmarkDummyBufferPool8192(b *testing.B) { benchmarkDummyBufferPool(8192, b) }
