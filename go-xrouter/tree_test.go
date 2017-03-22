// tree_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-03-16
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-03-22

package xrouter

import (
	"fmt"
	"github.com/X-Plan/xgo/go-xassert"
	"net/http"
	"strings"
	"testing"
)

// Test 'node.add' function, but doesn't include the error case.
func TestAddCorrect(t *testing.T) {
	var paths = []string{
		"/who/are/you/?",
		"/who/is/:she",
		"/how/are/*you",
		"/where/is/:xxx/*she",
		"/how/old/are/you/",
		"/root/who/are/you/",
		"/root/how/are/you",
		"/root/how/are/you/my/friend",
		"/root/who/is/he",
		"/what/you/want/to/:do/",
		"/can/you/tell/me/what/:be/:possession/*favorite",
		"/could/you/take/a/pass/at/this/implementation",
		"/what's/your/favorite/?/If you known, please/tell me.",
		"/一花一世界/一叶一菩提/", // We don't use chinese in url path. :)
		"/一叶:障目",
	}

	n, handle := &node{}, func(http.ResponseWriter, *http.Request, XParams) {}
	for _, path := range paths {
		xassert.IsNil(t, n.add(path, handle))
	}
	xassert.Equal(t, int(n.priority), len(paths))
	printNode(n, 0)
}

// Test 'node.construct' function, and the 'path' is valid.
func TestConstructCorrect(t *testing.T) {
	var paths = []string{
		"/who/are/you/?",
		"/",
		":hello/world/path",
		"he:llo/world/*path",
		"he:llo/world/*path",
		"he:llo/world/pa*th",
		"h:ello/wo:rld/path/",
		"h:ello/wo:rld/pa:th/",
		"h:ello/w:orld/pa:th",
		"h:ello/w:orld/path",
	}

	for _, path := range paths {
		n, handle := &node{}, func(http.ResponseWriter, *http.Request, XParams) {}
		xassert.IsNil(t, n.construct(path, "full path", handle))
		printNode(n, 0)
	}
}

// Test 'node.construct' function, but the 'path' is invalid.
func TestConstructError(t *testing.T) {
	var paths = []struct {
		path   string
		reason string
	}{
		{"/who/are/y*ou/", `'\*ou' in path 'full path': catch-all routes are only allowed at the end of the path`},
		{"/who/are/you*", `'\*' in path 'full path': catch-all wildcard can't be empty`},
		{":hello/wor:ld/Who:are:you/Am/I/alone", `':are:you/Am/I/alone' in path 'full path': only one wildcard per path segment is allowed`},
		{"hello/wo:rld*xxx/aaa", `':rld\*xxx/aaa' in path 'full path': only one wildcard per path segment is allowed`},
		{"hello/wo*rld:xxx/aaa", `'\*rld' in path 'full path': catch-all routes are only allowed at the end of the path`},
		{"hello:/world", `':/world' in path 'full path': param wildcard can't be empty`},
	}

	for _, p := range paths {
		n, handle := &node{}, func(http.ResponseWriter, *http.Request, XParams) {}
		xassert.Match(t, n.construct(p.path, "full path", handle), p.reason)
	}
}

func TestSplit(t *testing.T) {
	// The 'original' field and the 'n.path' need have a common
	// prefix, but it should be the substring of the 'n.path'
	// (can't be equal to the 'n.path'), otherwise the initial
	// condition of the 'n.split' function can't be statisfied.
	var examples = [][]struct {
		original string
		rest     string
		priority int
		handle   XHandle
	}{
		{{"begin/who/is/she", "is/she", 1, nil}},
		{{"begin/who/are you?", " you?", 1, nil}},
		{{"begin/how/are/you/?", "how/are/you/?", 1, nil}},
		{{"begin/who/a", "", 2, nil}},
		{{"begin/who", "", 2, nil}},
		{{"begin/", "", 2, nil}},
		{{"begin/who/is/he", "is/he", 1, nil}, {"begin/who", "", 2, nil}, {"beg", "", 3, nil}},
	}

	var (
		n      *node
		handle = func(http.ResponseWriter, *http.Request, XParams) {}
	)

	for _, e := range examples {
		n = &node{}
		xassert.IsNil(t, n.construct("begin/who/are/you/?/", "full path", handle))
		for _, p := range e {
			p.handle = func(http.ResponseWriter, *http.Request, XParams) {}
			xassert.Match(t, n.split(nil, lcp(p.original, n.path), p.original, p.handle), p.rest)
			xassert.Equal(t, int(n.priority), p.priority)
			printNode(n, 0)
		}
	}
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
