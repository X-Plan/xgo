// wrap.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-02-07
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-06-29

package xlog

type FatalWrapper struct{ *XLogger }

func (fw FatalWrapper) Write(data []byte) (int, error) {
	return fw.XLogger.output(FATAL, string(data))
}

type ErrorWrapper struct{ *XLogger }

func (ew ErrorWrapper) Write(data []byte) (int, error) {
	return ew.XLogger.output(ERROR, string(data))
}

type WarnWrapper struct{ *XLogger }

func (ww WarnWrapper) Write(data []byte) (int, error) {
	return ww.XLogger.output(WARN, string(data))
}

type InfoWrapper struct{ *XLogger }

func (iw InfoWrapper) Write(data []byte) (int, error) {
	return iw.XLogger.output(INFO, string(data))
}

type DebugWrapper struct{ *XLogger }

func (dw DebugWrapper) Write(data []byte) (int, error) {
	return dw.XLogger.output(DEBUG, string(data))
}
