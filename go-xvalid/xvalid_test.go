// xvalid_test.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-01-10
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-01-27
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

type joke struct {
	ignore string         `xvalid:"hello"`
	Array  [3]string      `xvalid:"default=world"`
	Map    map[string]int `xvalid:"imin=12.7"`
}

func TestIndirect(t *testing.T) {
	var a = struct {
		Bool   *bool          `xvalid:"idefault=true"`
		Int    *int           `xvalid:"idefault=12"`
		Float  *float64       `xvalid:"idefault=19"`
		String *string        `xvalid:"idefault=hello world"`
		Array  *[3]string     `xvalid:"idefault=who are you?"`
		Slice  []uint32       `xvalid:"idefault=10"`
		Map    map[string]int `xvalid:"imax=300"`
	}{
		Bool:   new(bool),
		Int:    new(int),
		Float:  new(float64),
		String: new(string),
		Array:  new([3]string),
		Slice:  make([]uint32, 10),
		Map:    map[string]int{"hello": 100, "world": 400},
	}
	xassert.Match(t, Validate(&a), `Map\[world\]: can't satisfy term 'imax=300'`)
	xassert.IsTrue(t, *a.Bool)
	xassert.Equal(t, *a.Int, 12)
	xassert.Equal(t, *a.Float, float64(19))
	xassert.Equal(t, *a.String, "hello world")
	for _, v := range *a.Array {
		xassert.Equal(t, v, "who are you?")
	}
	for _, v := range a.Slice {
		xassert.Equal(t, v, uint32(10))
	}

	var tmpSlice = make([]string, 10)
	var b = struct {
		Slice *[]string `xvalid:"idefault=hello"`
	}{
		Slice: &tmpSlice,
	}
	xassert.Match(t, cpanic(func() { Validate(&b) }), `Slice: slice type can't support 'idefault' term`)

	var tmpMap = make(map[string]int)
	var c = struct {
		Map *map[string]int `xvalid:"imax=10"`
	}{
		Map: &tmpMap,
	}
	xassert.Match(t, cpanic(func() { Validate(&c) }), `Map: map type can't support 'imax' term`)

	var d = struct {
		Struct *struct {
			Foo string `xvalid:"noempty"`
		}
	}{Struct: nil}
	xassert.IsNil(t, Validate(&d))

	var e = struct {
		Int   *int             `xvalid:"imin=100"`
		Dummy map[string]*joke `xvalid:"noempty,inoempty"`
	}{
		Dummy: map[string]*joke{
			"foo": &joke{},
			"bar": &joke{
				Map: map[string]int{"hello": 13, "world": 12},
			},
		},
	}
	xassert.Match(t, Validate(&e), `Dummy\[bar\].Map\[world\]: can't satisfy term 'imin=12.7'`)

	var f = make([]joke, 10)
	f[5].Map = map[string]int{"kkk": 10}
	xassert.Match(t, Validate(&f), `\[5\].Map\[kkk\]: can't satisfy term 'imin=12.7'`)

	var g = make(map[float64]joke)
	g[3.3] = joke{Map: map[string]int{"bbb": 9}}
	xassert.Match(t, Validate(&g), `\[3\.3\]\.Map\[bbb\]: can't satisfy term 'imin=12.7'`)
}

type person struct {
	Name    string    `xvalid:"noempty"`
	Age     int       `xvalid:"min=1,max=200"`
	Tel     []string  `xvalid:"imatch=/^[[:digit:]]{2\,3}-[[:digit:]]+$/"`
	Friends []*person `xvalid:"inoempty"`
}

type foo struct {
	A *string `xvalid:"inoempty"`
}

type foo1 struct {
	A [3]int `xvalid:"idefault=10"`
}

type bar struct {
	A *[3]int `xvalid:"idefault=10"`
}

type foo2 struct {
	A uint8         `xvalid:"default=128"`
	B uint16        `xvalid:"max=123456789"`
	C int32         `xvalid:"min=-1234567"`
	D float64       `xvalid:"default=-123.4567"`
	F bool          `xvalid:"default=True"`
	G string        `xvalid:"default=123456789"`
	H time.Duration `xvalid:"default=20h"`
}

type foo3 struct {
	A uint8 `xvalid:"default=1234567"` // 溢出
	B int64 `xvalid:"default=40e40"	`  // 40e40超出了int64容纳空间, 因此B的值时未知的
	C int8  `xvalid:"default=string"`  // 字符串不能赋值个int8
}

func TestExample(t *testing.T) {
	p := &person{
		Name: "blinklv",
		Age:  20,
		Tel: []string{
			"086-123456789",
			"079-123456789",
		},
		Friends: []*person{
			&person{
				Name: "luna",
				Age:  21,
				Tel:  []string{"086-11111111"},
			},
		},
	}
	xassert.IsNil(t, Validate(&p))

	f1 := &foo{A: nil}
	xassert.IsNil(t, Validate(&f1))
	str := ""
	f2 := &foo{A: &str}
	xassert.NotNil(t, Validate(&f2))

	f := &foo1{}
	xassert.NotNil(t, cpanic(func() { Validate(f) }))

	b := &bar{}
	xassert.IsNil(t, Validate(b))

	b.A = &[3]int{}
	xassert.IsNil(t, Validate(b))
	for _, v := range *(b.A) {
		xassert.Equal(t, v, 10)
	}

	c := &foo2{}
	xassert.IsNil(t, Validate(c))

	d := &foo3{}
	xassert.NotNil(t, cpanic(func() { Validate(d) }))
}

type renameStruct1 struct {
	D renameStruct2 `xvalid:"noempty" xname:"xnamed"`
}

type renameStruct2 struct {
	E []renameStruct3 `xvalid:"inoempty" json:"jsone"`
}

type renameStruct3 struct {
	F map[string]int `xvalid:"imin=5" yaml:"yamlf"`
}

func TestFieldRename(t *testing.T) {
	var a = struct {
		A int `xvalid:"min=20" xname:"xnamea" json:"jsona" yaml:"yamla"`
	}{A: 10}
	xassert.Match(t, Validate(&a), `xnamea: can't satisfy term 'min=20'`)

	var b = struct {
		B int `xvalid:"min=20" xname:"" json:"jsonb" yaml:"yamlb"`
	}{B: 10}
	xassert.Match(t, Validate(&b), `jsonb: can't satisfy term 'min=20'`)

	var c = struct {
		C int `xvalid:"max=5" json:"" yaml:"yamlc"`
	}{C: 10}
	xassert.Match(t, Validate(&c), `yamlc: can't satisfy term 'max=5'`)

	var d = renameStruct1{
		D: renameStruct2{
			E: []renameStruct3{
				renameStruct3{F: map[string]int{"foo": 1}},
			},
		},
	}

	xassert.Match(t, Validate(&d), `xnamed.jsone\[0\].yamlf\[foo\]: can't satisfy term 'imin=5'`)
}
