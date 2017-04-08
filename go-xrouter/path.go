// path.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-04-08
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-04-08

package xrouter

// A lazybuf is a lazily constructed path buffer, It supports
// append, reading previously appended bytes, and retrieving
// the final string. It does not allocate a buffer to hold the
// output until that output diverges from s.
type lazybuf struct {
	s   string
	buf []byte
	w   int
}

func (b *lazybuf) index(i int) byte {
	if b.buf != nil {
		return b.buf[i]
	}
	return b.s[i]
}

func (b *lazybuf) append(c byte) {
	if b.buf == nil {
		if b.w < len(b.s) && b.s[b.w] == c {
			b.w++
			return
		}
		b.buf = make([]byte, len(b.s))
		copy(b.buf, b.s[:b.w])
	}
	b.buf[b.w] = c
	b.w++
}

func (b *lazybuf) string() string {
	if b.buf == nil {
		return b.s[:b.w]
	}
	return string(b.buf[:b.w])
}
