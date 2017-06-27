// xassert.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2016-10-14
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-06-27

// go-xassert is a assert package used to test.
package xassert

import (
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
)

// This interface is used to unify '*testing.T' and '*testing.B' type.
type XT interface {
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	FailNow()
}

// Assert the condition is true.
func IsTrue(xt XT, result bool) {
	assert(xt, result, func() { xt.Error("result is not true") }, 1)
}

// Assert the condition is false.
func IsFalse(xt XT, result bool) {
	assert(xt, !result, func() { xt.Error("result is not false") }, 1)
}

// Assert the actual value is equal to expected value.
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

// Assert the actual value is not equal to expected value.
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

// Assert the actual value is nil.
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

// Assert the actual value is nil.
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

// Assert the string format of the actual value is matched with pattern (regular expression).
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

// Assert the string format of the actual value is not matched with pattern (regular expression).
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
