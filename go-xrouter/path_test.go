// path_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-04-08
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-04-08

package xrouter

import (
	"fmt"
	"github.com/X-Plan/xgo/go-xassert"
	"path"
	"runtime"
	"testing"
)

var cleanTests = []struct {
	path, result string
}{
	// Already clean
	{"/", "/"},
	{"/abc", "/abc"},
	{"/a/b/c", "/a/b/c"},
	{"/abc/", "/abc/"},
	{"/a/b/c/", "/a/b/c/"},

	// missing root
	{"", "/"},
	{"abc", "/abc"},
	{"abc/def", "/abc/def"},
	{"a/b/c", "/a/b/c"},

	// Remove doubled slash
	{"//", "/"},
	{"/abc//", "/abc/"},
	{"/abc/def//", "/abc/def/"},
	{"/a/b/c//", "/a/b/c/"},
	{"/abc//def//ghi", "/abc/def/ghi"},
	{"//abc", "/abc"},
	{"///abc", "/abc"},
	{"//abc//", "/abc/"},

	// Remove . elements
	{".", "/"},
	{"./", "/"},
	{"/abc/./def", "/abc/def"},
	{"/./abc/def", "/abc/def"},
	{"/abc/.", "/abc/"},

	// Remove .. elements
	{"..", "/"},
	{"../", "/"},
	{"../../", "/"},
	{"../..", "/"},
	{"../../abc", "/abc"},
	{"/abc/def/ghi/../jkl", "/abc/def/jkl"},
	{"/abc/def/../ghi/../jkl", "/abc/jkl"},
	{"/abc/def/..", "/abc"},
	{"/abc/def/../..", "/"},
	{"/abc/def/../../..", "/"},
	{"/abc/def/../../..", "/"},
	{"/abc/def/../../../ghi/jkl/../../../mno", "/mno"},

	// Combinations
	{"abc/./../def", "/def"},
	{"abc//./../def", "/def"},
	{"abc/../../././../def", "/def"},
}

func TestPathClean(t *testing.T) {
	for _, test := range cleanTests {
		xassert.Equal(t, CleanPath(test.path), test.result)
		xassert.Equal(t, CleanPath(test.result), test.result)
	}
}

func TestPathCleanMallocs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping malloc count in short mode")
	}
	if runtime.GOMAXPROCS(0) > 1 {
		t.Log("skipping AllocsPerRun checks; GOMAXPROCS>1")
		return
	}

	for _, test := range cleanTests {
		xassert.IsTrue(t, int(testing.AllocsPerRun(100, func() { CleanPath(test.result) })) == 0)
	}
}

func BenchmarkCleanPath(b *testing.B) {
	p := "/hello/world/my/name/../is/foo/../how/./old/./../are/you"
	b.Run(fmt.Sprintf("CleanPath: %s", CleanPath(p)), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			CleanPath(p)
		}
	})

	b.Run(fmt.Sprintf("path.Clean: %s", path.Clean(p)), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			path.Clean(p)
		}
	})
}
