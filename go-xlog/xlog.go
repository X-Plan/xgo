// xlog.go
//
//		Copyright (C), blinklv. All rights reserved.
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-10-26
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2016-11-02

// xlog实现了一个单进程下并发安全的滚动日志.
package xlog

import (
	"bufio"
	"fmt"
	"os"
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

// 如果日志已经关闭还对其进行操作会抛出
// 该错误.
var ErrClosed = errors.New("XLogger has been closed")

// 用于创建XLogger的配置.
type XConfig struct {
	// 日志文件的存储目录, 默认为当前目录.
	Dir string `json:"dir"`

	// 每个日志文件的最大值, 单位是字节.
	// 当日志文件的大小超过该值的时候会引发
	// 日志文件的切换. 该值的设定必须大于
	// 或等于0, 等于0的时候代表没有限制.
	MaxSize int64 `json:"max_size"`

	// 日志文件数量的最大值. 当文件数量超过
	// 改值的时候, 最旧的日志文件会被删除.
	// 该值的设定必须大于或等于0, 等于0的时候
	// 代表没有限制.
	MaxBackups int64 `json:"max_backups"`

	// 日志文件存储的最长时间. 当一个日志文件
	// 的存储时间超过该值也会被删除. 该值的的
	// 设定必须大于或等于0, 等于0的时候代表
	// 没有限制.
	MaxAge string `json:"max_age"`

	// 日志标签.
	Tag string `json:"tag"`

	// 日志级别 (调用Write函数时不考虑该值).
	// 只有操作优先级高于或等于日志级别的
	// 情况下日志才会被写入到文件.
	Level int `json:"level"`
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
// 直接调用Write函数时只会记录message, 没有前面的附加记录.
type XLogger struct {
	dir   string
	ms    int64
	mb    int64
	ma    time.Duration
	tag   string
	level int

	// 为了在并发环境下使用XLogger, 涉及到
	// 写文件的操作时, 需要将并行化的消息
	// 串行化写入到文件中. 因此这里选用一个
	// 通道来完成这一串行化的过程.
	bc chan []byte

	// 用来终止flush协程.
	exitChan chan int
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
		xl.dir = xcfg.Dir
	} else {
		xl.dir = "."
	}

	if xcfg.MaxSize < 0 {
		return nil, fmt.Errorf("MaxSize is invalid")
	}
	xl.ms = xcfg.MaxSize

	if xcfg.MaxBackups < 0 {
		return nil, fmt.Errorf("MaxBackups is invalid")
	}
	xl.mb = xcfg.MaxBackups

	xl.ma, err = time.ParseDuration(xcfg.MaxAge)
	if err != nil {
		return nil, err
	}

	xl.tag = xcfg.Tag

	if xcfg.Level >= 0 && xcfg.Level <= DEBUG {
		xl.level = xcfg.Level
	} else {
		return nil, fmt.Errorf("Level is invalid")
	}

	// 缓存的大小为60条消息.
	xl.bc = make(chan []byte, 60)
	xl.exitChan = make(chan int)

	// 异步执行数据同步到文件中的任务.
	go xl.flush()

	return xl, nil
}

// 将数据写入到XLogger. 通常情况下你不需要直接
// 调用该接口. 除非你已经拥有良好的日志格式.
func (xl *XLogger) Write(b []byte) (int, error) {
	// 捕获写已经关闭的channel所产生的panic.
	defer func() {
		if x := recover(); x != nil {
			return 0, ErrClosed
		}
	}()

	xl.bc <- b
	// 这样的返回是为了满足io.Writer接口.
	return len(b), nil
}

func (xl *XLogger) Close() error {
	// 关闭一个已经关闭的channel会产生
	// panic, 这里也需要对其进行捕获.
	defer func() {
		if x := recover(); x != nil {
			return ErrClosed
		}
	}()

	// 先关闭bc, 首先阻止Write函数的使用.
	// 但是其中的残留数据依然会被读取.
	close(xl.bc)

	// 发送退出信号, 终止flush协程.
	close(xl.exitChan)

	return nil
}

func (xl *XLogger) flush() {
	// flush函数是以bc为优先的,
	// 这样即便收到了退出信号也会
	// 将残留的日志写入到磁盘.
	for {
		select {
		case b := <-xl.bc:
		case <-xl.exitChan:
			// 收到退出信号, 执行退出.
			return
		}
	}
}

// 判断文件名是否符合g-xlog的规范.
func isValidName(name string) bool {
	matched, _ := regexp.MatchString(`^\d{4}(_\d{2}){5}_\d{9}$`, name)
	return matched
}

// 将规范的文件名转换为时间结构.
func name2time(name string) time.Time {
	var year, month, day, hour, min, sec, nsec int
	name = strings.Replace(name, "_", " ", -1)
	fmt.Sscanf(name, "%d%d%d%d%d%d%d", &year, &month, &day, &hour, &min, &sec, &nsec)
	t := time.Date(year, time.Month(month), day, hour, min, sec, nsec, time.Local)
	return t
}

// 将时间是结构转换为规范的文件名.
// 格式为:year_month_day_hour_min_sec_nsec
func time2name(t time.Time) string {
	return fmt.Sprintf("%04d_%02d_%02d_%02d_%02d_%02d_%09d", int(t.Year()), int(t.Month()),
		int(t.Day()), int(t.Hour()), int(t.Minute()), int(t.Second()), int(t.Nanosecond()))
}
