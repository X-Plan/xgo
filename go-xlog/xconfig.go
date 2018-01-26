// xconfig.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2018-01-24
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2018-01-26
package xlog

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"time"
)

// This configure type is used to create 'XLogger'.
type XConfig struct {
	// The directory to store log files. If it's empty, './log' directory
	// will be used by default.  If this directory doesn't exist, it will
	// be created automatically.
	Dir string `json:"dir" yaml:"dir"`

	// The max file size of each log file, the unit is byte. When the size
	// of current log file exceeds this limit, switching to new log file
	// for writing. This field should be greater than or equal to zero,
	// zero represent unlimited.
	MaxSize int64 `json:"max_size" yaml:"max_size"`

	// The max number of log files in the directory. The oldest log file
	// will be deleted when the number of log files exceeds this limit.
	// This field should be greater than or equal to zero, zero represent unlimited.
	MaxBackups int64 `json:"max_backups" yaml:"max_backups"`

	// The max age of each log file. When the age of a log file exceed this
	// limit, it will be deleted.
	MaxAge string `json:"max_age" yaml:"max_age"`

	// Log tag. If not set, the process name will be used by default.
	Tag string `json:"tag" yaml:"tag"`

	// Log level. Only the priority of operation is higher than this field,
	// messages can be written to a log file, otherwise written to standard
	// error output. Call 'Write' function will ignore this field.
	Level int `json:"level" yaml:"level"`
}

// Import a readable format data to the XConfig instance. Even through this function
// returns a error, some fields of the XConfig instance have been changed. So you should
// be careful with this feature.
func (xcfg *XConfig) Import(data map[string]interface{}) error {
	var err error

	if str, ok := (data["dir"]).(string); ok {
		xcfg.Dir = str
	}

	if str, ok := (data["max_size"]).(string); ok {
		if xcfg.MaxSize, err = parseMaxSize(str); err != nil {
			return fmt.Errorf("'max_size' %s", err)
		}
	}

	maxBackups := reflect.ValueOf(data["max_backups"])
	switch maxBackups.Kind() {
	case reflect.Float32, reflect.Float64:
		xcfg.MaxBackups = int64(maxBackups.Float())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		xcfg.MaxBackups = maxBackups.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		xcfg.MaxBackups = int64(maxBackups.Uint())
	}

	if str, ok := (data["max_age"]).(string); ok {
		if xcfg.MaxAge, err = parseMaxAge(str); err != nil {
			return fmt.Errorf("'max_age' %s", err)
		}
	}

	if str, ok := (data["tag"]).(string); ok {
		xcfg.Tag = str
	}

	if str, ok := (data["level"]).(string); ok {
		if xcfg.Level, err = parseLevel(str); err != nil {
			return fmt.Errorf("'level' %s", err)
		}
	}

	return nil
}

// The readable format of 'max_size' field: NUMBER [N space] {kb|KB|mb|MB|gb|GB}
// You can add some spaces at the head, but I don't recommend it.
var reMaxSize = regexp.MustCompile(`^\s*(\d+)\s*(b|B|kb|KB|mb|MB|gb|GB)$`)

const (
	gigaByte = 1024 * megaByte
	megaByte = 1024 * kiloByte
	kiloByte = 1024
)

func parseMaxSize(str string) (int64, error) {
	var (
		err     error
		number  int
		results = reMaxSize.FindStringSubmatch(str)
	)

	if len(results) != 3 {
		goto format_error
	}

	number, err = strconv.Atoi(results[1])
	if err != nil {
		goto format_error
	}

	switch results[2] {
	case "gb", "GB":
		number *= gigaByte
	case "mb", "MB":
		number *= megaByte
	case "kb", "KB":
		number *= kiloByte
	}

	return int64(number), nil

format_error:
	return -1, fmt.Errorf("invalid format (%s)", str)
}

// The readable format of 'max_age' field: NUMBER [N space] {min|hour|day|week|month|year}
// You can add some spaces at the head, but I don't recommend it.
var reMaxAge = regexp.MustCompile(`^\s*(\d+)\s*(min|hour|day|week|month|year)$`)

