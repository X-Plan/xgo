// xassert.go
//
//		Copyright (C), blinklv. All rights reserved.
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-10-14
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2016-10-22

// go-xassert是一个方便测试使用的断言包.
package xassert

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
)

var (
	Version = "1.1.0"
)

// 该接口的目的是为了统一
// *testing.T和*testing.B,
// 它是该库用到的最小功能
// 集合.
type XT interface {
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	FailNow()
}

// 断言实际值等于期望值.
func Equal(xt XT, exp, act interface{}, args ...interface{}) {
	result := reflect.DeepEqual(exp, act)
	assert(xt, result, func() {
		str := fmt.Sprintf("%#v != %#v", exp, act)
		if len(args) > 0 {
			str += " - " + fmt.Sprint(args...)
		}
		xt.Error(str)
	}, 1)
}

// 断言实际值不等于期望值.
func NotEqual(xt XT, exp, act interface{}, args ...interface{}) {
	result := !reflect.DeepEqual(exp, act)
	assert(xt, result, func() {
		str := fmt.Sprintf("%#v == %#v", exp, act)
		if len(args) > 0 {
			str += " - " + fmt.Sprint(args...)
		}
		xt.Error(str)
	}, 1)
}

// 断言实际值为空.
func IsNil(xt XT, act interface{}, args ...interface{}) {
	result := isNil(act)
	assert(xt, result, func() {
		str := fmt.Sprintf("%#v is not nil", act)
		if len(args) > 0 {
			str += " - " + fmt.Sprint(args...)
		}
		xt.Error(str)
	}, 1)
}

// 断言实际值不为空.
func NotNil(xt XT, act interface{}, args ...interface{}) {
	result := !isNil(act)
	assert(xt, result, func() {
		str := fmt.Sprintf("%#v is nil", act)
		if len(args) > 0 {
			str += " - " + fmt.Sprint(args...)
		}
		xt.Error(str)
	}, 1)
}

func isNil(act interface{}) bool {
	if act == nil {
		return true
	}

	switch v := reflect.ValueOf(act); v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	}

	return false
}

func assert(xt XT, result bool, cb func(), cd int) {
	if !result {
		_, file, line, _ := runtime.Caller(cd + 1)
		xt.Errorf("%s:%d", filepath.Base(file), line)
		cb()
		xt.FailNow()
	}
}
