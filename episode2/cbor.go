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
		var _, err = enc.w.Write([]byte{0xf6})
		return err
	case bool:
		var err error
		if v.(bool) {
			_, err = enc.w.Write([]byte{0xf5}) // true
		} else {
			_, err = enc.w.Write([]byte{0xf4}) // false
		}
		return err
	}
	return ErrNotImplemented
}