const (
	hour2min  = 60
	day2min   = 24 * hour2min
	week2min  = 7 * day2min
	month2min = 31 * day2min  // We think each month has 31 days.
	year2min  = 366 * day2min // We think each year has 366 days.
)

func parseMaxAge(str string) (string, error) {
	var (
		err     error
		number  int
		results = reMaxAge.FindStringSubmatch(str)
	)
	if len(results) != 3 {
		goto format_error
	}

	number, err = strconv.Atoi(results[1])
	if err != nil {
		goto format_error
	}

	switch results[2] {
	case "year":
		number *= year2min
	case "month":
		number *= month2min
	case "week":
		number *= week2min
	case "day":
		number *= day2min
	case "hour":
		number *= hour2min
	}

	return fmt.Sprintf("%dm", number), nil

format_error:
	return "", fmt.Errorf("invalid format (%s)", str)
}

// The readable format of 'level' field: {fatal|error|warn|info|debug}
// You can add some spaces at the head, but I don't recommend it.
var reLevel = regexp.MustCompile(`^\s*(fatal|error|warn|info|debug)$`)

func parseLevel(str string) (int, error) {
	results := reLevel.FindStringSubmatch(str)
	if len(results) != 2 {
		return -1, fmt.Errorf("unknown level (%s)", str)
	}

	var level int

	switch results[1] {
	case "fatal":
		level = FATAL
	case "error":
		level = ERROR
	case "warn":
		level = WARN
	case "info":
		level = INFO
	case "debug":
		level = DEBUG
	}

	return level, nil
}

var levelReadable = [...]string{"", "fatal", "error", "warn", "info", "debug"}

// Export a XConfig instance to a readable format data.
func (xcfg *XConfig) Export(data map[string]interface{}) error {
	if len(xcfg.Dir) != 0 {
		data["dir"] = xcfg.Dir
	}
	if xcfg.MaxSize > 0 {
		data["max_size"] = readableMaxSize(int(xcfg.MaxSize))
	}
	if xcfg.MaxBackups > 0 {
		data["max_backups"] = xcfg.MaxBackups
	}
	if len(xcfg.MaxAge) != 0 {
		if maxAge, err := time.ParseDuration(xcfg.MaxAge); err != nil {
			return fmt.Errorf("invalid 'MaxAge' (%s)", err)
		} else if min := int(maxAge / time.Minute); min > 0 {
			// The 'MaxAge' should be greater than or equal to one minute at least.
			data["max_age"] = readableMaxAge(min)
		}
	}
	if len(xcfg.Tag) != 0 {
		data["tag"] = xcfg.Tag
	}
	if xcfg.Level > 0 && xcfg.Level < len(levelReadable) {
		data["level"] = levelReadable[xcfg.Level]
	}

	return nil
}

// When we transform 'MaxSize' to its readable format, which maybe loses precision,
// but it's less than one percent.
func readableMaxSize(maxSize int) string {
	if gb, r := maxSize/gigaByte, maxSize%gigaByte; 100*r < gb {
		return fmt.Sprintf("%d GB", gb)
	}

	if mb, r := maxSize/megaByte, maxSize%megaByte; 100*r < mb {
		return fmt.Sprintf("%d MB", mb)
	}

	if kb, r := maxSize/kiloByte, maxSize%kiloByte; 100*r < kb {
		return fmt.Sprintf("%d KB", kb)
	}

	return fmt.Sprintf("%d B", maxSize)
}

// When we transform 'MaxAge' to its readable format, which maybe loses precision,
// but it's less than one percent.
func readableMaxAge(min int) string {
	if year, r := min/year2min, min%year2min; 100*r < year {
		return fmt.Sprintf("%d year", year)
	}
	if month, r := min/month2min, min%month2min; 100*r < month {
		return fmt.Sprintf("%d month", month)
	}
	if week, r := min/week2min, min%week2min; 100*r < week {
		return fmt.Sprintf("%d week", week)
	}
	if day, r := min/day2min, min%day2min; 100*r < day {
		return fmt.Sprintf("%d day", day)
	}
	if hour, r := min/hour2min, min%hour2min; 100*r < hour {
		return fmt.Sprintf("%d hour", hour)
	}
	return fmt.Sprintf("%d min", min)
}
