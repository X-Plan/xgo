// xvalid_test.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-01-10
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-01-14
package xvalid

import (
	"github.com/X-Plan/xgo/go-xassert"
	"testing"
)

// 测试解析阶段.
type parse1 struct {
	A int `xvalid:"min=200,default=-100"`
}

type parse2 struct {
	A int `xvalid:"max=200,default=200.2"`
}

type parse3 struct {
	A int `xvalid:"max=200,default=200.2,min=-20"`
}

type parse7 struct {
	A *int `xvalid:"imin = 100, min = 50, idefault = 80"`
}

type parse8 struct {
	A *int `xvalid:"max = 100, idefault = 80, imax = 50"`
}

type parse9 struct {
	A *int `xvalid:"imin = 100, idefault = 200, imax = 150"`
}

func TestConflictTerm(t *testing.T) {
	p1 := &parse1{}
	xassert.Match(t, cpanic(func() { Validate(p1) }), `term 'default=-100' and term 'min=200' are contradictory`)
	p2 := &parse2{}
	xassert.Match(t, cpanic(func() { Validate(p2) }), `term 'default=200.2' and term 'max=200' are contradictory`)
	p3 := &parse3{}
	xassert.Match(t, cpanic(func() { Validate(p3) }), `term '.*' and term '.*' are contradictory`)
	p7 := &parse7{}
	xassert.Match(t, cpanic(func() { Validate(p7) }), `term 'idefault=80' and term 'imin=100' are contradictory`)
	p8 := &parse8{}
	xassert.Match(t, cpanic(func() { Validate(p8) }), `term 'idefault=80' and term 'imax=50' are contradictory`)
	p9 := &parse9{}
	xassert.Match(t, cpanic(func() { Validate(p9) }), `term 'idefault=200' and term '.*' are contradictory`)
}

type parse4 struct {
	A int `xvalid:",foo=100"`
}

func TestUnknownTerm(t *testing.T) {
	p4 := &parse4{}
	xassert.Match(t, cpanic(func() { Validate(p4) }), `unknown term 'foo'`)
}

type parse5 struct {
	A string `xvalid:"match= /helloworld"`
}

type parse6 struct {
	A int `xvalid:"min=helloworld"`
}

func InvalidTerm(t *testing.T) {
	p5 := &parse5{}
	xassert.Match(t, cpanic(func() { Validate(p5) }), `invalid term 'match=/helloworld'`)
	p6 := &parse6{}
	xassert.Match(t, cpanic(func() { Validate(p6) }), `invalid term 'min=helloworld'`)
}

func TestBool(t *testing.T) {
	var x = struct {
		Bool bool `xvalid:"noempty"`
	}{Bool: true}
	xassert.Match(t, cpanic(func() { Validate(&x) }), `Bool: bool type can't support 'noempty' term`)

	var y = struct {
		Bool bool `xvalid:"default=True"`
	}{}
	xassert.IsNil(t, Validate(&y))
	xassert.IsTrue(t, y.Bool)

	var z = struct {
		Bool bool `xvalid:"min=10"`
	}{}
	xassert.Match(t, cpanic(func() { Validate(&z) }), `Bool: bool type can't support 'min' term`)

	var a = struct {
		Bool bool `xvalid:"max=17.2"`
	}{}
	xassert.Match(t, cpanic(func() { Validate(&a) }), `Bool: bool type can't support 'max' term`)

	var b = struct {
		Bool bool `xvalid:"match=/hello world/"`
	}{}
	xassert.Match(t, cpanic(func() { Validate(&b) }), `Bool: bool type can't support 'match' term`)

	var c = struct {
		Bool bool `xvalid:"idefault=true"`
	}{}
	xassert.Match(t, cpanic(func() { Validate(&c) }), `Bool: bool type can't support 'idefault' term`)
}

func TestNumber(t *testing.T) {
	var a = struct {
		Int int `xvalid:"min=-10,max=10,default=-2"`
	}{}
	xassert.IsNil(t, Validate(&a))

	var b = struct {
		Uint uint `xvalid:"min=-10,max=10,default=-2"`
	}{Uint: 5}
	xassert.IsNil(t, Validate(&b))
}
