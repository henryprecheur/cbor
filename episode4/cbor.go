// Implements CBOR encoding:
//
//   https://tools.ietf.org/html/rfc7049
//
package cbor

import (
	"encoding/binary"
	"errors"
	"io"
	"reflect"
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

func (e *Encoder) Encode(v interface{}) error {
	x := reflect.ValueOf(v)
	switch x.Kind() {
	case reflect.Invalid:
		// naked nil value == invalid type
		return e.writeHeader(majorSimpleValue, simpleValueNil)
	case reflect.Ptr:
		if x.IsNil() {
			return e.writeHeader(majorSimpleValue, simpleValueNil)
		} else {
			return e.Encode(reflect.Indirect(x).Interface())
		}
	case reflect.Bool:
		var minor byte
		if x.Bool() {
			minor = simpleValueTrue
		} else {
			minor = simpleValueFalse
		}
		return e.writeHeader(majorSimpleValue, minor)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return e.writeInteger(x.Uint())
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
