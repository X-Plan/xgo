// xlog.go
//
//		Copyright (C), blinklv. All rights reserved.
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-10-26
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2016-12-19

// xlog实现了一个单进程下并发安全的滚动日志.
package xlog

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	Version = "1.2.0"
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

var levelTags = [...]string{
	"",
	"[FATAL]",
	"[ERROR]",
	"[WARN]",
	"[INFO]",
	"[DEBUG]",
}

// 一个应用程序不应该将两个XLogger实体
// 对应到同一个目录下, 这样会异常结果.
// dirMap目的是为了防止这种现象发生.
var dirMap = make(map[string]bool)
var dirMtx = &sync.Mutex{}

// 如果日志已经关闭还对其进行操作会抛出
// 该错误.
var ErrClosed = errors.New("XLogger has been closed")

// 用于创建XLogger的配置.
type XConfig struct {
	// 日志文件的存储目录. 如果为空则替换为
	// 当前执行目录下的log目录, 如果目录不存
	// 在则主动创建(可能会因为权限原因创建
	// 失败), 如果目录存在则该用具必须具备
	// 写权限.
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

	// 日志标签. 如果没有设置则默认为
	// 进程名称.
	Tag string `json:"tag"`

	// 日志级别 (调用Write函数时不考虑该值).
	// 只有操作优先级高于或等于日志级别的
	// 情况下日志才会被写入到文件. 否则会被
	// 打印到标准错误输出.
	Level int `json:"level"`
}

// XLogger的日志纪录格式为:
// [yyyy-mm-dd hh:mm:ss][tag][level][location]: message
// '[yyyy-mm-dd hh:mm:ss]' 为写记录的时间
// '[tag]' 为用户自定义标签
// '[level]' 为记录的优先级别
// '[location]' 事件发生的位置, 包括文件名, 行号, 相关的函数.
// Info()函数不会打印location信息.
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
	// 通道来完成这一串行化的过程. 对该管道
	// 写操作是阻塞的.
	bc chan []byte

	// 用于flush协程通知主调协程flush关闭完成.
	exitChan chan int

	// 用来从flush协程中获取错误
	// 信息.
	errorChan chan error

	// 文件指针.
	f *file

	// 标示是否为半关闭状态.
	half int32
}

func New(xcfg *XConfig) (xl *XLogger, err error) {

	// 收尾工作, 比如解绑目录和xl
	// 置为空.
	defer func() {
		if err != nil {
			unbindDir(xl.dir)
			xl = nil
		}
	}()

	xl = &XLogger{}

	// 参数校验
	if xcfg == nil {
		err = fmt.Errorf("XConfig is nil")
		return
	}

	// 如果没有设置目录则默认使用当前目录.
	if xcfg.Dir != "" {
		xl.dir = xcfg.Dir
	} else {
		xl.dir = "./log"
	}

	if err = bindDir(xl.dir); err != nil {
		return
	}

	// 创建目录, 如果目录已经存在则该操作
	// 会被忽略. 可以参考GoDoc中对MkdirAll
	// 的描述.
	if err = os.MkdirAll(xl.dir, 0777); err != nil {
		return
	}
	// 检测用户是否对该目录具备写权限.
	if err = isWritable(xl.dir); err != nil {
		return
	}

	if xcfg.MaxSize < 0 {
		err = fmt.Errorf("MaxSize is invalid")
		return
	}
	xl.ms = xcfg.MaxSize

	if xcfg.MaxBackups < 0 {
		err = fmt.Errorf("MaxBackups is invalid")
		return
	}
	xl.mb = xcfg.MaxBackups

	if xcfg.MaxAge != "" {
		xl.ma, err = time.ParseDuration(xcfg.MaxAge)
		if err != nil {
			return
		}
	} else {
		xl.ma = time.Duration(0)
	}

	if xcfg.Tag != "" {
		xl.tag = xcfg.Tag
	} else {
		xl.tag = filepath.Base(os.Args[0])
	}

	if xcfg.Level >= FATAL && xcfg.Level <= DEBUG {
		xl.level = xcfg.Level
	} else {
		err = fmt.Errorf("Level is invalid")
		return
	}

	// 缓存的大小为128条消息.
	xl.bc = make(chan []byte, 128)
	xl.exitChan = make(chan int)
	xl.errorChan = make(chan error, 8)

	// 异步执行数据同步到文件中的任务.
	go xl.flush()

	return xl, nil
}

