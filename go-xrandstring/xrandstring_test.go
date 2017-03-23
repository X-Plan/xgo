// xrandstring_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-01-07
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-03-23
package xrandstring

import (
	"fmt"
	"github.com/X-Plan/xgo/go-xassert"
	"strings"
	"testing"
)

func TestReplace(t *testing.T) {
	// The character of the 'str' isn't included in 'LetterBytes'.
	str := "!@#$%^"
	for n := 1; n <= 1000; n++ {
		oldstr := Get(n)
		newstr := Replace(oldstr, str)
		fmt.Println(newstr)
		xassert.Equal(t, len(oldstr), len(newstr))

		if n >= len(str) {
			i := strings.Index(newstr, str)
			xassert.IsTrue(t, i != -1 && i < n-len(str)+1)
		} else {
			xassert.Equal(t, newstr, oldstr)
		}
	}
}

func BenchmarkGet16(b *testing.B) {
	benchmarkGet(b, 16)
}

func BenchmarkGet256(b *testing.B) {
	benchmarkGet(b, 256)
}

func BenchmarkGet1024(b *testing.B) {
	benchmarkGet(b, 1024)
}

func benchmarkGet(b *testing.B, n int) {
	for i := 0; i < b.N; i++ {
		Get(n)
	}
}
