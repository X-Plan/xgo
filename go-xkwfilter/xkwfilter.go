// xkwfilter.go
//
//      Copyright (C), blinklv. All rights reserved.
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-11-23
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2016-11-24

// go-xkwfilter基于Aho-Corasick算法实现了一个关键字过滤器.
package xkwfilter

import (
	"io"
)

var (
	Version = "1.0.0"
)

// XKeywordFilter是一个关键字过滤器的实体, 它将输入
// 流中的敏感字符替换成预定义的字符, 起到屏蔽的效果.
// XKeywordFilter在实现上为了减少空间的损耗, 其内部
// 结构并不太适合动态的更改. 同时考虑到存在并发场景,
// XKeywordFilter一旦被创建就不能进行关键字的添加和
// 删除.
type XKeywordFilter struct {
	mask string

	// 并不是每个字节都会出现在关键字中,
	// 因此对关键字中真正出现的字节进行
	// 编号.
	cm [256]int

	// 状态/输入转移表. 第一个纬度为当前状态,
	// 第二个纬度为输入字节.
	tm [][]int

	// 接受状态表, 表明该状态是否为
	// 接受状态.
	am []bool

	// 失败回溯表.
	fm []int
}

func New(mask string, keywords ...string) *XKeywordFilter {
	xkwf := &XKeywordFilter{mask: mask}

	// cm中的值为-1表示该字节不存在.
	for i := 0; i < 256; i++ {
		xkwf.cm[i] = -1
	}

	// 初始化字节映射表.
	var (
		n = 0
	)
	for _, kw := range keywords {
		a := []byte(kw)
		for _, c := range a {
			if xkwf.cm[int(c)] == -1 {
				xkwf.cm[int(c)] = n
				n++
			}
		}
	}

	return xkwf
}

// Filter功能类似io.Copy函数, 将r中的字节流替换为相应
// 的屏蔽字符串. 当出现EOF条件或出现错误的情况下该函数
// 才会返回. 一个成功的返回必须要求err为nil, 而不是err
// 为EOF, 因为Filter默认情况下就是要求r到达EOF.
func (xkwf *XKeywordFilter) Filter(w io.Writer, r io.ByteReader) (int, error) {
	var (
		rerr, werr error
		n          int
		c          byte
		s          = 0
	)

	for {
		if c, rerr = r.ReadByte(); rerr != nil && rerr != io.EOF {
			return n, rerr
		}

		nc = xkwf.cm[int(c)]

		for xkwf.transfer(s, nc) == -1 {
			s = xkwf.fm[s]
		}
		s = xkwf.transfer(s, nc)

		if xkwf.am[s] {
		}

		// 到达文件输入末尾, 退出循环.
		if rerr == io.EOF {
			break
		}
	}

	return n, nil
}

func (xkwf *XKeywordFilter) transfer(s int, nc int) int {
	if nc == -1 {
		return -1
	} else {
		return xkwf.tm[s][nc]
	}
}