// 将数据写入到XLogger. 通常情况下你不需要直接
// 调用该接口. 除非你已经拥有良好的日志格式.
func (xl *XLogger) Write(b []byte) (n int, err error) {
	// 捕获写已经关闭的channel所产生的panic.
	defer func() {
		if x := recover(); x != nil {
			n, err = 0, ErrClosed
		}
	}()

	select {
	case xl.bc <- b:
		// 这样的返回是为了满足io.Writer接口.
		n, err = len(b), nil
	case err = <-xl.errorChan:
		// 这里需要处理errorChan已经关闭
		// 情形.
		if err == nil {
			n, err = 0, ErrClosed
		} else {
			n, err = 0, err
		}
	}

	return
}

func (xl *XLogger) Fatal(format string, args ...interface{}) error {
	return xl.output(FATAL, fmt.Sprintf(format, args...))
}

func (xl *XLogger) Error(format string, args ...interface{}) error {
	return xl.output(ERROR, fmt.Sprintf(format, args...))
}

func (xl *XLogger) Warn(format string, args ...interface{}) error {
	return xl.output(WARN, fmt.Sprintf(format, args...))
}

func (xl *XLogger) Info(format string, args ...interface{}) error {
	return xl.output(INFO, fmt.Sprintf(format, args...))
}

func (xl *XLogger) Debug(format string, args ...interface{}) error {
	return xl.output(DEBUG, fmt.Sprintf(format, args...))
}

// 关闭XLogger. 如果一个日志已经关闭, 则对日志的
// 任何操作都会返回ErrClosed错误. 如果日志在写文
// 件的时候出现问题, 日志也会被异常关闭.
func (xl *XLogger) Close() (err error) {
	// 关闭一个已经关闭的channel会产生
	// panic, 这里也需要对其进行捕获.
	defer func() {
		if x := recover(); x != nil {
			err = ErrClosed
		}
	}()

	// 关闭资源的策略是资源被哪个协程写
	// 就在对应的协程被关闭. 这里的资源
	// 主要是Channel和文件. 尤其是Channel,
	// 当多协程的时候出现读/写一个关闭的
	// Channel是很常见的情况. 读一个关闭的
	// Channel并没有有什么影响, 但是写一个
	// Channel则会抛出panic. 所以将对管道的
	// 写/关闭操作放在同一个协程可以保证
	// 操作的正确性.
	close(xl.bc)

	// 等待flush协程的关闭, 这里进行等待
	// 的主要目的是防止flush未执行关闭操作
	// 前进程已经退出, 这种情况下部分数据
	// 未能写到持久化设备上, 但是这个步骤
	// 只能保证flush执行完毕, 并不能保证
	// 最后的数据一定能落地.
	<-xl.exitChan

	// 解除对目录的绑定.
	unbindDir(xl.dir)

	return
}

func (xl *XLogger) flush() {
	var (
		err error
		b   []byte
	)

	// 当主协程关闭bc的时候该循环也会终止.
	for b = range xl.bc {
		if err = xl.write(b); err != nil {

			// 传递错误给主调协程.
			// errorChan的缓冲区有限, 所以
			// errorChan中的错误还没有被主协
			// 程捕获之前又生成了新的错误,
			// 这里的需要将老的错误排出, 然后
			// 写入新的错误. 这里的写法并不多
			// 余. 必须保证两个操作都是非阻塞的.
			select {
			case xl.errorChan <- err:
			default:
				select {
				// 因为操作不是原子的, 所以在
				// 判断没有空间和读取老数据之间
				// 父协程可能将数据已经读取了.
				case <-xl.errorChan:
					xl.errorChan <- err
				default:
					// 无操作.
				}
			}
		}
	}

	// 将文件落地到持久化设备上.
	// 这里不进行错误的捕获.
	if xl.f != nil {
		xl.f.Close()

	}
	close(xl.errorChan)
	close(xl.exitChan)

	return

}

