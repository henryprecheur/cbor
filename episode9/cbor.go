// Implements CBOR encoding:
//
//   https://tools.ietf.org/html/rfc7049
//
package cbor

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"math/bits"
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

func (e *Encoder) writeStruct(v reflect.Value) error {
	type fieldKeyValue struct {
		Name  string
		Value reflect.Value
	}
	var fields []fieldKeyValue
	// Iterate over each field and add its key & value to fields
	for i := 0; i < v.NumField(); i++ {
		var fType = v.Type().Field(i)
		var fValue = v.Field(i)
		var tag = fType.Tag.Get("cbor")
		if tag == "-" {
			continue
		}
		name, opts := parseTag(tag)
		// with the option omitempty skip the value if it's empty
		if opts.Contains("omitempty") && isEmptyValue(fValue) {
			continue
		}
		if name == "" {
			name = fType.Name
		}
		fields = append(fields, fieldKeyValue{Name: name, Value: fValue})
	}
	if err := e.writeInteger(majorMap, uint64(len(fields))); err != nil {
		return err
	}
	for _, kv := range fields {
		if err := e.writeUnicodeString(kv.Name); err != nil {
			return err
		}
		if err := e.encode(kv.Value); err != nil {
			return err
		}
	}
	return nil
}

const (
	float16ExpBits  = 5
	float16MantBits = 10
	float16ExpBias  = 15
	float32ExpBits  = 8
	float32MantBits = 23
	float64ExpBits  = 11
	float64ExpBias  = 1023
	float64MantBits = 52

	// Minimum number of trailing zeros needed in the mantissa
	float16MinZeros = float64MantBits - float16MantBits
	float32MinZeros = float64MantBits - float32MantBits

	expMask  = (1 << float64ExpBits) - 1
	mantMask = (1 << float64MantBits) - 1
)

func (e *Encoder) writeFloat16(negative bool, exp uint16, mant uint64) error {
	if err := e.writeHeader(majorSimpleValue, minorFloat16); err != nil {
		return err
	}
	var output uint16
	if negative {
		output = 1 << 15
	}
	output |= exp << float16MantBits
	output |= uint16(mant >> (float64MantBits - float16MantBits))
	return binary.Write(e.w, binary.BigEndian, output)
}

func unpackFloat64(f float64) (exp int, mant uint64) {
	var r = math.Float64bits(f)
	exp = int(r>>float64MantBits&expMask) - float64ExpBias
	mant = r & mantMask
	return
}

func (e *Encoder) writeFloat(input float64) error {
	// First check if we have a special value: 0, NaN, Inf, -Inf
	switch {
	case input == 0:
		return e.writeFloat16(math.Signbit(input), 0, 0)
	case math.IsNaN(input):
		return e.writeFloat16(
			math.Signbit(input),
			(1<<float16ExpBits)-1,
			1<<(float64MantBits-1),
		)
	case math.IsInf(input, 0):
		return e.writeFloat16(math.Signbit(input), (1<<float16ExpBits)-1, 0)
	}

	var (
		exp, mant     = unpackFloat64(input)
		trailingZeros = bits.TrailingZeros64(mant)
	)
	switch {
	case (-14 <= exp) && (exp <= 15) && (trailingZeros >= float16MinZeros):
		return e.writeFloat16(math.Signbit(input), uint16(exp+float16ExpBias), mant)
	case (-126 <= exp) && (exp <= 127) && (trailingZeros >= float32MinZeros):
		if err := e.writeHeader(majorSimpleValue, minorFloat32); err != nil {
			return err
		}
		return binary.Write(e.w, binary.BigEndian, float32(input))
	default:
		if err := e.writeHeader(majorSimpleValue, minorFloat64); err != nil {
			return err
		}
		return binary.Write(e.w, binary.BigEndian, input)
	}
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
	case reflect.Struct:
		return e.writeStruct(x)
	case reflect.Float32, reflect.Float64:
		return e.writeFloat(x.Float())
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

	// floating point types
	minorFloat16 = 25
	minorFloat32 = 26
	minorFloat64 = 27

	// simple values == major type 7
	simpleValueFalse = 20
	simpleValueTrue  = 21
	simpleValueNil   = 22
)
