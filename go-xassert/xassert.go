// xassert.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-10-14
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-01-03

// go-xassert是一个方便测试使用的断言包.
package xassert

import (
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
)

var (
	Version = "1.2.2"
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

// 断言条件为真.
func IsTrue(xt XT, result bool) {
	assert(xt, result, func() { xt.Error("result is not true") }, 1)
}

// 断言条件为假.
func IsFalse(xt XT, result bool) {
	assert(xt, !result, func() { xt.Error("result is not false") }, 1)
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
		var str string
		if _, ok := act.(error); ok {
			str = fmt.Sprintf("error (%s) is not nil", act)
		} else {
			str = fmt.Sprintf("%#v is not nil", act)
		}
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
		str := fmt.Sprintf("actual value is nil")
		if len(args) > 0 {
			str += " - " + fmt.Sprint(args...)
		}
		xt.Error(str)
	}, 1)
}

// 正则表达式匹配. 用act的字符串格式和正则
// 表达式匹配.
func Match(xt XT, act interface{}, pattern string, args ...interface{}) {
	result := regexp.MustCompile(pattern).MatchString(fmt.Sprintf("%s", act))
	assert(xt, result, func() {
		str := fmt.Sprintf("(%s) not match (%s)", act, pattern)
		if len(args) > 0 {
			str += " - " + fmt.Sprint(args...)
		}
		xt.Error(str)
	}, 1)
}

func NotMatch(xt XT, act interface{}, pattern string, args ...interface{}) {
	result := !regexp.MustCompile(pattern).MatchString(fmt.Sprintf("%s", act))
	assert(xt, result, func() {
		str := fmt.Sprintf("(%s) match (%s)", act, pattern)
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
