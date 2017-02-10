// xdebug.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-02-10
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-02-10

// go-xdebug提供了一个用于打印调试信息的接口.
package xdebug

import (
	"io"
	"log"
)

const Version = "1.0.0"

// 用于打印调试信息的接口.
type XDebugger struct {
	d debugger
}

func (xd *XDebugger) Printf(format string, argv ...interface{}) {
	if xd != nil {
		xd.d.Printf(format, argv...)
	}
}

// 创建一个新的调试接口.
func New(prefix string, w io.Writer) *XDebugger {
	return &XDebugger{&rootDebugger{l: log.New(w, "["+prefix+"]", 0)}}
}

// 继承一个已有的调试接口.
func Inherit(prefix string, xd XDebugger) *XDebugger {
	return &XDebugger{&childDebugger{parent: xd.d, prefix: prefix}}
}

type debugger interface {
	Printf(format string, argv ...interface{})
}

type rootDebugger struct {
	l *log.Logger
}

func (rd *rootDebugger) Printf(format string, argv ...interface{}) {
	rd.l.Printf(format, argv...)
}

type childDebugger struct {
	parent debugger
	prefix string
}

func (cd *childDebugger) Printf(format string, argv ...interface{}) {
	cd.parent.Printf("["+cd.prefix+"]"+format, argv...)
}
