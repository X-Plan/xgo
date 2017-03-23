// xrandstring.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-01-07
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-03-23

// go-xrandstring package contains some random operations about string.
package xrandstring

import (
	"math/rand"
	"time"
)

const Version = "1.1.0"
const LetterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const (
	letterIdxBits = 6
	letterIdxMask = 1<<letterIdxBits - 1
	letterIdxMax  = 63 / letterIdxBits
)

var src = rand.NewSource(time.Now().UnixNano())

// Generate a random string of length n, its character set is 'LetterBytes'.
func Get(n int) string {
	b := make([]byte, n)
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
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
// is less than the 'str', directly return old string. You can assume that this
// function will randomly remove a substring when the 'str' is empty.
func Replace(old string, str string) string {
	if len(old) > len(str) {
		i := int(src.Int63()) % (len(old) - len(str) + 1)
		return old[:i] + str + old[i+len(str):]
	} else if len(old) == len(str) {
		return str
	} else {
		return old
	}
}
