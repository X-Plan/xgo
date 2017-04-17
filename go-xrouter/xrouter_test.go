// xrouter_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-03-23
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-04-17

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

func TestHandle(t *testing.T) {
	xr := New(&XConfig{})
	for _, method := range methods {
		paths, _ := generatePaths(100, 3, 6)
		for _, path := range paths {
			xassert.IsNil(t, xr.Handle(method, path, generateXHandle(path)))
		}
	}

	// Test the path duplicate case, which is produced by CleanPath function.
	xassert.IsNil(t, xr.Handle("POST", "/hello/world/", generateXHandle("/hello/world/")))
	xassert.NotNil(t, xr.Handle("post", "/hello/xxx/../world/", generateXHandle("/hello/xxx/../world/")))
	xassert.IsNil(t, xr.Handle("GET", "/hello/xxx/../world/", generateXHandle("/hello/xxx/../world/")))

	// Test the unsupported methods.
	ums := []string{"POXX", "GEX", "XATCH", "FOO", "bar"}
	for _, um := range ums {
		xassert.Match(t, xr.Handle(um, "/foo", nil), fmt.Sprintf(`http method \(%s\) is unsupported`, um))
	}
}

func TestServeHTTP(t *testing.T) {
	setupServer()
	setupClient(t)
}

func setupServer() {
}

func setupClient(t *testing.T) {
}
