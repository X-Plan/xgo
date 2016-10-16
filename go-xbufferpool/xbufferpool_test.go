// xbufferpool_test.go
//
//		Copyright (C), blinklv. All right reserved.
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-10-16
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2016-10-16
package xbufferpool

import (
	"github.com/X-Plan/xgo/go-xassert"
	"strconv"
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
