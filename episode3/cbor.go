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

// Can encode nil, false, true, and integers
func (e *Encoder) Encode(v interface{}) error {
	switch v.(type) {
	case nil:
		return e.writeHeader(majorSimpleValue, simpleValueNil)
	case bool:
		var minor byte
		if v.(bool) {
			minor = simpleValueTrue
		} else {
			minor = simpleValueFalse
		}
		return e.writeHeader(majorSimpleValue, minor)
	case uint, uint8, uint16, uint32, uint64, int, int8, int16, int32, int64:
		if v.(uint64) >= 0 {
			return e.writeInteger(v.(uint64))
		}
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
