// xrouter_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-03-23
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-03-23

package xrouter

import (
	"fmt"
	"github.com/X-Plan/xgo/go-xassert"
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