// 这个write函数不同于Write函数, 它是将
// 用于将数据写向文件的. XLogger会在打
// 开就文件和创建新文件这两个操作之前进
// 行清理操作.
func (xl *XLogger) write(b []byte) error {
	var err error
	// 初始化XLogger.
	if xl.f == nil {
		if err = cleanup(xl.dir, xl.ma, xl.mb, xl.mb); err != nil {
			return err
		}

		if xl.f, err = openFile(xl.dir); err != nil {
			return err
		}
	}

	// 目录下的日志文件都不符合要求或
	// 当前日志文件的大小超标.
	if xl.f == nil || (xl.ms > 0 && xl.f.Size >= xl.ms) {
		// 在使用新的文件前必须要确保
		// 老文件的数据已经落地. 这步
		// 操作应该放置在cleanup操作
		// 之前.
		if xl.f != nil {
			if err = xl.f.Close(); err != nil {
				return err
			}
		}

		// 因为之后会创建新的文件, 因此需要在这里
		// 将文件数量的上限减少一个单位.
		if err = cleanup(xl.dir, xl.ma, xl.mb, xl.mb-1); err != nil {
			return err
		}

		if xl.f, err = createFile(xl.dir); err != nil {
			return err
		}

	}

	_, err = xl.f.Write(b)
	return err
}

// 对输出进行包装. 附带一些标签信息.
func (xl *XLogger) output(level int, m string) (err error) {
	s := strings.Join([]string{
		timeTag(),
		"[", xl.tag, "]",
		levelTags[level],
		locationTag(level, 2),
		":", m, "\n",
	}, "")

	if level <= xl.level {
		_, err = xl.Write([]byte(s))
	} else {
		_, err = fmt.Fprintf(os.Stdout, s)
	}
	return
}

// 对文件指针进行一层封装, 提供一个写
// 缓存和大小的记录.
type file struct {
	Fp   *os.File
	Size int64
}

