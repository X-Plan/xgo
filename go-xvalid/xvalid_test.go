// xvalid_test.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-01-10
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-01-11
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
	xassert.Match(t, cpanic(func() { validate(p1) }), `term 'default=-100' and term 'min=200' are contradictory`)
	p2 := &parse2{}
	xassert.Match(t, cpanic(func() { validate(p2) }), `term 'default=200.2' and term 'max=200' are contradictory`)
	p3 := &parse3{}
	xassert.Match(t, cpanic(func() { validate(p3) }), `term '.*' and term '.*' are contradictory`)
	p7 := &parse7{}
	xassert.Match(t, cpanic(func() { validate(p7) }), `term 'idefault=80' and term 'imin=100' are contradictory`)
	p8 := &parse8{}
	xassert.Match(t, cpanic(func() { validate(p8) }), `term 'idefault=80' and term 'imax=50' are contradictory`)
	p9 := &parse9{}
	xassert.Match(t, cpanic(func() { validate(p9) }), `term 'idefault=200' and term '.*' are contradictory`)
}

type parse4 struct {
	A int `xvalid:",foo=100"`
}

func TestUnknownTerm(t *testing.T) {
	p4 := &parse4{}
	xassert.Match(t, cpanic(func() { validate(p4) }), `unknown term 'foo'`)
}

type parse5 struct {
	A string `xvalid:"match= /helloworld"`
}

type parse6 struct {
	A int `xvalid:"min=helloworld"`
}

func InvalidTerm(t *testing.T) {
	p5 := &parse5{}
	xassert.Match(t, cpanic(func() { validate(p5) }), `invalid term 'match=/helloworld'`)
	p6 := &parse6{}
	xassert.Match(t, cpanic(func() { validate(p6) }), `invalid term 'min=helloworld'`)
}

type foo1 struct {
	A bool           `xvalid:"default=true"`
	B uint           `xvalid:"min=10,max=100,default=50"`
	C int            `xvalid:"min=-10,max=100,noempty,default=-1"`
	D string         `xvalid:"noempty,default= hello world"`
	E [3]string      `xvalid:"default=blinklv"`
	F [3]int         `xvalid:"min=10,max=100"`
	G *[3]string     `xvalid:"idefault=x-plan"`
	H []uint         `xvalid:"imin=100, imax=200"`
	I map[string]int `xvalid:"imin=-100, imax=200"`
}

func TestValidate(t *testing.T) {
	f1 := &foo1{}
	xassert.Match(t, validate(f1), `F: can't satisfy term 'min.*'`)
	f1.A = false
	f1.E = [3]string{"hello"}
	f1.F = [3]int{10, 100, 50}
	f1.G = &[3]string{"X-Plan"}
	xassert.IsNil(t, validate(f1))
	f1.H = []uint{101, 100, 99, 200}
	xassert.Match(t, validate(f1), `.*\[2\]: can't satisfy term 'imin=100'`)
	f1.H[2] = 201
	xassert.Match(t, validate(f1), `.*\[2\]: can't satisfy term 'imax=200'`)
	f1.H[2] = 150
	f1.I = map[string]int{"hello": 200, "world": -200}
	xassert.Match(t, validate(f1), `.*\[world\]: can't satisfy term 'imin=-100'`)
}
