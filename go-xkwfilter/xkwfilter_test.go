// xkwfilter_test.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-11-28
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2016-12-31

// go-xkwfilter的测试文件.
package xkwfilter

import (
	"bytes"
	"github.com/X-Plan/xgo/go-xassert"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
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

// 验证Filter函数的正确性.
func TestFilter(t *testing.T) {
	var (
		xkwf = New(
			"***",
			"", // 空关键字.
			"..",
			"...",
			"!!!",
			"...!!",
			"^^^...",
			"你好",
			"世界",
			"你是谁",
			"你叫什么名字",
		)

		article = strings.Join([]string{
			"...",
			"^^^...",
			"...",
			randString(20),
			"...",
			randString(20),
			"...!!",
			"!!!",
			"...",
			"^^^...",
			"...",
			"!!",
			"!!!",
			"..",
			randString(20),
			"^^^",
			"...!!",
			"!",
			randString(20),
			"你是谁",
			randString(20),
			"世界",
			"你叫什么名字",
			"你你好",
			randString(20),
		}, "")
	)

	buf := bytes.NewBuffer([]byte(article))
	n, err := xkwf.Filter(os.Stdout, buf)
	xassert.Equal(t, n, 144)
	xassert.IsNil(t, err)
}

func BenchmarkFilter(b *testing.B) {
	var (
		keywords = []string{
			".....",
			"^^^^^",
			"#####",
			"...##",
			"^^..##",
			"..^^#",
			"#^^^^.",
			"^^^..",
		}
		xkwf    = New("***", keywords...)
		article string
	)

	for i := 0; i < 1000; i++ {
		article += randString(100) + keywords[i%len(keywords)]
	}
	buf := bytes.NewBuffer([]byte(article))

	b.StartTimer()
	for i := 0; i < 100000; i++ {
		xkwf.Filter(ioutil.Discard, buf)
	}
	b.StopTimer()
}
