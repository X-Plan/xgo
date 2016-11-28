// xkwfilter_test.go
//
//      Copyright (C), blinklv. All rights reserved.
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-11-28
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2016-11-28

// go-xkwfilter的测试文件.
package xkwfilter

import (
	"github.com/X-Plan/xgo/go-xassert"
	"math/rand"
	"testing"
	"time"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6
	letterIdxMask = 1<<letterIdxBits - 1
	letterIdxMax  = 63 / letterIdxBits
)

var src = rand.NewSource(time.Now().UnixNano())

func randString(n int) string {
	b := make([]byte, n)
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

func getAmCount(xkwf *XKeywordFilter) (ac int) {
	for _, b := range xkwf.am {
		if b {
			ac++
		}
	}
	return
}

func TestNew(t *testing.T) {
	var (
		keywords []string
		xkwf     *XKeywordFilter
	)
	for i := 0; i < 100; i++ {
		keywords = append(keywords, randString(i+20))
		xkwf = New("***", keywords...)
		xassert.NotNil(t, xkwf)
		xassert.Equal(t, xkwf.cl <= 52, true)
		xassert.Equal(t, getAmCount(xkwf), i+1)
	}
}
