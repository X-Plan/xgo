// xlog_test.go
//
//      Copyright (C), blinklv. All rights reserved.
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-11-08
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2016-11-09

// go-xlog的测试文件.
package xlog

import (
	"github.com/X-Plan/xgo/go-xassert"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// 测试创建日志的合法性.
func TestNew(t *testing.T) {
	var (
		err error
		xl  *XLogger
	)

	xl, err = New(nil)
	xassert.IsNil(t, xl)
	xassert.NotNil(t, err)

	xcfg := &XConfig{
		Level: ERROR,
	}

	xl, err = New(xcfg)
	xassert.NotNil(t, xl)
	xassert.IsNil(t, err)
	xassert.Equal(t, xl.dir, "./log")
	xassert.NotEqual(t, xl.tag, "")
	xassert.IsNil(t, xl.Close())
	xassert.Equal(t, xl.Close(), ErrClosed)
	xassert.IsNil(t, os.Remove("./log"))

	xcfg.MaxSize = -1
	xl, err = New(xcfg)
	xassert.IsNil(t, xl)
	xassert.NotNil(t, err)
	xcfg.MaxSize = 1024 * 1024 * 100

	xcfg.MaxBackups = -1
	xl, err = New(xcfg)
	xassert.IsNil(t, xl)
	xassert.NotNil(t, err)
	xcfg.MaxBackups = 50

	xcfg.MaxAge = "lsdf"
	xl, err = New(xcfg)
	xassert.IsNil(t, xl)
	xassert.NotNil(t, err)
	xcfg.MaxAge = "168h"

	xcfg.Level = 0
	xl, err = New(xcfg)
	xassert.IsNil(t, xl)
	xassert.NotNil(t, err)
	xcfg.Level = INFO

	xl, err = New(xcfg)
	xassert.NotNil(t, xl)
	xassert.IsNil(t, err)
	xassert.IsNil(t, xl.Close())
	xassert.Equal(t, xl.Close(), ErrClosed)
	xassert.IsNil(t, os.Remove("./log"))
}

// 测试写入函数的正确性, 测试方法采取
// 10个协程并发写入, 最后应该确保每个
// 协程写入的信息是完整的且局部保序.
func TestWrite(t *testing.T) {
	xcfg := &XConfig{
		Dir:        "/tmp/xlog",
		MaxSize:    100 * 1024,
		MaxBackups: 100,
		MaxAge:     "1m",
		Level:      DEBUG,
	}
	writeUnitTest(t, xcfg)
}

func writeUnitTest(t *testing.T, xcfg *XConfig) {
	xl, err := New(xcfg)
	xassert.NotNil(t, xl)
	xassert.IsNil(t, err)

	var (
		wg  = &sync.WaitGroup{}
		ids = []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	)

	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			for i := 0; i < 500; i++ {
				n, err := xl.Write([]byte(genBlock(id, i, 1024)))
				xassert.IsNil(t, err)
				xassert.Equal(t, n, 1024)
			}
			wg.Done()
		}(id)
	}

	wg.Wait()
	// 这里需要关闭, 确保内容已经被
	// 持久化.
	xassert.IsNil(t, xl.Close())
}

func genBlock(id string, num, size int) string {
	var (
		snum = strconv.Itoa(num)
	)

	return snum + strings.Repeat(id, size-1-len(snum)) + "\n"
}