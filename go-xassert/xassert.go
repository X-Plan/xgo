// xassert.go
//
//		Copyright (C), blinklv. All right reserved.
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-10-14
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2016-10-15

// go-xassert是一个方便测试使用的断言包.
package xassert

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

var (
	Version = "1.0.0"
)

func Equal(t *testing.T, exp, act interface{}, args ...interface{}) {
	result := reflect.DeepEqual(exp, act)
	assert(t, result, func() {
		str := fmt.Sprintf("%#v != %#v", exp, act)
		if len(args) > 0 {
			str += " - " + fmt.Sprint(args...)
		}
		t.Error(str)
	}, 1)
}

func NotEqual(t *testing.T, exp, act interface{}, args ...interface{}) {
	result := !reflect.DeepEqual(exp, act)
	assert(t, result, func() {
		str := fmt.Sprintf("%#v == %#v", exp, act)
		if len(args) > 0 {
			str += " - " + fmt.Sprint(args...)
		}
		t.Error(str)
	}, 1)
}

func IsNil(t *testing.T, act interface{}, args ...interface{}) {
	result := isNil(act)
	assert(t, result, func() {
		str := fmt.Sprintf("%#v is not nil", act)
		if len(args) > 0 {
			str += " - " + fmt.Sprint(args...)
		}
		t.Error(str)
	}, 1)
}

func NotNil(t *testing.T, act interface{}, args ...interface{}) {
	result := !isNil(act)
	assert(t, result, func() {
		str := fmt.Sprintf("%#v is nil", act)
		if len(args) > 0 {
			str += " - " + fmt.Sprint(args...)
		}
		t.Error(str)
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

func assert(t *testing.T, result bool, cb func(), cd int) {
	if !result {
		_, file, line, _ := runtime.Caller(cd + 1)
		t.Errorf("%s:%d", filepath.Base(file), line)
		cb()
		t.FailNow()
	}
}
