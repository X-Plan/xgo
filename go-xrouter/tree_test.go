// tree_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-06-13
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-06-14
package xrouter

import (
	"fmt"
	"github.com/X-Plan/xgo/go-xassert"
	"net/http"
	"testing"
)

func generateHandle(path string) XHandle {
	return func(w http.ResponseWriter, _ *http.Request, xps XParams) {
		msg := fmt.Sprintf("path: %s, params: %s", path, xps)
		fmt.Println(msg)
		w.Write([]byte(msg))
	}
}

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
		if err := n.add(p.path, p.path, generateHandle(p.path)); err != nil {
			if !p.ok {
				fmt.Println(err)
			} else {
				xassert.IsNil(t, err)
			}
		}
		n.print(0)
		xassert.IsNil(t, n.checkPriority())
		xassert.IsNil(t, n.checkMaxParams())
		xassert.IsNil(t, n.checkIndex())
		fmt.Println("")
	}
}
