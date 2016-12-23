// xassert_test.go
//
//		Copyright (C), blinklv. All rights reserved.
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-10-14
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2016-12-23

package xassert

import (
	"errors"
	"testing"
)

type dummy struct {
	field1 int
	field2 float32
	field3 string
	field4 []int
}

var (
	exp = dummy{
		field1: 1,
		field2: 1.0,
		field3: "1",
		field4: []int{1, 2, 3},
	}

	act = dummy{
		field1: 1,
		field2: 1.0,
		field3: "1",
		field4: []int{2, 3, 4},
	}
)

func TestEqual(t *testing.T) {
	Equal(t, exp, exp, "msg")
	//  Equal(t, exp, act, "error")
}

func TestNotEqual(t *testing.T) {
	NotEqual(t, exp, act, "msg")
	//  NotEqual(t, exp, exp)
}

func TestIsNil(t *testing.T) {
	var (
		nilChan      chan int
		nilFunc      func()
		nilInterface interface{}
		nilMap       map[int]int
		nilPtr       *int
		nilSlice     []int
	)
	//  IsNil(t, exp)
	IsNil(t, nil, "nil")
	IsNil(t, nilChan, "nilChan")
	IsNil(t, nilFunc, "nilFunc")
	IsNil(t, nilInterface, "nilInterface")
	IsNil(t, nilMap, "nilMap")
	IsNil(t, nilPtr, "nilPtr")
	IsNil(t, nilSlice, "nilSlice")
}

func TestNotNil(t *testing.T) {
	var (
		notNilChan      chan int    = make(chan int)
		notNilFunc      func()      = func() {}
		notNilInterface interface{} = 1
		notNilMap       map[int]int = make(map[int]int)
		notPtr          *int        = new(int)
		notSlice        []int       = make([]int, 0)
	)

	//  NotNil(t, nil)
	NotNil(t, exp, "notNil")
	NotNil(t, notNilChan, "notNilChan")
	NotNil(t, notNilFunc, "notNilFunc")
	NotNil(t, notNilInterface, "notNilInterface")
	NotNil(t, notNilMap, "notNilMap")
	NotNil(t, notPtr, "notPtr")
	NotNil(t, notSlice, "notSlice")
}

type foo struct {
	x int
	b string
}

type bar struct {
	a int
	b string
	c map[int]string
	d foo
}

func TestFoo(t *testing.T) {
	var (
		a map[int]string
		b = bar{
			a: 1,
			b: "hello world",
			c: make(map[int]string),
			d: foo{
				x: 1,
				b: "who are you?",
			},
		}
	)

	IsNil(t, a)
	NotNil(t, b)
	NotEqual(t, a, b)
	// Equal(t, a, b)
}

func TestMatch(t *testing.T) {
	Match(t, errors.New("Hello World"), `[Hh]ello\s+[Ww]orld`)
	NotMatch(t, errors.New("Are You OK?"), `You\s{2}`)
}
