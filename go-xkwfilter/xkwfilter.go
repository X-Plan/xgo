// xkwfilter.go
//
//      Copyright (C), blinklv. All rights reserved.
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-11-23
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2016-11-24

// go-xkwfilter基于Aho-Corasick算法, 实现了一个关键字
// 过滤器.
package xkwfilter

import (
	"io"
)

var (
	Version = "1.0.0"
)

// XKeywordFilter是一个关键字过滤器的实体, 它将输入
// 流中的敏感字符替换成预定义的字符, 起到屏蔽的效果.
// 出于效率方面的考虑, XKeywordFilter本身不支持动态
// 添加关键字 (因为这样会处于一种并发环境). 所以面对
// 新增关键字的情况需要重新创建XKeywordFilter实体.
type XKeywordFilter struct {
	mask  string
	nodes []*node // 节点的线性表示.
	tree  *node   // 节点的树状表示.
}

// 创建一个XKeywordFilter实体. mask为替换关键字的屏蔽
// 字符串, keywords为需要进行屏蔽的关键字.
func New(mask string, keywords ...string) (*XKeywordFilter, error) {
}

// Filter功能类似io.Copy函数, 将r中的字节流替换为相应
// 的屏蔽字符串. 当出现EOF条件或出现错误的情况下该函数
// 才会返回. 一个成功的返回必须要求err为nil, 而不是err
// 为EOF, 因为Filter默认情况下就是要求r到达EOF.
func (xkwf *XKeywordFilter) Filter(w io.Writer, r io.Reader) (n int64, err error) {
}

type node struct {
	value string
	child []*node
}
