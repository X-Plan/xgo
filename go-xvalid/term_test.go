// term_test.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-01-08
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-01-08
package xvalid

import (
	"fmt"
	"github.com/X-Plan/xgo/go-xassert"
	rft "reflect"
	"testing"
)

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
