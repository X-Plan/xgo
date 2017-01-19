// term_test.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-01-08
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-01-19
package xvalid

import (
	"fmt"
	"github.com/X-Plan/xgo/go-xassert"
	rft "reflect"
	"regexp"
	"testing"
	"time"
)

func TestNewTerm(t *testing.T) {
	testNewTerm("", t)
	testNewTerm("i", t)
}

func testNewTerm(iprefix string, t *testing.T) {
	xassert.IsNil(t, cpanic(func() { newTerm("test", iprefix+"noempty", "") }))
	xassert.IsNil(t, cpanic(func() { newTerm("test", iprefix+"noempty", "\n\t \t\n\r\v") }))
	xassert.Match(t, cpanic(func() { newTerm("test", iprefix+"noempty", "hello") }), `invalid term '`+iprefix+`noempty=.*'`)
	xassert.Match(t, cpanic(func() { newTerm("test", iprefix+"min", "true") }), `invalid term 'min=true'`)
	xassert.Match(t, cpanic(func() { newTerm("test", iprefix+"max", "FALSE") }), `invalid term 'max=FALSE'`)
	xassert.IsNil(t, cpanic(func() { newTerm("test", iprefix+"default", "True") }))
	_ = newTerm("test", iprefix+"min", "10").v.(uint64)
	_ = newTerm("test", iprefix+"max", "-10").v.(int64)
	_ = newTerm("test", iprefix+"min", "-12.33").v.(float64)
	_ = newTerm("test", iprefix+"default", "1").v.(uint64)
	_ = newTerm("test", iprefix+"default", "0").v.(uint64)
	_ = newTerm("test", iprefix+"min", "10h").v.(time.Duration)
	xassert.Match(t, cpanic(func() { newTerm("test", iprefix+"min", "hello world") }), `invalid term 'min=hello world'`)
	xassert.Match(t, cpanic(func() { newTerm("test", iprefix+"max", "[1,2,3]") }), `invalid term 'max=\[1,2,3\]'`)
	_ = newTerm("test", iprefix+"default", `{ "a": 1, "b": "hello" }`).v.(string)

	tm := newTerm("test", iprefix+"default", "128")
	if iprefix == "" {
		xassert.Equal(t, tm.t, tdefault)
		xassert.Equal(t, tm.v, uint64(128))
		xassert.Equal(t, tm.name, "test")
	} else {
		xassert.Equal(t, tm.t, tidefault)
		xassert.Equal(t, tm.v, uint64(128))
		xassert.Equal(t, tm.name, "test")
	}

	tm = newTerm("test1", iprefix+"noempty", "")
	if iprefix == "" {
		xassert.Equal(t, tm.t, tnoempty)
		xassert.Equal(t, tm.name, "test1")
	} else {
		xassert.Equal(t, tm.t, tinoempty)
		xassert.Equal(t, tm.name, "test1")
	}

	tm = newTerm("test2", iprefix+"min", "-12.8")
	if iprefix == "" {
		xassert.Equal(t, tm.t, tmin)
		xassert.Equal(t, tm.v, -12.8)
		xassert.Equal(t, tm.name, "test2")
	} else {
		xassert.Equal(t, tm.t, timin)
		xassert.Equal(t, tm.v, -12.8)
		xassert.Equal(t, tm.name, "test2")
	}

	tm = newTerm("test3", iprefix+"max", "20.3")
	if iprefix == "" {
		xassert.Equal(t, tm.t, tmax)
		xassert.Equal(t, tm.v, 20.3)
		xassert.Equal(t, tm.name, "test3")
	} else {
		xassert.Equal(t, tm.t, timax)
		xassert.Equal(t, tm.v, 20.3)
		xassert.Equal(t, tm.name, "test3")
	}

	tm = newTerm("test4", iprefix+"match", "/hello world/")
	if iprefix == "" {
		xassert.Equal(t, tm.t, tmatch)
		xassert.Equal(t, tm.v, regexp.MustCompile(`hello world`))
		xassert.Equal(t, tm.name, "test4")
	} else {
		xassert.Equal(t, tm.t, timatch)
		xassert.Equal(t, tm.v, regexp.MustCompile(`hello world`))
		xassert.Equal(t, tm.name, "test4")
	}
}

