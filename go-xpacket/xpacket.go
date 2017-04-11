// xpacket.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-01-25
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-04-12

// A simple binary protocol encode/decode package.
package xpacket

import (
	"encoding/binary"
	"errors"
	"io"
)

const (
	sop = "SOP" // start of packet
	eop = "EOP" // end of packet
)

// Packet Format:
// [SOP (3 octet)][LEN (4 octet, big endian)][BODY (n octet)][EOP (3 octet)]
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
