// xlog.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2016-10-26
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2018-01-24

// go-xlog implement a concurrently safe rotate-log.
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

// The priority of logger. The smaller value is, the higher priority is.
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

// One application shouldn't create two 'XLogger' on the same directory,
// 'dirMap' is used to prevent this case from happening.
var dirMap = make(map[string]bool)
var dirMtx = &sync.Mutex{}

// Throw this error when operating on a closed log.
var ErrClosed = errors.New("XLogger has been closed")

// The format of log:
// [yyyy-mm-dd hh:mm:ss][tag][level][location]: message
// '[yyyy-mm-dd hh:mm:ss]' - The timestamp writing record
// '[tag]' - User defined tag
// '[level]' - The priority of record
// '[location]' - Location of an event happening, Info() function doesn't print the location information.
// 'message' - User defined data
type XLogger struct {
	dir   string
	ms    int64
	mb    int64
	ma    time.Duration
	tag   string
	level int

	// Because the XLogger instance should be used safely in concurrency environment,
	// I use a channel to impelment this feature, sequence the data.
	bc chan []byte

	// Notify to the main routine that the flush routine has exited.
	exitChan chan int

	// Get the error information of the flush routine.
	errorChan chan error

	f    *file
	half int32
}

func New(xcfg *XConfig) (xl *XLogger, err error) {

	defer func() {
		if err != nil {
			unbindDir(xl.dir)
			xl = nil
		}
	}()

	xl = &XLogger{}

	if xcfg == nil {
		err = fmt.Errorf("XConfig is nil")
		return
	}

	// If 'Dir' field is empty, using './log' by default.
	if xcfg.Dir != "" {
		xl.dir = xcfg.Dir
	} else {
		xl.dir = "./log"
	}

	if err = bindDir(xl.dir); err != nil {
		return
	}

	if err = os.MkdirAll(xl.dir, 0777); err != nil {
		return
	}
	// Check whether user can write to this directory.
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

	xl.bc = make(chan []byte, 128)
	xl.exitChan = make(chan int)
	xl.errorChan = make(chan error, 8)

	// 'flush' routine is used to flush data to disk.
	go xl.flush()

	return xl, nil
}

// Write data to the XLogger instance. You don't need call
// this method in most cases.
func (xl *XLogger) Write(b []byte) (n int, err error) {
	if xl == nil {
		return 0, nil
	}

	// Write data to closed channel will throw a panic,
	// capture this panic and return a readable error to
	// the invoker.
	defer func() {
		if x := recover(); x != nil {
			n, err = 0, ErrClosed
		}
	}()

	select {
	case xl.bc <- b:
		n, err = len(b), nil
	case err = <-xl.errorChan:
		if err == nil {
			n, err = 0, ErrClosed
		} else {
			n, err = 0, err
		}
	}

	return
}

func (xl *XLogger) Fatal(format string, args ...interface{}) error {
	_, err := xl.output(FATAL, xl.sprintf(format, args...))
	return err
}

func (xl *XLogger) Error(format string, args ...interface{}) error {
	_, err := xl.output(ERROR, xl.sprintf(format, args...))
	return err
}

func (xl *XLogger) Warn(format string, args ...interface{}) error {
	_, err := xl.output(WARN, xl.sprintf(format, args...))
	return err
}

func (xl *XLogger) Info(format string, args ...interface{}) error {
	_, err := xl.output(INFO, xl.sprintf(format, args...))
	return err
}

func (xl *XLogger) Debug(format string, args ...interface{}) error {
	_, err := xl.output(DEBUG, xl.sprintf(format, args...))
	return err
}

// In fact, I should implement two types interface likes Go standard package.
// One interface is used to print directly, the other is used to print format
// data, but it's too late, so I have to combine them.
func (xl *XLogger) sprintf(format string, args ...interface{}) string {
	if len(args) == 0 {
		return fmt.Sprintln(format)
	}
	return fmt.Sprintf(format+"\n", args...)
}

// Close the XLogger instance. Operating on a closed XLogger instance
// will return the 'ErrClosed' error. The XLogger instance will be also
// closed if meeting an exception when flush data to disk.
func (xl *XLogger) Close() (err error) {
	if xl == nil {
		return nil
	}
	// Closing a closed channel will throw a panic,
	// capture this panic and return a readable error to
	// the invoker.
	defer func() {
		if x := recover(); x != nil {
			err = ErrClosed
		}
	}()

	close(xl.bc)

	// Waitting for 'flush' routine exited.  The purpose of
	// this operation is avoiding the main routine exited before
	// 'flush' routine, which may cause partial data lost.
	<-xl.exitChan

	// Unbind the directory.
	unbindDir(xl.dir)

	return
}

