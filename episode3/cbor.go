// Implements CBOR encoding:
//
//   https://tools.ietf.org/html/rfc7049
//
package cbor

import (
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
		var _, err = enc.w.Write([]byte{hdr})
		return err
	}
	return ErrNotImplemented
}

const (
	// major types
	majorPositiveInteger = 0
	majorSimpleValue     = 7

	// simple values == major type 7
	simpleValueFalse = 20
	simpleValueTrue  = 21
	simpleValueNil   = 22
)

func header(major, additional byte) byte {
	return byte(major<<5) | additional
}
