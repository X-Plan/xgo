// xkwfilter.go
//
//      Copyright (C), blinklv. All rights reserved.
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-11-23
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2016-11-29

// go-xkwfilter基于Aho-Corasick算法实现了一个关键字过滤器.
package xkwfilter

import (
	"bufio"
	"container/list"
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
	cl int

	// 状态/输入转移表. 第一个纬度为当前状态,
	// 第二个纬度为输入字节.
	tm [][]int

	// 接受状态表, 表明该状态是否为
	// 接受状态.
	am []bool

	// 失败回溯表.
	fm []int

	// 记录状态在树中的深度.
	dm []int
}

func New(mask string, keywords ...string) *XKeywordFilter {
	xkwf := &XKeywordFilter{mask: []byte(mask)}

	// cm中的值为-1表示该字节不存在.
	for i := 0; i < 256; i++ {
		xkwf.cm[i] = -1
	}

	// 初始化字节映射表.
	for _, kw := range keywords {
		a := []byte(kw)

		for _, c := range a {
			if xkwf.cm[int(c)] == -1 {
				xkwf.cm[int(c)] = xkwf.cl
				xkwf.cl++
			}
		}
	}

	// 构造转移矩阵tm和状态接受表am.
	for _, kw := range keywords {
		xkwf.enter(kw)
	}

	for i, s := range xkwf.tm[0] {
		if s == -1 {
			xkwf.tm[0][i] = 0
		}
	}

	// 构造失效函数表fm和状态深度表dm.
	xkwf.constructFm()

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
		i, inc  int
		br      = bufio.NewReader(r)
		bw      = bufio.NewWriter(w)
		buf     = make([]byte, 0, 1024)

		// flag标识是否已经写入了mask.  Filter函数
		// 在进行关键字过滤的时候遵循最长匹配的原则.
		// 如果aab和aaab都是关键字的话, 会优先考虑
		// aaab, 将其替换为mask. 如果多个关键字重叠
		// 或者衔接在一起, 则会被替换为一个mask.
		mi   int
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

		case s == 0 && olds == 0:
			if err = bw.WriteByte(c); err != nil {
				return
			}
			n++
			flag = false
		case s > 0 && olds >= 0:
			buf = append(buf, c)
			if xkwf.am[s] {
				mi = len(buf)
			}
		case s >= 0 && olds == -1:
			i = len(buf) - xkwf.dm[s]
			if i >= mi {
				if mi > 0 && !flag {
					if inc, err = bw.Write(xkwf.mask); err != nil {
						return
					}
					n += inc
					flag = true
				}
				if inc, err = bw.Write(buf[mi:i]); err != nil {
					return
				}

				if inc > 0 {
					n += inc
					flag = false
				}

				mi = 0
				buf = buf[i:]
			}
			goto again
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
		if s == 0 {
			return 0
		} else {
			return -1
		}
	} else {
		return xkwf.tm[s][nc]
	}
}

func (xkwf *XKeywordFilter) enter(kw string) {
	if len(kw) == 0 {
		return
	}

	var (
		ns, i, s int
		a        = []byte(kw)
	)

	for i = 0; xkwf.getTm(s, a[i]) != -1; i++ {
		s = xkwf.getTm(s, a[i])
	}

	for ns = len(xkwf.tm); i < len(a); i++ {
		xkwf.setTm(s, a[i], ns)
		s, ns = ns, ns+1
	}

	// 将最后的终止状态设置为接受状态.
	xkwf.setAm(s)
}

// 对从tm中取值进行封装, 添加了分配资源的判断.
// 该函数只用于构造tm的过程中使用.
func (xkwf *XKeywordFilter) getTm(s int, c byte) int {
	if s >= len(xkwf.tm) {
		xkwf.newCodes()
	}
	return xkwf.tm[s][xkwf.cm[int(c)]]
}

// 对设置tm中的值进行封装, 添加了分配资源判断.
// 该函数只用于构造tm的过程中.
func (xkwf *XKeywordFilter) setTm(s int, c byte, ns int) {
	if ns >= len(xkwf.tm) {
		xkwf.newCodes()
	}
	xkwf.tm[s][xkwf.cm[int(c)]] = ns
}

func (xkwf *XKeywordFilter) newCodes() {
	codes := make([]int, xkwf.cl)
	for i := 0; i < xkwf.cl; i++ {
		codes[i] = -1
	}
	xkwf.tm = append(xkwf.tm, codes)
}

// 对设置am中的值进行封装, 添加了分配资源判断.
// 该函数只用于构造am的过程中.
func (xkwf *XKeywordFilter) setAm(s int) {
	if s >= len(xkwf.am) {
		xkwf.am = append(xkwf.am, make([]bool, s+1-len(xkwf.am))...)
	}
	xkwf.am[s] = true
}

func (xkwf *XKeywordFilter) constructFm() {
	var (
		nc, s, r, d int
		q           = list.New()
	)

	xkwf.fm = make([]int, len(xkwf.tm))
	xkwf.dm = make([]int, len(xkwf.tm))

	for _, s = range xkwf.tm[0] {
		if s != 0 {
			q.PushBack(s)
			xkwf.fm[s] = 0
			xkwf.dm[s] = 1
		}
	}

	for q.Len() > 0 {
		r = (q.Remove(q.Front())).(int)
		d = xkwf.dm[r]
		for nc, s = range xkwf.tm[r] {
			if s != -1 {
				q.PushBack(s)

				r = xkwf.fm[r]
				for xkwf.tm[r][nc] == -1 {
					r = xkwf.fm[r]
				}

				xkwf.fm[s] = xkwf.tm[r][nc]
				xkwf.dm[s] = d + 1
			}
		}
	}
}
