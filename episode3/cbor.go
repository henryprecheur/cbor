// Implements CBOR encoding:
//
//   https://tools.ietf.org/html/rfc7049
//
package cbor

import (
	"encoding/binary"
	"errors"
	"io"
)

type Encoder struct {
	w io.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

var ErrNotImplemented = errors.New("Not Implemented")

func (e *Encoder) writeHeader(major, minor byte) error {
	h := byte((major << 5) | minor)
	_, err := e.w.Write([]byte{h})
	return err
}

// writeHeaderInteger writes out a header created from major and minor magic
// numbers and write the value v as a big endian value
func (e *Encoder) writeHeaderInteger(major, minor byte, v interface{}) error {
	if err := e.writeHeader(major, minor); err != nil {
		return err
	}
	return binary.Write(e.w, binary.BigEndian, v)
}

func (e *Encoder) writeInteger(i uint64) error {
	switch {
	case i <= 23:
		return e.writeHeader(majorPositiveInteger, byte(i))
	case i <= 0xff:
		return e.writeHeaderInteger(
			majorPositiveInteger, minorPositiveInt8, uint8(i),
		)
	case i <= 0xffff:
		return e.writeHeaderInteger(
			majorPositiveInteger, minorPositiveInt16, uint16(i),
		)
	case i <= 0xffffffff:
		return e.writeHeaderInteger(
			majorPositiveInteger, minorPositiveInt32, uint32(i),
		)
	default:
		return e.writeHeaderInteger(
			majorPositiveInteger, minorPositiveInt64, uint64(i),
		)
	}
}

// Can only encode nil, false, and true
func (enc *Encoder) Encode(v interface{}) error {
	switch v.(type) {
	case nil:
		var hdr = header(majorSimpleValue, simpleValueNil)
		var _, err = enc.w.Write([]byte{hdr})
		return err
	case bool:
		var hdr byte
		if v.(bool) {
			hdr = header(majorSimpleValue, simpleValueTrue)
		} else {
			hdr = header(majorSimpleValue, simpleValueFalse)
		}
		_, err := enc.w.Write([]byte{hdr})
		return err
	case uint64:
		return enc.writeInteger(v.(uint64))
	}

	return ErrNotImplemented
}

const (
	// major types
	majorPositiveInteger = 0
	majorSimpleValue     = 7

	// extended integers
	minorPositiveInt8  = 24
	minorPositiveInt16 = 25
	minorPositiveInt32 = 26
	minorPositiveInt64 = 27

	// simple values == major type 7
	simpleValueFalse = 20
	simpleValueTrue  = 21
	simpleValueNil   = 22
)

func header(major, additional byte) byte {
	return (major << 5) | additional
}
