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

// Test 'node.construct' function, and the 'path' is valid.
func TestConstructCorrect(t *testing.T) {
	n, handle := &node{}, func(http.ResponseWriter, *http.Request, XParams) {}
	xassert.IsNil(t, n.construct("/who/are/you/?", "full path", handle))
	printNode(n, 0)

	n = &node{}
	xassert.IsNil(t, n.construct("/", "full path", handle))
	printNode(n, 0)

	n = &node{}
	xassert.IsNil(t, n.construct(":hello/world/path", "full path", handle))
	printNode(n, 0)

	n = &node{}
	xassert.IsNil(t, n.construct("he:llo/world/*path", "full path", handle))
	printNode(n, 0)

	n = &node{}
	xassert.IsNil(t, n.construct("he:llo/world/pa*th", "full path", handle))
	printNode(n, 0)

	n = &node{}
	xassert.IsNil(t, n.construct("h:ello/wo:rld/path/", "full path", handle))
	printNode(n, 0)

	n = &node{}
	xassert.IsNil(t, n.construct("h:ello/wo:rld/pa:th/", "full path", handle))
	printNode(n, 0)
}

// Test 'node.construct' function, but the 'path' is invalid.
func TestConstructError(t *testing.T) {
	n, handle := &node{}, func(http.ResponseWriter, *http.Request, XParams) {}
	xassert.Match(t, n.construct("/who/are/y*ou/", "full path", handle),
		`'\*ou' in path 'full path': catch-all routes are only allowed at the end of the path`)

	n = &node{}
	xassert.Match(t, n.construct("/who/are/you*", "full path", handle),
		`'\*' in path 'full path': catch-all wildcard can't be empty`)

	n = &node{}
	xassert.Match(t, n.construct(":hello/wor:ld/Who:are:you/Am/I/alone", "full path", handle),
		`':are:you/Am/I/alone' in path 'full path': only one wildcard per path segment is allowed`)

	n = &node{}
	xassert.Match(t, n.construct("hello/wo:rld*xxx/aaa", "full path", handle),
		`':rld\*xxx/aaa' in path 'full path': only one wildcard per path segment is allowed`)

	n = &node{}
	xassert.Match(t, n.construct("hello/wo*rld:xxx/aaa", "full path", handle),
		`'\*rld' in path 'full path': catch-all routes are only allowed at the end of the path`)

	n = &node{}
	xassert.Match(t, n.construct("hello:/world", "full path", handle),
		`':/world' in path 'full path': param wildcard can't be empty`)
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
