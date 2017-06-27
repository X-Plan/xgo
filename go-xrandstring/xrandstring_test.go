// xrandstring_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-01-07
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-06-27
package xrandstring

import (
	"fmt"
	"github.com/X-Plan/xgo/go-xassert"
	"strings"
	"testing"
)

// Compute the collision rate of 'Get' function.
func TestCollisionRate(t *testing.T) {
	for length := 1; length <= 8; length++ {
		for n := 1000; n <= 10000; n += 1000 {
			fmt.Printf("n=%d length=%d collision rate=%.3v\n", n, length, collisionRate(n, length))
		}
	}
}

func collisionRate(n, length int) float64 {
	var (
		set            = make(map[string]struct{})
		collisionCount int
	)
	for i := 0; i < n; i++ {
		str := Get(length)
		if _, ok := set[str]; !ok {
			set[str] = struct{}{}
		} else {
			collisionCount++
		}
	}

	return float64(collisionCount) / float64(n)
}

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

func TestPerm(t *testing.T) {
	var strs = []string{
		"",
		"a",
		"你",
		"Who Are You?",
		"Am I Alone?",
		"你是谁?",
		"我是一个人吗?",
		"If today is my last day, what should I do? what I'm best to do?",
	}

	for _, str := range strs {
		nstr := Perm(str)
		xassert.Equal(t, len(str), len(nstr))
		fmt.Println(nstr)
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
