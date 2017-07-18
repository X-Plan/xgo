// tree_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-06-13
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-07-18
package xrouter

import (
	"fmt"
	"github.com/X-Plan/xgo/go-xassert"
	"github.com/X-Plan/xgo/go-xrandstring"
	"math/rand"
	"strings"
	"testing"
)

var paths = []struct {
	path string
	ok   bool
}{
	{"", false}, // path argument is empty
	{"/", true},
	{"/get/user/scheme", true},
	{"/get/user/scheme", false}, // path has already been registered
	{"/get/user/count", true},
	{"/get/user/info/:name", true},
	{"/get/user/info/:nick", false}, // coflict with existing param wildcard
	{"/get/user/info/:name/sex", true},
	{"/get/user/info/:name/:property", false},      // wildcard conflict with the existing path segment
	{"/get/user/info/:name/friends/*name/", false}, // catch-all routes are only allowed at the end of the path
	{"/add/user/info/:name/friends/*/", false},     // ditto
	{"/get/user/info/:name/friends/*name", true},
	{"/add/user/:", false},     // param wildcard can't be empty
	{"/add/user/:/foo", false}, // ditto
	{"/add/user/*", false},     // ditto
	{"/add/user/", true},
	{"/add/user/:a/:b/", true},
	{"/add/user/:a/:b/:c/:d/*e", true},
	{"/add/user/:a/:b/:c/:d/hello", false}, // conflict with the existing catch-all wildcard
	{"/add/user/:name", false},             // path has already been registered
	{"/del/user/:hell:o/world", false},     // only one wildcard per path segment is allowed
	{"/del/user/:he*llo", false},           // ditto
	{"/del/user/he:l:l:o/world", false},    // ditto
	{"/del/user/:name/information", true},
	{"/del/user/:name/info:foo:bar", false}, // trigger split operation
	{"/del/user/:name/info/sex", true},
	{"/del/user/:name", true},
	{"/update/:a/b/:c/d/:e/f/:g/h/:i/j/:k/l/*m", true},
	{"/update/:a/b/:c/d/:e/f/:g/h/:i/j/:k/l/foo", false},
	{"/update/:a/b/:c/d/:e/f/:g/h/:i/j/:k/l/", true},
	{"/update/:a/b/:c/d/:e/f/:g/h/:i/j/:k/l", true},
	{"/update/:a/b/:c/d/:e/f/:g/h/:i/j/:k/lmn", true},
	{"/update/:a/b/:c/d/:e/f/:g/h/:i/j/:k/", true},
	{"/update/:a/b/:c/d/:e/f/:g/h/:i/j/:k", true},
}

func TestAdd(t *testing.T) {
	var n = &node{}
	for _, p := range paths {
		if err := n.add(p.path, p.path, generateHandle("GET", p.path)); err != nil {
			if !p.ok {
				// 				fmt.Println(err)
			} else {
				xassert.IsNil(t, err)
			}
		}
		// 		n.print(0)
		xassert.IsNil(t, n.check())
	}
}

func TestGet(t *testing.T) {
	var n = &node{}
	for _, p := range paths {
		if p.ok {
			xassert.IsNil(t, n.add(p.path, p.path, generateHandle("GET", p.path)))
		}
	}

	for _, p := range paths {
		if p.ok {
			for i := 0; i < 100; i++ {
				path, as := generatePath(p.path)
				h, xps, _ := n.get(path, false)
				xassert.NotNil(t, h)
				xassert.IsTrue(t, xps.Equal(as))
			}
		}
	}
}

