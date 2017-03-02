// xdebug_test.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-02-10
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-03-02

package xdebug

import (
	"os"
	"strconv"
	"testing"
)

func TestNilXDebugger(t *testing.T) {
	var xd *XDebugger
	xd.Printf("I'm %s", "nil")
	xd = Inherit("nil", xd)
	xd.Printf("I'm %s too", "nil")
}

func TestXDebugger(t *testing.T) {
	var (
		xds []*XDebugger = make([]*XDebugger, 10)
	)

	xds[0] = New("root", os.Stderr)

	for i := 1; i < 10; i++ {
		xds[i] = Inherit("child"+strconv.Itoa(i), xds[i-1])
	}

	for i := 0; i < 10; i++ {
		xds[i].Printf("Hello, Number %d", i)
	}
}
