// xbufferpool.go
//
//		Copyright (C), blinklv. All rights reserved.
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2016-10-16
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2016-10-18

// go-xbufferpool实现了一个并发安全的缓冲池, 该缓冲池
// 可以用来管理和重用缓冲区.
package xbufferpool

import (
	"bytes"
	"errors"
)

// 版本信息
var Version = "1.0.0"

// 如果缓冲池已经关闭却还对其进行操作
// 会抛出该错误.
var ErrClosed = errors.New("XBufferPool has been closed")

var ErrXBufferIsNil = errors.New("XBuffer is nil")

// XBuffer是对bytes.Buffer一层封装. 你可以将
// XBuffer当成bytes.Buffer一样使用. XBuffer
// 与一个特定的连接池绑定, 使XBuffer的Close
// 操作可以有选择的将其归还到连接池. XBuffer
// 并不是并发安全的对象, 所以你不应该将其置
// 于并发环境.
type XBuffer struct {
	*bytes.Buffer
	xbp *XBufferPool
}

// 每个XBuffer都会与一个特定的XBufferPool相关联.
// XBuffer的关闭操作不是简单的忽略该资源让垃圾
// 回收进行处理, 而是有选择的将缓冲区归还到
// XBufferPool进行重用.
//
// 说明: XBuffer的设计思想和XConn很像, 但这里有
// 细微的不同. 它不需要一个特殊的标志位来表明不
// 在使用该XBuffer, 如果你真的不再使用, 只需要
// 忽略它然后等待垃圾回收帮你解决就行了.
func (xb *XBuffer) Close() error {
	err := xb.xbp.put(xb.Buffer)
	xb.Buffer = nil
	// 这里不用将相关的XBufferPool也设置为
	// 为空, 首先这个字段对用户不可见, 其次
	// 当用户再次调用XBuffer的Close函数时候
	// 不会因此而panic.
	return err
}

// 缓冲区池, 为了满足并发安全, 它在实现上用
// 到了Go的原生Channel.
type XBufferPool struct {
	buffers    chan *bytes.Buffer
	bufferSize int // 每个缓冲区的初始大小.
}

// 创建一个缓冲区池. capacity参数用于指定连接
// 池的最大值. bufferSize参数指定每个缓冲区的
// 初始大小, 如果为0则为系统默认大小. 调用者
// 应该检测返回值是否为nil.
func New(capacity int, bufferSize int) *XBufferPool {
	if capacity <= 0 {
		return nil
	}

	if bufferSize < 0 {
		return nil
	}

	xbp := &XBufferPool{
		buffers:    make(chan *bytes.Buffer, capacity),
		bufferSize: bufferSize,
	}

	return xbp
}

// 该函数从缓冲池中获取缓冲区. 如果缓冲池
// 中存在空闲缓冲区, 直接返回. 否则从系统
// 获取新的缓冲区.
func (xbp *XBufferPool) Get() (*XBuffer, error) {
	var (
		buf *bytes.Buffer
	)

	select {
	case buf = <-xbp.buffers:
		// 只有缓冲池已经关闭的情况下才会
		// 直接返回空值.
		if buf == nil {
			return nil, ErrClosed
		}
	default:
		// 没有空闲的缓冲区, 需要创建新的缓冲区.
		if xbp.bufferSize > 0 {
			buf = bytes.NewBuffer(make([]byte, 0, xbp.bufferSize))
		} else {
			buf = new(bytes.Buffer)
		}
	}
	return xbp.wrapBuffer(buf), nil
}

// 将*bytes.Buffer封装成XBuffer.
func (xbp *XBufferPool) wrapBuffer(buf *bytes.Buffer) *XBuffer {
	xb := &XBuffer{xbp: xbp}
	xb.Buffer = buf
	return xb
}

// 将缓冲区放回缓冲池, 该功能没有直接提供给用户.
// 而是通过XBuffer的Close函数间接让用户使用.
func (xbp *XBufferPool) put(buf *bytes.Buffer) (err error) {
	// 空缓冲区直接拒绝.
	if buf == nil {
		err = ErrXBufferIsNil
		return
	}

	// 因为XBufferPool在并发环境, 所以put操作的
	// 时候XBufferPool可能已经关闭. 然而写关闭的
	// 管道不像读关闭的管道那么温和, 所抛出的异常
	// 可能直接导致程序的crash(程序没有在外围对
	// 异常进行捕获). 因此这里对异常进行捕获, 同时
	// 忽略该Buffer,并返回相应的错误信息给调用方.
	defer func() {
		if tmpErr := recover(); tmpErr != nil {
			err = ErrClosed
		}
	}()

	// 将内容重置.
	buf.Reset()

	select {
	// 缓冲池未关闭且存在空闲空间, 则归还成功.
	case xbp.buffers <- buf:
	default:
		// 缓冲池无空闲空间, 则直接忽略该缓冲区,
		// 让Golang的垃圾回收来处理.
	}
	return
}

// 关闭缓冲池.
func (xbp *XBufferPool) Close() (err error) {
	defer func() {
		if tmpErr := recover(); tmpErr != nil {
			err = ErrClosed
		}
	}()
	close(xbp.buffers)
	// 将所有空闲的buffer忽略掉.
	for {
		select {
		case <-xbp.buffers:
		default:
			break
		}
	}
	return
}

// 获取缓冲池当前的大小.
func (xbp *XBufferPool) Size() int {
	return len(xbp.buffers)
}
