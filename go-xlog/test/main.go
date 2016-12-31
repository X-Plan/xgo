// main.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-11-09
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2016-12-31

// 一个用于测试go-xlog包的工具.
package main

import (
	"flag"
	"fmt"
	"github.com/X-Plan/xgo/go-xlog"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	flagDir        = flag.String("dir", "", "log directory")
	flagMaxSize    = flag.Int64("max-size", 100*1024, "max size of log file (unit: byte)")
	flagMaxBackups = flag.Int64("max-backups", 50, "max number of backup log files")
	flagMaxAge     = flag.String("max-age", "3m", "max live time of log file")
	flagLevel      = flag.Int("level", xlog.INFO, "log level")
	flagTag        = flag.String("tag", "", "log tag")
	flagBlock      = flag.Int("block-size", 1024, "block size in every write op")
	flagNumber     = flag.Int("number", 1000, "number of write op")
	flagInterval   = flag.Duration("interval", 10*time.Millisecond, "the interval between two write ops")
	flagParallel   = flag.Int("parallel", 10, "number of go-routine")
	flagRaw        = flag.Bool("raw", false, "write op by the Write() function")
)

func main() {
	flag.Parse()

	var (
		err  error
		xcfg *xlog.XConfig
		xl   *xlog.XLogger
	)

	xcfg = &xlog.XConfig{
		Dir:        *flagDir,
		MaxSize:    *flagMaxSize,
		MaxBackups: *flagMaxBackups,
		MaxAge:     *flagMaxAge,
		Tag:        *flagTag,
		Level:      *flagLevel,
	}

	if xl, err = xlog.New(xcfg); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return
	}
	defer xl.Close()

	var wg = &sync.WaitGroup{}

	for gid := 0; gid < *flagParallel; gid++ {
		wg.Add(1)
		go func() {
			worker(xl)
			wg.Done()
		}()
	}
	wg.Wait()

}

func worker(xl *xlog.XLogger) {
	var (
		err error
		it  = *flagInterval
	)

	if *flagRaw {
		var block = []byte(strings.Repeat("A", *flagBlock))
		for i := 0; i < *flagNumber; i++ {
			if _, err = xl.Write(block); err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
			}
			time.Sleep(it)
		}
	} else {
		var block = strings.Repeat("A", *flagBlock)
		for i := 0; i < *flagNumber; i++ {
			switch i % 5 {
			case 0:
				err = xl.Fatal("%s", block)
			case 1:
				err = xl.Error("%s", block)
			case 2:
				err = xl.Warn("%s", block)
			case 3:
				err = xl.Info("%s", block)
			case 4:
				err = xl.Debug("%s", block)
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
			}
			time.Sleep(it)
		}
	}
}
