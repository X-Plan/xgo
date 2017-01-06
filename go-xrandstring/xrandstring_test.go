// xrandstring_test.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-01-07
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-01-07
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
