// xconfig_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2018-01-26
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2018-01-26
package xlog

import (
	"github.com/X-Plan/xgo/go-xassert"
	"testing"
)

func TestParseMaxSize(t *testing.T) {
	elements := []struct {
		str     string
		maxSize int64
		ok      bool
	}{
		{" 978   b", 978, true},
		{" 1024 B", 1024, true},
		{"10 kb", 10 * kiloByte, true},
		{" 100KB", 100 * kiloByte, true},
		{" 210 mb", 210 * megaByte, true},
		{"220MB", 220 * megaByte, true},
		{"128 gb", 128 * gigaByte, true},
		{"200    GB", 200 * gigaByte, true},
		{"128 MB  ", 0, false},
		{"abc GB", 0, false},
		{"137 TB", 0, false},
	}

	for _, element := range elements {
		maxSize, err := parseMaxSize(element.str)
		if element.ok {
			xassert.Equal(t, maxSize, element.maxSize)
		} else {
			xassert.NotNil(t, err)
		}
	}
}

func TestParseMaxAge(t *testing.T) {
	elements := []struct {
		str    string
		maxAge string
		ok     bool
	}{
		{"  20   min", "20m", true},
		{"1 hour", "60m", true},
		{"7day", "10080m", true},
		{"2 week", "20160m", true},
		{"3 month", "133920m", true},
		{"2 year", "1054080m", true},
		{" 32 year ", "", false},
		{" asd hour", "", false},
	}

	for _, element := range elements {
		maxAge, err := parseMaxAge(element.str)
		if element.ok {
			xassert.Equal(t, maxAge, element.maxAge)
		} else {
			xassert.NotNil(t, err)
		}
	}
}
