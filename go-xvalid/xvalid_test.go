// xvalid_test.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-01-10
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-01-10
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

func TestConflictTerm(t *testing.T) {
	p1 := &parse1{}
	xassert.Match(t, cpanic(func() { validate(p1) }), `term 'default=-100' and term 'min=200' are contradictory`)
	p2 := &parse2{}
	xassert.Match(t, cpanic(func() { validate(p2) }), `term 'default=200.2' and term 'max=200' are contradictory`)
	p3 := &parse3{}
	xassert.Match(t, cpanic(func() { validate(p3) }), `term '.*' and term '.*' are contradictory`)
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
	A bool              `xvalid:"default=true"`
	B int               `xvalid:"min=-10,max=200, default=-5"`
	C uint              `xvalid:"min=10,max=300, default=100"`
	D float64           `xvalid:"noempty, min=-12.2,max=32.3,default=23.3"`
	E string            `xvalid:"noempty,match = /^hello[[:space:]]+world$/"`
	G []string          `xvalid:"noempty"`
	H *[3]string        `xvalid:"noempty"`
	I foo2              `xvalid:"noempty"`
	J *foo2             `xvalid:"noempty"`
	K map[string]string `xvalid:"noempty"`
}

type foo2 struct {
	A int8        `xvalid:"default=10,min=5,max=20"`
	B uint8       `xvalid:"min=10"`
	C interface{} `xvalid:"noempty"`
	D *foo3       `xvalid:"noempty"`
}

type foo3 struct {
	A bool     `xvalid:"default=true"`
	B int      `xvalid:"min=-10,max=200, default=-5"`
	C uint     `xvalid:"min=10,max=300, default=100"`
	D float64  `xvalid:"min=-12.2,max=32.3,default=23.3"`
	E string   `xvalid:"noempty,match = /^hello[[:space:]]*world$/"`
	G []string `xvalid:"noempty"`
}

func TestValidate(t *testing.T) {
	var f = &foo1{
		D: 30.42,
		E: "hello   world",
		G: []string{"Are you ok"},
		H: &[3]string{"hello", "world", "!"},
		I: foo2{
			B: 20,
			C: 10,
			D: &foo3{
				E: "hello world",
				G: []string{"hello", "world"},
			},
		},
		J: &foo2{
			B: 20,
			C: 10,
			D: &foo3{
				E: "hello world",
				G: []string{"hello", "world"},
			},
		},
		K: map[string]string{"hello": "world"},
	}

	xassert.IsNil(t, validate(f))
}
