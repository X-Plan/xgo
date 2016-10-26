// xlog.go
//
//		Copyright (C), blinklv. All rights reserved.
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-10-26
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2016-10-26

// xlog实现了一个单进程下并发安全的滚动日志.
package xlog

import (
	"fmt"
	"time"
)

const (
	Version = "1.0.0"
)

// 日志优先级, 数值越小, 优先级越高.
const (
	_ = iota
	FATAL
	ERROR
	WARN
	INFO
	DEBUG
)

// 用于创建XLogger的配置.
type XConfig struct {
	Dir        string `json:"dir"`
	MaxSize    int64  `json:"max_size"`
	MaxBackups int64  `json:"max_backups"`
	MaxAge     string `json:"max_age"`
	Tag        string `json:"tag"`
	Level      int    `json:"level"`
}

// XLogger的日志纪录格式为:
// [yyyy-mm-dd hh:mm:ss][tag][level][location]: message
// '[yyyy-mm-dd hh:mm:ss]' 为写记录的时间
// '[tag]' 为用户自定义标签
// '[level]' 为记录的优先级别
// '[location]' 事件发生的位置, 包括文件名, 行号, 相关的函数.
//  这个标签只有在使用Debug()函数时才会打印.
// 'message' 为用户自定义数据.
//
// 直接调用Write函数时只存在message, 没有前面的附加记录.
type XLogger struct {
	// 日志文件的存储目录, 默认为当前目录.
	Dir string

	// 每个日志文件的最大值, 单位是字节.
	// 当日志文件的大小超过该值的时候会引发
	// 日志文件的切换. 该值的设定必须大于
	// 或等于0, 等于0的时候代表没有限制.
	MaxSize int64

	// 日志文件数量的最大值. 当文件数量超过
	// 改值的时候, 最旧的日志文件会被删除.
	// 该值的设定必须大于或等于0, 等于0的时候
	// 代表没有限制.
	MaxBackups int64

	// 日志文件存储的最长时间. 当一个日志文件
	// 的存储时间超过该值也会被删除. 该值的的
	// 设定必须大于或等于0, 等于0的时候代表
	// 没有限制.
	MaxAge time.Duration

	// 日志标签.
	Tag string

	// 日志级别 (调用Write函数时不考虑该值).
	// 只有操作优先级高于或等于日志级别的
	// 情况下日志才会被写入到文件.
	Level int
}

func New(xcfg *XConfig) (*XLogger, error) {
	// 参数校验
	if xcfg == nil {
		return nil, fmt.Errorf("XConfig is nil")
	}

	var (
		xl  = &XLogger{}
		err error
	)

	// 如果没有设置目录则默认使用当前目录.
	if xcfg.Dir != "" {
		xl.Dir = xcfg.Dir
	} else {
		xl.Dir = "."
	}

	if xcfg.MaxSize < 0 {
		return nil, fmt.Errorf("MaxSize is invalid")
	}
	xl.MaxSize = xcfg.MaxSize

	if xcfg.MaxBackups < 0 {
		return nil, fmt.Errorf("MaxBackups is invalid")
	}
	xl.MaxBackups = xcfg.MaxBackups

	xl.MaxAge, err = time.ParseDuration(xcfg.MaxAge)
	if err != nil {
		return nil, err
	}

	xl.Tag = xcfg.Tag

	if xcfg.Level >= 0 && xcfg.Level <= DEBUG {
		xl.Level = xcfg.Level
	} else {
		return nil, fmt.Errorf("Level is invalid")
	}

	return xl, nil
}

func (xl *XLogger) Write(b []byte) (int, error) {
}

func (xl *XLogger) Close() error {
}