// 创建新文件.
func createFile(dir string) (*file, error) {
	var (
		err error
		f   = &file{}
	)

	f.Fp, err = os.OpenFile(getName(dir, time.Now()), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	f.Size = int64(0)

	return f, nil
}

// 打开指定目录下的最新文件.
func openFile(dir string) (*file, error) {
	var (
		fiq *fileInfoQueue
		err error
		f   = &file{}
	)

	if fiq, err = createFileInfoQueue(dir); err != nil {
		return nil, err
	}
	fiq.Sort()

	if !fiq.IsEmpty() {
		fi := fiq.Last()
		// 这里默认不进行文件的创建.
		f.Fp, err = os.OpenFile(getName(dir, fi.CreateTime), os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}

		f.Size = fi.Size

		return f, nil
	}

	// 没有文件不代表存在错误.
	return nil, nil
}

// 清除指定目录下的文件.
func cleanup(dir string, ma time.Duration, mb, amb int64) error {
	var (
		fiq *fileInfoQueue
		err error
	)

	if fiq, err = createFileInfoQueue(dir); err != nil {
		return err
	}

	fiq.Sort()
	fiq.CleanUp(dir, ma, mb, amb)

	return nil
}

func (f *file) Write(b []byte) (int, error) {
	n, err := f.Fp.Write(b)
	f.Size += int64(n)
	return n, err
}

func (f *file) Close() error {
	return f.Fp.Close()
}

// 文件信息的简易队列, 这是一个辅助结构.
// 用于过期文件的清除.
type fileInfo struct {
	CreateTime time.Time
	Size       int64
}

type fileInfoQueue []fileInfo

func createFileInfoQueue(dir string) (*fileInfoQueue, error) {
	var (
		err error
		sts []os.FileInfo
		fiq fileInfoQueue
	)

	err = os.MkdirAll(dir, 0744)
	if err != nil {
		return nil, err
	}

	// 获取文件状态信息.
	sts, err = ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	fiq = make([]fileInfo, 0, len(sts))
	for _, st := range sts {
		if !st.IsDir() && isValidName(st.Name()) {
			fiq = append(fiq, fileInfo{name2time(st.Name()), st.Size()})
		}
	}

	return &fiq, nil
}

// 这几个函数的定义是为了使用sort.Sort函数.
func (fiq fileInfoQueue) Len() int {
	return len(fiq)
}

func (fiq fileInfoQueue) Less(i, j int) bool {
	return fiq[i].CreateTime.Before(fiq[j].CreateTime)
}

func (fiq fileInfoQueue) Swap(i, j int) {
	fiq[i], fiq[j] = fiq[j], fiq[i]
}

func (fiq fileInfoQueue) Sort() {
	sort.Sort(fiq)
}

func (fiq fileInfoQueue) IsEmpty() bool {
	return len(fiq) == 0
}

func (fiq fileInfoQueue) Last() fileInfo {
	return fiq[len(fiq)-1]
}

func (fiq *fileInfoQueue) Append(fi fileInfo) {
	(*fiq) = append(*fiq, fi)
}

// 只有在队列处于排序状态下才可以使用该函数.
// 该函数从保留个数和保留时间两个纬度来删除
// 老文件.
func (fiq *fileInfoQueue) CleanUp(dir string, ma time.Duration, mb, amb int64) {
	var (
		removes   []fileInfo
		i         int64
		n         int64
		threshold time.Time
	)

	if mb > 0 && int64(fiq.Len()) > amb {
		i += int64(fiq.Len()) - amb
	}

	if ma > time.Duration(0) {
		threshold = time.Now().Add((-1) * ma)
		n = int64(fiq.Len())

		for ; i < n; i++ {
			if (*fiq)[i].CreateTime.After(threshold) {
				break
			}
		}
	}

	removes = (*fiq)[:i]
	*fiq = (*fiq)[i:]

	for _, info := range removes {
		os.Remove(filepath.Join(dir, time2name(info.CreateTime)))
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

// 获取完整的文件名称.
func getName(dir string, t time.Time) string {
	return filepath.Join(dir, time2name(t))
}

// 生成一个时间字段用于日志的
// 格式化输出.
func timeTag() string {
	t := time.Now()
	return fmt.Sprintf("[%04d-%02d-%02d %02d:%02d:%02d]",
		int(t.Year()), int(t.Month()), int(t.Day()), int(t.Hour()), int(t.Minute()), int(t.Second()))
}

// 定位标签, 在Info函数中不会使用.
func locationTag(level int, skip int) string {
	if level == INFO {
		return ""
	}

	pc, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return "[]" // Location failure.
	}
	f := runtime.FuncForPC(pc)
	return fmt.Sprintf("[%s(%d) - %s]", filepath.Base(file), line, f.Name())
}

// 检测用户是否对特定的目录具备写权限.
// 判断的依据就是查看用户是否可以在该
// 目录下创建文件. 这是最可靠的办法,
// 如果依靠file bit, 在文件系统是只读
// 的方式进行挂载, 以及一些文件系统进行
// 了ACL扩展的情况下会失败.
func isWritable(dir string) error {
	tmp, err := ioutil.TempFile(dir, "xlog_test")
	if err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("Write %s directory permission denied", dir)
		} else {
			return err
		}
	}
	os.Remove(tmp.Name())
	return nil
}

// 绑定目录.
func bindDir(dir string) error {
	dirMtx.Lock()
	defer dirMtx.Unlock()

	if _, ok := dirMap[dir]; !ok {
		dirMap[dir] = true
		return nil
	} else {
		return fmt.Errorf("%s directory has been occupied")
	}
}

// 目录解绑.
func unbindDir(dir string) {
	dirMtx.Lock()
	defer dirMtx.Unlock()
	delete(dirMap, dir)
}
