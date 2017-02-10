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

// 用于打印调试信息的接口. 要么不初始化该值.
// 要么使用New和Inherit函数创建该值.
type XDebugger struct {
	d debugger
}

// 打印调试信息.
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
func Inherit(prefix string, xd *XDebugger) *XDebugger {
	return &XDebugger{&childDebugger{parent: xd.d, prefix: prefix}}
}

type debugger interface {
	Printf(format string, argv ...interface{})
}

type rootDebugger struct {
	l *log.Logger
}

func (rd *rootDebugger) Printf(format string, argv ...interface{}) {
	// *log.Logger已经是并发安全的.
	rd.l.Printf(format, argv...)
}

type childDebugger struct {
	parent debugger
	prefix string
}

func (cd *childDebugger) Printf(format string, argv ...interface{}) {
	cd.parent.Printf("["+cd.prefix+"]"+format, argv...)
}
