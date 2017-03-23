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

const Version = "1.0.0"
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