func (xl *XLogger) flush() {
	var (
		err error
		b   []byte
	)

	// This loop will be ended when the main routine closes 'bc' channel.
	for b = range xl.bc {
		if err = xl.write(b); err != nil {

			// Transmit the error to the main routine. The old error in 'errorChan'
			// will block a new error entering 'errorChan'  before it's captured by
			// the main routine, because the cache of 'errorChan' is limited. So we
			// need to discard the old error at first, then write the new error to it.
			// The following statements is not redundant, two read operations should
			// be nonblocking.
			select {
			case xl.errorChan <- err:
			default:
				select {
				// Because the two read operations are not atomic, the error in
				// 'errorChan' may have been read by the main routine, so I have
				// to add 'default' statement to make this read operation nonblocking.
				case <-xl.errorChan:
					xl.errorChan <- err
				default:
					// nothing
				}
			}
		}
	}

	if xl.f != nil {
		xl.f.Close()

	}
	close(xl.errorChan)
	close(xl.exitChan)

	return

}

// This function is not smiliar to 'Write' function, it's used to
// write data to the file.
func (xl *XLogger) write(b []byte) error {
	var err error
	// Init
	if xl.f == nil {
		if err = cleanup(xl.dir, xl.ma, xl.mb, xl.mb); err != nil {
			return err
		}

		if xl.f, err = openFile(xl.dir); err != nil {
			return err
		}
	}

	// The size of current log file exceeds the limit.
	if xl.f == nil || (xl.ms > 0 && xl.f.Size >= xl.ms) {
		// We must ensure that the data of old log file has been persisted
		// before using the new log file.
		if xl.f != nil {
			if err = xl.f.Close(); err != nil {
				return err
			}
		}

		// Because we will create a new file, so the number of files decreases one.
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

// Decorate the output information, equipped with some tags.
func (xl *XLogger) output(level int, m string) (n int, err error) {
	if xl == nil {
		return 0, nil
	}

	s := strings.Join([]string{
		timeTag(),
		"[", xl.tag, "]",
		levelTags[level],
		locationTag(level, 2),
		":", m,
	}, "")

	if level <= xl.level {
		n, err = xl.Write([]byte(s))
	} else {
		n, err = fmt.Fprintf(os.Stdout, s)
	}
	return
}

type file struct {
	Fp   *os.File
	Size int64
}

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
		// Don't create a new file by default.
		f.Fp, err = os.OpenFile(getName(dir, fi.CreateTime), os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}

		f.Size = fi.Size

		return f, nil
	}

	// No file doesn't mean exist an error.
	return nil, nil
}

// Cleanup files of the directory.
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

	// Get the state of a file.
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

// Calling this function only when the queue is sorted, it will delete
// old log files based on the backups limit and the age limit.
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

func isValidName(name string) bool {
	matched, _ := regexp.MatchString(`^\d{4}(_\d{2}){5}_\d{9}$`, name)
	return matched
}

// Convert the standard file name to the time struct.
func name2time(name string) time.Time {
	var year, month, day, hour, min, sec, nsec int
	name = strings.Replace(name, "_", " ", -1)
	fmt.Sscanf(name, "%d%d%d%d%d%d%d", &year, &month, &day, &hour, &min, &sec, &nsec)
	t := time.Date(year, time.Month(month), day, hour, min, sec, nsec, time.Local)
	return t
}

// Convert the time struct to the standard file name.
// Format: year_month_day_hour_min_sec_nsec
func time2name(t time.Time) string {
	return fmt.Sprintf("%04d_%02d_%02d_%02d_%02d_%02d_%09d", int(t.Year()), int(t.Month()),
		int(t.Day()), int(t.Hour()), int(t.Minute()), int(t.Second()), int(t.Nanosecond()))
}

// Get the complete file name.
func getName(dir string, t time.Time) string {
	return filepath.Join(dir, time2name(t))
}

func timeTag() string {
	t := time.Now()
	return fmt.Sprintf("[%04d-%02d-%02d %02d:%02d:%02d]",
		int(t.Year()), int(t.Month()), int(t.Day()), int(t.Hour()), int(t.Minute()), int(t.Second()))
}

// Location tag, 'Info' function won't use it.
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

// Check whether the user has the writeable privilege on the special directory
// by creating a new temp file on it, it's the surest way. If we depend on file
// bit, it will be failed when the file system mounted is read-only or exists
// the ACL extension.
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

func unbindDir(dir string) {
	dirMtx.Lock()
	defer dirMtx.Unlock()
	delete(dirMap, dir)
}
