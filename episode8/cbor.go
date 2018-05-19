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

func (e *Encoder) writeInteger(major byte, i uint64) error {
	switch {
	case i <= 23:
		return e.writeHeader(major, byte(i))
	case i <= 0xff:
		return e.writeHeaderInteger(major, minorInt8, uint8(i))
	case i <= 0xffff:
		return e.writeHeaderInteger(major, minorInt16, uint16(i))
	case i <= 0xffffffff:
		return e.writeHeaderInteger(major, minorInt32, uint32(i))
	default:
		return e.writeHeaderInteger(major, minorInt64, uint64(i))
	}
}

func (e *Encoder) writeByteString(s []byte) error {
	if err := e.writeInteger(majorByteString, uint64(len(s))); err != nil {
		return err
	}
	_, err := e.w.Write(s)
	return err
}

func (e *Encoder) writeUnicodeString(s string) error {
	if err := e.writeInteger(majorUnicodeString, uint64(len(s))); err != nil {
		return err
	}
	_, err := io.WriteString(e.w, s)
	return err
}

func (e *Encoder) writeArray(v reflect.Value) error {
	if err := e.writeInteger(majorArray, uint64(v.Len())); err != nil {
		return err
	}
	for i := 0; i < v.Len(); i++ {
		if err := e.encode(v.Index(i)); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) writeMap(v reflect.Value) error {
	if err := e.writeInteger(majorMap, uint64(v.Len())); err != nil {
		return err
	}

	for _, key := range v.MapKeys() {
		e.encode(key)
		e.encode(v.MapIndex(key))
	}
	return nil
}

func (e *Encoder) Encode(v interface{}) error {
	return e.encode(reflect.ValueOf(v))
}

func (e *Encoder) encode(x reflect.Value) error {
	switch x.Kind() {
	case reflect.Invalid:
		// naked nil value == invalid type
		return e.writeHeader(majorSimpleValue, simpleValueNil)
	case reflect.Interface:
		return e.encode(x.Elem())
	case reflect.Ptr:
		if x.IsNil() {
			return e.writeHeader(majorSimpleValue, simpleValueNil)
		} else {
			return e.encode(reflect.Indirect(x))
		}
	case reflect.Bool:
		var minor byte
		if x.Bool() {
			minor = simpleValueTrue
		} else {
			minor = simpleValueFalse
		}
		return e.writeHeader(majorSimpleValue, minor)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var i = x.Int()
		if i < 0 {
			return e.writeInteger(majorNegativeInteger, uint64(-(i + 1)))
		} else {
			return e.writeInteger(majorPositiveInteger, uint64(i))
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return e.writeInteger(majorPositiveInteger, x.Uint())
	case reflect.Array:
		// Create slice from array
		var n = reflect.New(x.Type())
		n.Elem().Set(x)
		x = reflect.Indirect(n).Slice(0, x.Len())
		fallthrough
	case reflect.Slice:
		if x.Type().Elem().Kind() == reflect.Uint8 {
			return e.writeByteString(x.Bytes())
		}
		return e.writeArray(x)
	case reflect.String:
		return e.writeUnicodeString(x.String())
	case reflect.Map:
		return e.writeMap(x)
	}
	return ErrNotImplemented
}

const (
	// major types
	majorPositiveInteger = 0
	majorNegativeInteger = 1
	majorByteString      = 2
	majorUnicodeString   = 3
	majorArray           = 4
	majorMap             = 5
	majorSimpleValue     = 7

	// extended integers
	minorInt8  = 24
	minorInt16 = 25
	minorInt32 = 26
	minorInt64 = 27

	// simple values == major type 7
	simpleValueFalse = 20
	simpleValueTrue  = 21
	simpleValueNil   = 22
)
