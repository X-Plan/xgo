// tree_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-06-13
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-06-13
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
	{"/get/user/info/:name/:property", false}, // wildcard conflict with the existing path segment
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
		fmt.Println("")
	}
}
