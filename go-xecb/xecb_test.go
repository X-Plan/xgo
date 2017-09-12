// xecb_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-09-12
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-09-12

package xecb

import (
	"bytes"
	"crypto/des"
	"fmt"
	"github.com/X-Plan/xgo/go-xassert"
	"testing"
)

var block, _ = des.NewTripleDESCipher([]byte("123456789012345678901234"))

func TestPanic(t *testing.T) {
	var (
		e = NewECBEncrypter(block)
		d = NewECBDecrypter(block)
	)

	xassert.NotNil(t, capture(func() {
		e.CryptBlocks(nil, make([]byte, block.BlockSize()+1))
	}))
	xassert.NotNil(t, capture(func() {
		e.CryptBlocks(make([]byte, block.BlockSize()*10), make([]byte, block.BlockSize()*20))
	}))
	xassert.IsNil(t, capture(func() {
		e.CryptBlocks(make([]byte, block.BlockSize()*10), make([]byte, block.BlockSize()*10))
	}))
	xassert.NotNil(t, capture(func() {
		d.CryptBlocks(nil, make([]byte, block.BlockSize()+1))
	}))
	xassert.NotNil(t, capture(func() {
		d.CryptBlocks(make([]byte, block.BlockSize()*10), make([]byte, block.BlockSize()*20))
	}))
	xassert.IsNil(t, capture(func() {
		d.CryptBlocks(make([]byte, block.BlockSize()*10), make([]byte, block.BlockSize()*10))
	}))
}

func TestECB(t *testing.T) {
	var text = `
    Somebody tell me.
    Why it feels more real when I dream than when I am awake.
    How can I know if my senses are lying?

    There is some fiction in your truth,
    and some truth in your fiction.
    To the truth, you must risk everything.

    Who are you?
    Am I alone?

    You are not alone.

                                        --- A Kid's Story
    `
	var (
		e   = NewECBEncrypter(block)
		d   = NewECBDecrypter(block)
		src = pkcs5padding([]byte(text), e.BlockSize())
		dst = make([]byte, len(src))
	)

	e.CryptBlocks(dst, src)
	d.CryptBlocks(dst, dst)

	xassert.Equal(t, string(pkcs5unpadding(dst)), text)
}

func capture(cb func()) (err error) {
	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("%s", x)
		}
	}()
	cb()
	return
}

func pkcs5padding(data []byte, bs int) []byte {
	n := bs - len(data)%bs
	return append(data, bytes.Repeat([]byte{byte(n)}, n)...)
}

func pkcs5unpadding(data []byte) []byte {
	n := len(data)
	return data[:(n - int(data[n-1]))]
}
