// pid.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-11-14
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-11-14

// This package is used to store and get the PID information of a process.
package xpid

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
)

// Write the PID information of the current process to a specific file.
func Set(pidfile string) error {
	if pidfile == "" {
		return fmt.Errorf("PID file can't be empty")
	}

	err := os.MkdirAll(filepath.Dir(pidfile), os.FileMode(07555))
	if err != nil {
		return err
	}

	temp, err := ioutil.TempFile(filepath.Dir(pidfile), filepath.Base(pidfile))
	if err != nil {
		return err
	}

	defer func() {
		// If failed, delete the temporary file, otherwise rename
		// it to pidfile.
		if err != nil {
			os.Remove(temp.Name())
		}
	}()

	if err = os.Chmod(temp.Name(), os.FileMode(0644)); err != nil {
		return err
	}

	if _, err = fmt.Fprintf(temp, "%d", os.Getpid()); err != nil {
		return err
	}

	if err = temp.Close(); err != nil {
		return err
	}

	err = os.Rename(temp.Name(), pidfile)
	return err
}

// Get the PID information from a specific file.
func Get(pidfile string) (int, error) {
	if pidfile == "" {
		return 0, fmt.Errorf("PID file can't be empty")
	}

	buf, err := ioutil.ReadFile(pidfile)
	if err != nil {
		return 0, fmt.Errorf("read PID file (%s) failed (%s)", pidfile, err)
	}

	pid, err := strconv.Atoi(string(bytes.TrimSpace(buf)))
	if err != nil {
		return 0, fmt.Errorf("parsing PID failed (%s)", err)
	}

	return pid, nil
}
