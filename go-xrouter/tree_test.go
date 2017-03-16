// tree_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-03-16
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-03-16

package xrouter

import (
	"fmt"
	"github.com/X-Plan/xgo/go-xassert"
	"net/http"
	"strings"
	"testing"
)

// Test 'node.construct' function, but the 'path' is valid.
func TestConstructCorrect(t *testing.T) {
	n, handle := &node{}, func(http.ResponseWriter, *http.Request, XParams) {}
	xassert.IsNil(t, n.construct("/who/are/you/?", "full path", handle))
	printNode(n, 0)

	n = &node{}
	xassert.IsNil(t, n.construct(":hello/world/path", "full path", handle))
	printNode(n, 0)

	n = &node{}
	xassert.IsNil(t, n.construct("he:llo/world/*path", "full path", handle))
	printNode(n, 0)

	n = &node{}
	xassert.IsNil(t, n.construct("h:ello/wo:rld/path/", "full path", handle))
	printNode(n, 0)

	n = &node{}
	xassert.IsNil(t, n.construct("h:ello/wo:rld/pa:th/", "full path", handle))
	printNode(n, 0)
}

// Print node in tree-text format.
func printNode(n *node, depth int) {
	if n == nil {
		return
	}

	if depth == 0 {
		fmt.Println("")
	}

	fmt.Printf("%s%s [%c] (%s:%v:%d) [%v] \n", strings.Repeat(" ", depth), n.path, n.index, n.nt, n.tsr, n.priority, n.handle)
	for _, child := range n.children {
		printNode(child, depth+len(n.path))
	}
}
