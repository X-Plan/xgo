// term_test.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-01-08
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-01-09
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
	xassert.IsNil(t, cpanic(func() { newTerm("test", "noempty", "") }))
	xassert.IsNil(t, cpanic(func() { newTerm("test", "noempty", "\n\t \t\n\r\v") }))
	xassert.Match(t, cpanic(func() { newTerm("test", "noempty", "hello") }), `invalid term 'noempty=.*'`)
	xassert.Match(t, cpanic(func() { newTerm("test", "min", "true") }), `invalid term 'min=true'`)
	xassert.Match(t, cpanic(func() { newTerm("test", "max", "FALSE") }), `invalid term 'max=FALSE'`)
	xassert.IsNil(t, cpanic(func() { newTerm("test", "default", "True") }))
	_ = newTerm("test", "min", "10").v.(uint64)
	_ = newTerm("test", "max", "-10").v.(int64)
	_ = newTerm("test", "min", "-12.33").v.(float64)
	_ = newTerm("test", "default", "1").v.(uint64)
	_ = newTerm("test", "default", "0").v.(uint64)
	_ = newTerm("test", "min", "10h").v.(time.Duration)
	xassert.Match(t, cpanic(func() { newTerm("test", "min", "hello world") }), `invalid term 'min=hello world'`)
	xassert.Match(t, cpanic(func() { newTerm("test", "max", "[1,2,3]") }), `invalid term 'max=\[1,2,3\]'`)
	_ = newTerm("test", "default", `{ "a": 1, "b": "hello" }`).v.(string)
}

func TestNewTerms(t *testing.T) {
	xassert.Match(t, cpanic(func() { newTerms("test", "min=1,max=10,default=5, max = 7") }), `duplicate term 'max'`)
	xassert.Match(t, cpanic(func() { newTerms("test", "   noempty, min = 10, , , noempty ") }), `duplicate term 'noempty'`)
	tms := newTerms("test", "  noempty , min = 10, max = 30.7, default=-10 ")
	xassert.Equal(t, len(tms), 4)
	xassert.Equal(t, tms[0].t, tnoempty)
	xassert.Equal(t, tms[1].t, tmin)
	xassert.Equal(t, tms[2].t, tmax)
	xassert.Equal(t, tms[3].t, tdefault)
	tms = newTerms("test", ",,,noempty  ,   ,    ,")
	xassert.Equal(t, len(tms), 1)
	xassert.Equal(t, tms[0].t, tnoempty)
	xassert.Match(t, cpanic(func() { newTerms("test", " noempty =   min = 10   ") }), `invalid term 'noempty=min = 10'`)
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
	tm, d := term{}, dummy{}
	xassert.IsTrue(t, tm.iszero(rft.ValueOf(d)))
}

func TestIsZeroFalseA(t *testing.T) {
	tm, d := term{}, dummy{A: dummyUint{A: uint(12)}}
	xassert.IsFalse(t, tm.iszero(rft.ValueOf(d)))
}

func TestIsZeroFalseB(t *testing.T) {
	tm, d := term{}, dummy{B: dummyInt{A: int(-12)}}
	xassert.IsFalse(t, tm.iszero(rft.ValueOf(d)))
}

func TestIsZeroFalseC(t *testing.T) {
	tm, d := term{}, dummy{C: dummyFloat{B: float64(1.21)}}
	xassert.IsFalse(t, tm.iszero(rft.ValueOf(d)))
}

func TestIsZeroTrueD(t *testing.T) {
	tm, d := term{}, dummy{D: make(map[string]string)}
	xassert.IsTrue(t, tm.iszero(rft.ValueOf(d)))
}

func TestIsZeroFalseD(t *testing.T) {
	tm, d := term{}, dummy{D: map[string]string{"hello": "world"}}
	xassert.IsFalse(t, tm.iszero(rft.ValueOf(d)))
}

func TestIsZeroTrueE(t *testing.T) {
	tm, d := term{}, dummy{E: make([]string, 0, 16)}
	xassert.IsTrue(t, tm.iszero(rft.ValueOf(d)))
}

func TestIsZeroFalseE(t *testing.T) {
	tm, d := term{}, dummy{E: []string{"hello", "world"}}
	xassert.IsFalse(t, tm.iszero(rft.ValueOf(d)))
}

func TestIsZeroFalseG(t *testing.T) {
	tm, d := term{}, dummy{G: 20}
	xassert.IsFalse(t, tm.iszero(rft.ValueOf(d)))
}

func TestIsZeroFalseH(t *testing.T) {
	var i int = 10
	tm, d := term{}, dummy{H: &i}
	xassert.IsFalse(t, tm.iszero(rft.ValueOf(d)))
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