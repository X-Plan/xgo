// xrouter_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-03-23
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-04-13

package xrouter

import (
	"fmt"
	"github.com/X-Plan/xgo/go-xassert"
	"github.com/X-Plan/xgo/go-xrandstring"
	"testing"
)

func TestXParam(t *testing.T) {
	xassert.IsTrue(t, fmt.Sprintf("%s", XParam{"hello", "world"}) == "hello=world")
	xassert.IsTrue(t, fmt.Sprintf("%s", XParam{"foo", "bar"}) == "foo=bar")
	xassert.IsTrue(t, fmt.Sprintf("%s", XParam{"boy", "girl"}) == "boy=girl")
	xassert.IsTrue(t, fmt.Sprintf("%s", XParam{"name", "age"}) == "name=age")
}

func TestXParams(t *testing.T) {
	xassert.IsTrue(t, fmt.Sprintf("%s", XParams{}) == "")
	xassert.IsTrue(t, fmt.Sprintf("%s", XParams{XParam{"Who", "Are"}}) == "Who=Are")
	xassert.IsTrue(t, fmt.Sprintf("%s", XParams{XParam{"Who", "Are"}, XParam{"You", "Am"}}) == "Who=Are,You=Am")
	xassert.IsTrue(t, fmt.Sprintf("%s", XParams{XParam{"Who", "Are"}, XParam{"You", "Am"}, XParam{"I", "Alone"}}) == "Who=Are,You=Am,I=Alone")
}

func TestSupportMethod(t *testing.T) {
	ms := []string{"get", "Post", "hEad", "puT", "optIons", "patch", "delete"}
	for _, m := range ms {
		xassert.IsTrue(t, SupportMethod(m))
	}

	for _, m := range methods {
		xassert.IsTrue(t, SupportMethod(m))
	}

	for _, m := range methods {
		xassert.IsFalse(t, SupportMethod(xrandstring.Replace(m, "X")))
	}
}

func TestNew(t *testing.T) {
	xassert.IsNil(t, New(nil))
	xr := New(&XConfig{})
	for _, method := range methods {
		xassert.NotNil(t, xr.trees[method])
	}
}
