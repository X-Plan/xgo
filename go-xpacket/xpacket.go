// xpacket.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-01-25
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-01-25

// 一个简单的二进制协议编/解码包.
package xpacket

import (
	"encoding/binary"
	"errors"
	"io"
)

const (
	sop = "SOP"
	eop = "EOP"
)

// 数据编码格式为:
// SOP: 包起始标志 (Start Of Packet). [3 octet]
// LEN: 包体的长度. [4 octet, big endian]
// BODY: 包体. [LEN octet]
// EOP: 包结束标志 (End Of Packet). [3 octet]
func Encode(w io.Writer, data []byte) error {
	var (
		err error
	)

	if _, err = io.WriteString(w, sop); err != nil {
		return err
	}
	if err = binary.Write(w, binary.BigEndian, uint32(len(data))); err != nil {
		return err
	}
	if _, err = w.Write(data); err != nil {
		return err
	}
	if _, err = io.WriteString(w, eop); err != nil {
		return err
	}

	return nil
}

func Decode(r io.Reader) ([]byte, error) {
	var (
		err  error
		n    uint32
		data []byte
		buf  = make([]byte, 3)
	)

	if _, err = io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	if string(buf) != sop {
		return nil, errors.New("start of packet is invalid")
	}

	if err = binary.Read(r, binary.BigEndian, &n); err != nil {
		return nil, err
	}

	data = make([]byte, int(n))
	if _, err = io.ReadFull(r, data); err != nil {
		return nil, err
	}

	if _, err = io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	if string(buf) != eop {
		return nil, errors.New("end of packet is invalid")
	}

	return data, nil
}
