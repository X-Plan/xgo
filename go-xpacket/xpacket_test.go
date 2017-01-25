// xpacket_test.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-01-25
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-01-25
package xpacket

import (
	"bytes"
	"github.com/X-Plan/xgo/go-xassert"
	"github.com/X-Plan/xgo/go-xrandstring"
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
