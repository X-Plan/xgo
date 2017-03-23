// xrandstring_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-01-07
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-03-23
package xrandstring

import (
	"testing"
)

func TestDummy(t *testing.T) {
	// Nothing.
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
