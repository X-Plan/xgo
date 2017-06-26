// xrouter_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-06-26
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-06-26
package xrouter

import (
	"fmt"
	"github.com/X-Plan/xgo/go-xassert"
	"testing"
)

func TestXParam(t *testing.T) {
	xp1 := XParam{Key: "Hello", Value: "World"}
	xassert.Equal(t, fmt.Sprintf("%s", xp1), "Hello=World")
	xp2 := XParam{Key: "Foo", Value: "Bar"}
	xassert.Equal(t, fmt.Sprintf("%s", xp2), "Foo=Bar")
	xassert.IsFalse(t, xp1.Equal(xp2))
	xp3 := xp1
	xassert.IsTrue(t, xp1.Equal(xp3))
}

func TestXParams(t *testing.T) {
	xps1 := XParams{
		{"Hello", "World"},
		{"Who", "Are"},
		{"Foo", "Bar"},
	}

	xassert.Equal(t, fmt.Sprintf("%s", xps1), "Hello=World,Who=Are,Foo=Bar")
	for _, xp := range xps1 {
		xassert.Equal(t, xps1.Get(xp.Key), xp.Value)
	}
	xassert.Equal(t, xps1.Get("World"), "")

	xps2 := xps1
	xassert.IsTrue(t, xps1.Equal(xps2))

	// The order of the elements should be same.
	xps3 := make(XParams, len(xps1))
	copy(xps3, xps1)
	xps3[0], xps3[1] = xps3[1], xps3[0]
	xassert.IsFalse(t, xps1.Equal(xps3))

	xps4 := xps1[0 : len(xps1)-1]
	xassert.IsFalse(t, xps1.Equal(xps4))

	xps5 := append(xps1, XParam{"A", "B"})
	xassert.IsFalse(t, xps1.Equal(xps5))
}

func TestSupportMethod(t *testing.T) {
	var methodPair = []struct {
		method string
		ok     bool
	}{
		{"get", true},
		{"pOst", true},
		{"HEAD", true},
		{"puT", true},
		{"options", true},
		{"PATCh", true},
		{"DELETE", true},
		{"CONNECT", false},
		{"foo", false},
		{"BAR", false},
	}

	for _, pair := range methodPair {
		xassert.Equal(t, SupportMethod(pair.method), pair.ok)
	}
}
