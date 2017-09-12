// xecb.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-09-12
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-09-12

// This package implements ECB block mode to satisfy the cipher.BlockMode interface.
package xecb

import "crypto/cipher"

type ecb struct {
	block cipher.Block
	size  int
}

type encrypter ecb

func NewECBEncrypter(block cipher.Block) cipher.BlockMode {
	return &encrypter{block, block.BlockSize()}
}

func (e *encrypter) BlockSize() int {
	return e.size
}

func (e *encrypter) CryptBlocks(dst, src []byte) {
	if len(src)%e.size != 0 {
		panic("crypto/cipher: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}

	for len(src) > 0 {
		e.block.Encrypt(dst, src[:e.size])
		src = src[e.size:]
		dst = dst[e.size:]
	}
}

type decrypter ecb

func NewECBDecrypter(block cipher.Block) cipher.BlockMode {
	return &decrypter{block, block.BlockSize()}
}

func (d *decrypter) BlockSize() int {
	return d.size
}

func (d *decrypter) CryptBlocks(dst, src []byte) {
	if len(src)%d.size != 0 {
		panic("crypto/cipher: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}

	for len(src) > 0 {
		d.block.Decrypt(dst, src[:d.size])
		src = src[d.size:]
		dst = dst[d.size:]
	}
}
