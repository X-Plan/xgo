// xconfig.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2018-01-24
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2018-01-24
package xlog

import (
	"fmt"
	"regexp"
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

// Import a readable format data to the XConfig instance.
func (xcfg *XConfig) Import(data map[string]interface{}) error {
}

// The readable format of 'max_size' field: NUMBER [N space] {kb|KB|mb|MB|gb|GB}
// You can add some spaces at the head, but I don't recommend it.
var reMaxSize = regexp.MustCompile(`^\s*(\d+)\s*(kb|KB|mb|MB|gb|GB)$`)

const (
	gigaByte = 1024 * megaByte
	megaByte = 1024 * kiloByte
	kiloByte = 1024
)

func parseMaxSize(str string) (int64, error) {
	results := reMaxSize.FindStringSubmatch(str)
	if len(results) != 3 {
		goto format_error
	}

	number, err := strconv.Atoi(results[1])
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
	results := reMaxAge.FindStringSubmatch(str)
	if len(results) != 3 {
		goto format_error
	}

	number, err := strconv.Atoi(results[1])
	if err != nil {
		goto format_error
	}

	switch results[2] {
	case "year":
		number = year2min
	case "month":
		number = month2min
	case "week":
		number = week2min
	case "day":
		number = day2min
	case "hour":
		number = hour2min
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

// Export a XConfig instance to a readable format data.
func (xcfg *XConfig) Export(data map[string]interface{}) error {
}
