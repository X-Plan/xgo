// xrandstring.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-01-07
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-07-24

// go-xrandstring package contains some random operations about string.
package xrandstring

import (
	"math/rand"
	"time"
)

const LetterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const (
	letterIdxBits = 6
	letterIdxMask = 1<<letterIdxBits - 1
	letterIdxMax  = 63 / letterIdxBits
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Generate a random string of length n, its character set is 'LetterBytes'.
func Get(n int) string {
	b := make([]byte, n)
	for i, cache, remain := n-1, rand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = rand.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(LetterBytes) {
			b[i] = LetterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

// Replace a random substring of the 'old' with the 'str', return a new string,
// the random substring and 'str' are equal in length. If the length of the 'old'
// is less than the 'str' or str is empty, directly return old string.
func Replace(old string, str string) string {
	if len(str) > 0 && len(str) < len(old) {
		i := int(rand.Int63()) % (len(old) - len(str) + 1)
		return old[:i] + str + old[i+len(str):]
	} else if len(str) == len(old) {
		// If both 'old' and 'str' are empty, return 'str' is
		// equal to return 'old' effectively. But I don't know
		// who will do that, it's too strange.
		return str
	} else {
		return old
	}
}

// Generate a random permutation from the original string. The basic
// element of permutation is 'rune' type, not 'byte' type.
func Perm(str string) string {
	// The implemention is based on Fisher-Yates shuffle algorithm.
	var (
		rs = []rune(str)
		n  = len(rs)
	)

	for i := n - 1; i >= 0; i-- {
		j := int(rand.Int63()) % (i + 1)
		rs[i], rs[j] = rs[j], rs[i]
	}

	return string(rs)
}
