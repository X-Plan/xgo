// xdebug.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-02-10
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-03-02

// go-xdebug提供了一个用于打印调试信息的接口.
package xdebug

import (
	"fmt"
	"io"
	"log"
	"time"
)

const Version = "1.1.1"

// 用于打印调试信息的接口. 要么不初始化该值.
// 要么使用New和Inherit函数创建该值.
type XDebugger struct {
	d debugger
}

// 打印调试信息.
func (xd *XDebugger) Printf(format string, argv ...interface{}) {
	if xd != nil {
		xd.d.Printf(true, format, argv...)
	}
}

// 创建一个新的调试接口.
func New(prefix string, w io.Writer) *XDebugger {
	// 这里并不使用log.Logger打印附加信息, 只是利用其换行的能力
	// 和对并发访问的支持.
	return &XDebugger{&rootDebugger{prefix: prefix, l: log.New(w, "", 0)}}
}

// 继承一个已有的调试接口.
func Inherit(prefix string, xd *XDebugger) *XDebugger {
	if xd != nil {
		return &XDebugger{&childDebugger{parent: xd.d, prefix: prefix}}
	}
	return nil
}

type debugger interface {
	Printf(begin bool, format string, argv ...interface{})
}

type rootDebugger struct {
	l      *log.Logger
	prefix string
}

func (rd *rootDebugger) Printf(begin bool, format string, argv ...interface{}) {
	// *log.Logger已经是并发安全的.
	if begin {
		rd.l.Printf(timeTag()+"["+rd.prefix+"]: "+format, argv...)
	} else {
		rd.l.Printf(timeTag()+"["+rd.prefix+"]"+format, argv...)
	}
}

type childDebugger struct {
	parent debugger
	prefix string
}

func (cd *childDebugger) Printf(begin bool, format string, argv ...interface{}) {
	if cd.prefix != "" {
		if begin {
			cd.parent.Printf(false, "["+cd.prefix+"]: "+format, argv...)
		} else {
			cd.parent.Printf(false, "["+cd.prefix+"]"+format, argv...)
		}
	} else {
		cd.parent.Printf(begin, format, argv...)
	}
}

func timeTag() string {
	t := time.Now()
	return fmt.Sprintf("[%04d-%02d-%02d %02d:%02d:%02d]",
		int(t.Year()), int(t.Month()), int(t.Day()), int(t.Hour()), int(t.Minute()), int(t.Second()))
}
