// xconfig_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2018-01-26
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2018-01-29
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

func TestParseLevel(t *testing.T) {
	elements := []struct {
		str   string
		level int
		ok    bool
	}{
		{"fatal", FATAL, true},
		{"error", ERROR, true},
		{"warn", WARN, true},
		{"info", INFO, true},
		{"debug", DEBUG, true},
		{"Fatal", -1, false},
		{"eRror", -1, false},
		{"hello world", -1, false},
	}

	for _, element := range elements {
		level, err := parseLevel(element.str)
		if element.ok {
			xassert.Equal(t, level, element.level)
		} else {
			xassert.NotNil(t, err)
		}
	}
}

func TestReadableMaxSize(t *testing.T) {
	elements := []struct {
		maxSize int
		str     string
	}{
		{10 * gigaByte, "10 GB"},
		{100*gigaByte + 100*megaByte, "100 GB"},
		{20 * megaByte, "20 MB"},
		{5*gigaByte + 100*megaByte, "5220 MB"},
		{5*megaByte + 1*kiloByte, "5 MB"},
		{5*megaByte + 200*kiloByte, "5320 KB"},
		{100*kiloByte + 100, "100 KB"},
		{9*kiloByte + 100, "9316 B"},
		{956, "956 B"},
	}

	for _, element := range elements {
		xassert.Equal(t, readableMaxSize(element.maxSize), element.str)
	}
}

func TestReadableMaxAge(t *testing.T) {
	elements := []struct {
		maxAge int
		str    string
	}{
		{2 * year2min, "2 year"},
		{2*year2min + 3*day2min, "2 year"},
		{100*year2min + 1*month2min, "100 year"},
		{5*year2min + 1*month2min, "60 month"},
		{1*year2min + 1*month2min, "397 day"},
		{2*month2min + 1*day2min, "9 week"},
		{2*month2min + 2*day2min, "64 day"},
		{10*day2min + 1*hour2min, "10 day"},
		{1*day2min + 5*hour2min, "29 hour"},
		{3*hour2min + 1, "3 hour"},
		{1*hour2min + 1, "61 min"},
	}

	for _, element := range elements {
		xassert.Equal(t, readableMaxAge(element.maxAge), element.str)
	}
}

func TestImportAndExport(t *testing.T) {
	elements := []struct {
		data map[string]interface{}
		ok   bool
	}{
		{map[string]interface{}{
			"dir":         "/tmp/log",
			"max_size":    "2 GB",
			"max_backups": 50,
			"max_age":     "6 month",
			"tag":         "test 1",
			"level":       "info",
		}, true},
		{map[string]interface{}{
			"dir":         "/tmp/log",
			"max_size":    "512 mb",
			"max_backups": float64(32),
			"max_age":     "2 week",
			"tag":         "test 2",
			"level":       "debug",
		}, true},
		{map[string]interface{}{"dir": "/tmp/log"}, true},
		{map[string]interface{}{"max_size": 10 * kiloByte}, true},
		{map[string]interface{}{"max_size": "10 kb"}, true},
		{map[string]interface{}{"max_backups": 50.1}, true},
		{map[string]interface{}{"max_age": "5 month"}, true},
		{map[string]interface{}{"tag": "hello"}, true},
		{map[string]interface{}{"level": "fatal"}, true},

		{map[string]interface{}{
			"dir":         "/tmp/log",
			"max_size":    "512 mb",
			"max_backups": uint8(32),
			"max_age":     "6 day",
			"tag":         "test 3",
			"level":       "hello",
		}, false},
		{map[string]interface{}{"max_size": "a MB"}, false},
		{map[string]interface{}{"max_age": "10 days"}, false},
		{map[string]interface{}{"level": "nothing"}, false},
	}

	for _, element := range elements {
		xcfg := &XConfig{}
		err := xcfg.Import(element.data)
		if element.ok {
			xassert.IsNil(t, err)
			data := make(map[string]interface{})
			xassert.IsNil(t, xcfg.Export(data))

			tmp := &XConfig{}
			xassert.IsNil(t, tmp.Import(data))
			xassert.Equal(t, xcfg, tmp)
		} else {
			xassert.NotNil(t, err)
		}
	}
}