func TestTSR(t *testing.T) {
	var (
		n                = &node{}
		independentPaths []string
	)

	for _, p := range paths {
		if p.ok {
			if h, _, tsr := n.get(p.path, true); h == nil && tsr == notRedirect {
				xassert.IsNil(t, n.add(p.path, p.path, generateHandle("GET", p.path)))
				independentPaths = append(independentPaths, p.path)
			}
		}
	}

	for _, p := range independentPaths {
		path, as := generatePath(p)
		h, xps, tsr := n.get(path, false)
		xassert.NotNil(t, h)
		xassert.IsTrue(t, xps.Equal(as))
		xassert.Equal(t, tsr, notRedirect)

		if path[len(path)-1] == '/' {
			h, _, tsr := n.get(path[:len(path)-1], true)
			xassert.IsNil(t, h)
			xassert.Equal(t, tsr, addSlash)
		} else {
			h, _, tsr := n.get(path+"/", true)
			if h != nil {
				// Must contain catch-all wildcard.
				xassert.NotEqual(t, strings.IndexByte(p, '*'), -1)
			} else {
				if tsr != removeSlash {
					n.print(0)
					fmt.Println(path + "/")
				}
				xassert.Equal(t, tsr, removeSlash)
			}
		}
	}
}

func TestRemove(t *testing.T) {
	var (
		n         = &node{}
		truePaths []string
	)

	for i := 0; i < 100; i++ {
		truePaths = nil
		for _, p := range paths {
			if p.ok {
				xassert.IsNil(t, n.add(p.path, p.path, generateHandle("GET", p.path)))
				truePaths = append(truePaths, p.path)
			}
		}

		randomSortPaths(truePaths)
		for _, tp := range truePaths {
			xassert.IsTrue(t, n.remove(tp))
			fp, _ := generatePath(tp)
			// The path parameter must match a existing path exactly.
			xassert.IsFalse(t, n.remove(fp))
			xassert.IsNil(t, n.check())
		}

		// Check the empty tree.
		xassert.Equal(t, n.priority, uint32(0))
		xassert.Equal(t, n.path, "")
		xassert.IsFalse(t, n.remove("/"+xrandstring.Get(8)))
	}
}

func TestAddAndRemove(t *testing.T) {
	var truePaths []string
	for _, p := range paths {
		if p.ok {
			truePaths = append(truePaths, p.path)
		}
	}

	for i := 0; i < 100; i++ {
		randomSortPaths(truePaths)
		for gap := 1; gap < len(truePaths); gap++ {
			n1 := &node{}
			for _, path := range truePaths {
				xassert.IsNil(t, n1.add(path, path, generateHandle("GET", path)))
			}

			for j := 0; j < gap; j++ {
				xassert.IsTrue(t, n1.remove(truePaths[j]))
			}

			n2 := &node{}
			for j := gap; j < len(truePaths); j++ {
				path := truePaths[j]
				xassert.IsNil(t, n2.add(path, path, generateHandle("GET", path)))
			}

			if n1.Equal(n2) != nil {
				n1.print(0)
				n2.print(0)
			}
			xassert.IsNil(t, n1.Equal(n2))
		}
	}
}

// Replace wildcard with a random string. We think all pattern is valid.
func generatePath(pattern string) (path string, xps XParams) {
	for len(pattern) > 0 {
		var i int
		if i = strings.IndexAny(pattern, ":*"); i != -1 {
			path += pattern[:i]
			pattern = pattern[i:]
			if pattern[0] == ':' {
				if i = strings.IndexByte(pattern, '/'); i == -1 {
					i = len(pattern)
				}
				xps = append(xps, XParam{Key: pattern[1:i], Value: xrandstring.Get(8)})
				path += xps[len(xps)-1].Value
				pattern = pattern[i:]
			} else { // pattern[0] == '*'
				xps = append(xps, XParam{Key: pattern[1:], Value: xrandstring.Get(8)})
				path += xps[len(xps)-1].Value
				break
			}
		} else {
			path += pattern
			break
		}
	}
	return
}

// func generateHandle("GET",path string) XHandle {
// 	return func(w http.ResponseWriter, _ *http.Request, xps XParams) {
// 		msg := fmt.Sprintf("path: %s, params: %s", path, xps)
// 		fmt.Println(msg)
// 		if w != nil {
// 			w.Write([]byte(msg))
// 		}
// 	}
// }

// Based on Fisher-Yates shuffle algorithm.
func randomSortPaths(paths []string) {
	var n = len(paths)
	for i := 0; i < n-1; i++ {
		j := rand.Int()%(n-i) + i
		paths[i], paths[j] = paths[j], paths[i]
	}
}
