// xvalid_test.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-01-10
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-01-15
package xvalid

import (
	"github.com/X-Plan/xgo/go-xassert"
	"testing"
	"time"
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

	var c = struct {
		Int int `xvalid:"min=23,max=1.78e6"`
	}{Int: 10000000}
	xassert.Match(t, Validate(&c), "can't satisfy term 'max=.*'")

	var d = struct {
		Int int `xvalid:"min=999999,max=1000001,default=1ms"`
	}{}
	xassert.IsNil(t, Validate(&d))

	var e = struct {
		Float float32 `xvalid:"noempty"`
	}{}
	xassert.Match(t, Validate(&e), `Float: is empty`)

	var f = struct {
		Float float32 `xvalid:"default=4e40,max=4e39"`
	}{}
	xassert.Match(t, cpanic(func() { Validate(&f) }), `term '.*' and term '.*' are contradictory`)

	var g = struct {
		Uint uint32 `xvalid:"default=4294967296"`
	}{}
	xassert.Match(t, cpanic(func() { Validate(&g) }), `value of term '.*' overflow uint32`)

	var h = struct {
		Uint uint32 `xvalid:"default=4e40"`
	}{}
	xassert.Match(t, cpanic(func() { Validate(&h) }), `value of term '.*' overflow uint32`)

	var i = struct {
		Int int32 `xvalid:"default=2147483648"`
	}{}
	xassert.Match(t, cpanic(func() { Validate(&i) }), `value of term '.*' overflow int32`)

	var j = struct {
		Float float32 `xvalid:"default=4e40"`
	}{}
	xassert.Match(t, cpanic(func() { Validate(&j) }), `value of term '.*' overflow float32`)

	var k = struct {
		Int int64 `xvalid:"default=False"`
	}{}
	xassert.Match(t, cpanic(func() { Validate(&k) }), `value of term '.*' not match int64 type`)
}

func TestString(t *testing.T) {
	var a = struct {
		String string `xvalid:"default=TRUE"`
	}{}
	xassert.IsNil(t, Validate(&a))
	xassert.Equal(t, a.String, "TRUE")

	var b = struct {
		String string `xvalid:"default=10"`
	}{}
	xassert.IsNil(t, Validate(&b))
	xassert.Equal(t, b.String, "10")

	var c = struct {
		String string `xvalid:"default=-100"`
	}{}
	xassert.IsNil(t, Validate(&c))
	xassert.Equal(t, c.String, "-100")

	var d = struct {
		String string `xvalid:"default=17.2323e10"`
	}{}
	xassert.IsNil(t, Validate(&d))
	xassert.Equal(t, d.String, "17.2323e10")

	var e = struct {
		String string `xvalid:"default=10h40m"`
	}{}
	xassert.IsNil(t, Validate(&e))
	xassert.Equal(t, e.String, "10h40m")

	var f = struct {
		String string `xvalid:"default=/hello world /"`
	}{}
	xassert.IsNil(t, Validate(&f))
	xassert.Equal(t, f.String, "/hello world /")

	var g = struct {
		String string `xvalid:"min=10"`
	}{}
	xassert.Match(t, cpanic(func() { Validate(&g) }), `type can't support 'min' term`)

	var h = struct {
		String string `xvalid:"max=200.7"`
	}{}
	xassert.Match(t, cpanic(func() { Validate(&h) }), `type can't support 'max' term`)

	var i = struct {
		String string `xvalid:"noempty"`
	}{}
	xassert.Match(t, Validate(&i), `is empty`)

	var j = struct {
		String string `xvalid:"match=/ hello world$/"`
	}{String: "hello world"}
	xassert.Match(t, Validate(&j), `not match`)

	var k = struct {
		String string `xvalid:"min=10,max=200,default=TRUE"`
	}{}
	xassert.Match(t, cpanic(func() { Validate(&k) }), `term 'default=true' and term 'min=10' are contradictory`)
}

func TestDuration(t *testing.T) {
	var a = struct {
		Duration time.Duration `xvalid:"min=1s,max=1h,default=1m"`
	}{}
	xassert.IsNil(t, Validate(&a))

	var b = struct {
		Duration time.Duration `xvalid:"min=1s,max=200h,default=hello world"`
	}{}
	xassert.Match(t, cpanic(func() { Validate(&b) }), `term 'default=hello world' and term 'min=1s' are contradictory`)

	var c = struct {
		Duration time.Duration `xvalid:"min=21h12m31s,max=39h12s"`
	}{Duration: 20 * time.Hour}
	xassert.Match(t, Validate(&c), "can't satisfy term 'min=.*'")
}

func TestArray(t *testing.T) {
	var a = struct {
		Array [8]string `xvalid:"min=100"`
	}{}
	xassert.Match(t, cpanic(func() { Validate(&a) }), `Array\[0\]: string type can't support 'min' term`)

	var b = struct {
		Array [8]int `xvalid:"min=10,default=17.2"`
	}{Array: [8]int{13}}
	xassert.IsNil(t, Validate(&b))
	for i, v := range b.Array {
		if i == 0 {
			xassert.Equal(t, v, 13)
		} else {
			xassert.Equal(t, v, 17)
		}
	}

	var c = struct {
		Array [8]int `xvalid:"min=15.2,default=20.2"`
	}{Array: [8]int{0, 0, 0, 13}}
	xassert.Match(t, Validate(&c), `Array\[3\]: can't satisfy term 'min=15.2'`)

	var d = struct {
		Array [8]string `xvalid:"match=/hello[[:space:]]+world/"`
	}{Array: [8]string{"hello world", "hello  world", "helloworld"}}
	xassert.Match(t, Validate(&d), `Array\[2\]: 'helloworld' not match '.*'`)

	var e = struct {
		Array [8][8]string `xvalid:"match=/hello[[:space:]]+world/,default=hello world"`
	}{Array: [8][8]string{
		[8]string{},
		[8]string{},
		[8]string{},
		[8]string{"hello world", "hello  world", "helloworld"},
	}}
	xassert.Match(t, Validate(&e), `Array\[3\]\[2\]: 'helloworld' not match '.*'`)

	var f = struct {
		Array [8]int `xvalid:"noempty"`
	}{Array: [8]int{0, 0, 0, 1}}
	xassert.IsNil(t, Validate(&f))
}

func TestPointer(t *testing.T) {
	var a = struct {
		Pointer *int `xvalid:"noempty"`
	}{}
	xassert.Match(t, Validate(&a), `Pointer: is empty`)
}
