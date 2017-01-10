// xvalid_test.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-01-10
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-01-10
package xvalid

import (
	"fmt"
	"github.com/X-Plan/xgo/go-xassert"
	"testing"
)

type foo1 struct {
	A bool              `xvalid:"default=true"`
	B int               `xvalid:"min=-10,max=200, default=-5"`
	C uint              `xvalid:"default=100,min=10,max=300"`
	D float64           `xvalid:"min=-12.2,max=32.3,default=23.3"`
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
	fmt.Printf("%#v", f)
}
