// xlog_test.go
//
//      Copyright (C), blinklv. All rights reserved.
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-11-08
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2016-11-08

// go-xlog的测试文件.
package xlog

import (
	"github.com/X-Plan/xgo/go-xassert"
	"os"
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

func TestWrite(t *testing.T) {
}
