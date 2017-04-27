// xpacket_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-01-25
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-04-27
package xpacket

import (
	"bytes"
	"github.com/X-Plan/xgo/go-xassert"
	"github.com/X-Plan/xgo/go-xrandstring"
	"io/ioutil"
	"os"
	"testing"
)

func TestErrorDecode(t *testing.T) {
	var (
		err error
		buf *bytes.Buffer
	)

	buf = bytes.NewBuffer([]byte("XOP"))
	_, err = Decode(buf)
	xassert.Match(t, err, `^start of packet is invalid$`)

	buf = bytes.NewBuffer([]byte{'S', 'O', 'P', 0, 0, 0, 0, 'X', 'O', 'P'})
	_, err = Decode(buf)
	xassert.Match(t, err, `^end of packet is invalid$`)
}

func TestSmallPacket(t *testing.T) {
	var buf = bytes.NewBuffer(make([]byte, 0, 256))
	for n := 16; n < 128; n++ {
		testPacket(t, n, buf)
		buf.Reset()
		testFormat(t, n, buf)
		buf.Reset()
	}
}

func TestMiddlePacket(t *testing.T) {
	var buf = bytes.NewBuffer(make([]byte, 0, 8192))
	for n := 1024; n < 4096; n++ {
		testPacket(t, n, buf)
		buf.Reset()
	}
}

func TestBigPacket(t *testing.T) {
	var buf = bytes.NewBuffer(make([]byte, 0, 65536))
	for n := 16384; n < 32768; n++ {
		testPacket(t, n, buf)
		buf.Reset()
	}
}

func testPacket(t *testing.T, n int, buf *bytes.Buffer) {
	var (
		before = []byte(xrandstring.Get(n))
		after  []byte
		err    error
	)

	err = Encode(buf, before)
	xassert.IsNil(t, err)
	after, err = Decode(buf)
	xassert.IsNil(t, err)
	xassert.Equal(t, after, before)
}

func testFormat(t *testing.T, n int, buf *bytes.Buffer) {
	xassert.IsNil(t, Encode(buf, []byte(xrandstring.Get(n))))
	xassert.Match(t, buf.String(), `^SOP.+EOP$`)
}

// Reading the 'xpacket.input' file and encoding the content of it,
// writing the result to 'xpacket.output' file.
func TestEncode(t *testing.T) {
	raw, err := ioutil.ReadFile("xpacket.input")
	if err == nil {
		ofile, err := os.OpenFile("xpacket.output", os.O_RDWR|os.O_CREATE, 0644)
		if err == nil {
			xassert.IsNil(t, Encode(ofile, raw))
		}
	}
}

// Reading the 'xpacket.output' file and decoding the content of it,
// comparing its content with the content of 'xpacket.input' file.
func TestDecode(t *testing.T) {
	raw, err := ioutil.ReadFile("xpacket.input")
	if err == nil {
		ofile, err := os.Open("xpacket.output")
		if err == nil {
			out, err := Decode(ofile)
			xassert.IsNil(t, err)
			xassert.Equal(t, bytes.Compare(out, raw), 0)
		}
	}
}
