// xcache_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2018-02-11
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2018-02-11

package xcache

import (
	"github.com/X-Plan/xgo/go-xassert"
	"hash/fnv"
	"testing"
)

func TestFnv32a(t *testing.T) {
	strs := []string{
		"Somebody tell me.",
		"Why it feels more real when I dream than when I am awake.",
		"How can I know If my senses are lying?",
		"",
		"There is some fiction in your truth,",
		"and some truth in your fiction.",
		"To the truth, you must risk everything.",
		"",
		"Who are you?",
		"Am I alone?",
		"",
		"You are not alone.",
		"",
		"                               --- A Kid's Story",
	}

	for _, str := range strs {
		xassert.Equal(t, fnv32a(str), stdFnv32a(str))
	}
}

func stdFnv32a(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}