func TestNewTerms(t *testing.T) {
	xassert.Match(t, cpanic(func() { newTerms("test", "min=1,max=10,default=5, imax = 20.3, max = 7") }), `duplicate term 'max'`)
	xassert.Match(t, cpanic(func() { newTerms("test", "   noempty, min = 10, , , noempty ") }), `duplicate term 'noempty'`)
	tms := newTerms("test", "  noempty , min = 10, max = 30.7, default=-10, imatch = /who are you.*?/ ")
	xassert.Equal(t, len(tms), 5)
	xassert.Equal(t, tms[0].t, tnoempty)
	xassert.Equal(t, tms[1].t, tmin)
	xassert.Equal(t, tms[2].t, tmax)
	xassert.Equal(t, tms[3].t, tdefault)
	tms = newTerms("test", ",,,noempty  ,   ,    ,")
	xassert.Equal(t, len(tms), 1)
	xassert.Equal(t, tms[0].t, tnoempty)
	xassert.Match(t, cpanic(func() { newTerms("test", " noempty =   min = 10, idefault = 100 ") }), `invalid term 'noempty=min = 10'`)
	xassert.Match(t, cpanic(func() { newTerms("test", " noempty, = hello, min = 12") }), `unknown term ''`)
	xassert.Match(t, cpanic(func() { newTerms("test", "=hello,match= /hello[[:space:]]*world/  ") }), `unknown term ''`)
	tms = newTerms("test", " match = / hello[[:space:]]*world/ ")
	xassert.Equal(t, len(tms), 1)
	xassert.Equal(t, tms[0].t, tmatch)
	xassert.IsTrue(t, tms[0].v.(*regexp.Regexp).MatchString(" hello   \t   world"))
}

type dummyUint struct {
	A uint
	B uint8
	C uint16
	D uint32
	E uint64
}

type dummyInt struct {
	A int
	B int8
	C int16
	D int32
	E int64
}

type dummyFloat struct {
	A float32
	B float64
}

type dummy struct {
	A dummyUint
	B dummyInt
	C dummyFloat
	D map[string]string
	E []string
	F [3]int
	G interface{}
	H *int
}

func TestIsZeroTrue(t *testing.T) {
	d := dummy{}
	xassert.IsTrue(t, iszero(rft.ValueOf(d)))
}

func TestIsZeroFalseA(t *testing.T) {
	d := dummy{A: dummyUint{A: uint(12)}}
	xassert.IsFalse(t, iszero(rft.ValueOf(d)))
}

func TestIsZeroFalseB(t *testing.T) {
	d := dummy{B: dummyInt{A: int(-12)}}
	xassert.IsFalse(t, iszero(rft.ValueOf(d)))
}

func TestIsZeroFalseC(t *testing.T) {
	d := dummy{C: dummyFloat{B: float64(1.21)}}
	xassert.IsFalse(t, iszero(rft.ValueOf(d)))
}

func TestIsZeroTrueD(t *testing.T) {
	d := dummy{D: make(map[string]string)}
	xassert.IsTrue(t, iszero(rft.ValueOf(d)))
}

func TestIsZeroFalseD(t *testing.T) {
	d := dummy{D: map[string]string{"hello": "world"}}
	xassert.IsFalse(t, iszero(rft.ValueOf(d)))
}

func TestIsZeroTrueE(t *testing.T) {
	d := dummy{E: make([]string, 0, 16)}
	xassert.IsTrue(t, iszero(rft.ValueOf(d)))
}

func TestIsZeroFalseE(t *testing.T) {
	d := dummy{E: []string{"hello", "world"}}
	xassert.IsFalse(t, iszero(rft.ValueOf(d)))
}

func TestIsZeroFalseG(t *testing.T) {
	d := dummy{G: 20}
	xassert.IsFalse(t, iszero(rft.ValueOf(d)))
}

func TestIsZeroFalseH(t *testing.T) {
	var i int = 10
	d := dummy{H: &i}
	xassert.IsFalse(t, iszero(rft.ValueOf(d)))
}

func cpanic(cb func()) (err error) {
	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("%s", v)
		}
	}()
	cb()
	return
}
