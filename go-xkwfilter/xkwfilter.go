// xkwfilter.go
//
//      Copyright (C), blinklv. All rights reserved.
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-11-23
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2016-11-26

// go-xkwfilter基于Aho-Corasick算法实现了一个关键字过滤器.
package xkwfilter

import (
	"bufio"
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
	mask []byte

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

	// 以字节为单位, 所有关键字和mask中最大
	// 的长度.
	maxlen int
}

func New(mask string, keywords ...string) *XKeywordFilter {
	xkwf := &XKeywordFilter{mask: []byte(mask)}
	xkwf.maxlen = len(xkwf.mask)

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
		if len(a) > xkwf.maxlen {
			xkwf.maxlen = len(a)
		}

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
func (xkwf *XKeywordFilter) Filter(w io.Writer, r io.Reader) (n int, err error) {
	var (
		c       byte
		s, olds int
		inc, nc int
		br      = bufio.NewReader(r)
		bw      = bufio.NewWriter(w)
		buf     = make([]byte, 0, xkwf.maxlen)

		// flag标识是否已经写入了mask.  Filter函数
		// 在进行关键字过滤的时候遵循最长匹配的原则.
		// 如果aab和aaab都是关键字的话, 会优先考虑
		// aaab, 将其替换为mask. 如果多个关键字重叠
		// 或者衔接在一起, 则会被替换为一个mask.
		flag bool
	)

	defer bw.Flush()

	// Dead: s == -1 Origin: s == 0 Active: s > 0
	// 为了方便处理, 这里用s == -2表示Prepare状态.
	s = -2

	for {
		switch {
		case s == -1:
			// 执行了这步操作后s的取值只可能
			// 大于等于0.
			s, olds = xkwf.fm[olds], -1
			continue

		case s == 0:
			flag = false

			if olds == 0 {
				if err = bw.WriteByte(c); err != nil {
					return
				}
				n++
			} else {
				if inc, err = bw.Write(buf); err != nil {
					return
				}
				n += inc
				buf = buf[:0]
				goto again
			}

		case s > 0:
			if olds >= 0 {
				if xkwf.am[s] {
					if !flag {
						// 只有上次写入非mask的情况下才可以
						// 选择是否写入mask.
						if inc, err = bw.Write(xkwf.mask); err != nil {
							return
						}
						n += inc
						flag = true
					}
					buf = buf[:0]
				} else {
					buf = append(buf, c)
				}
			} else {
				goto again
			}
		case s == -2:
			// 该状态放在最后的位置, 因为执行该语句
			// 的可能性很小. 只有初始调用时才会执行.
			s = 0
		}

		if c, err = br.ReadByte(); err != nil {
			if err == io.EOF {
				err = nil
			}
			return
		}

	again:
		s, olds = xkwf.transfer(s, xkwf.cm[int(c)]), s
	}
}

func (xkwf *XKeywordFilter) transfer(s int, nc int) int {
	if nc == -1 {
		return -1
	} else {
		return xkwf.tm[s][nc]
	}
}
