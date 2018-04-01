// Implements CBOR encoding:
//
//   https://tools.ietf.org/html/rfc7049
//
package cbor

import (
        "io"
		"errors"
)

type Encoder struct {
        w   io.Writer
}

func NewEncoder(w io.Writer) *Encoder {
        return &Encoder{w: w}
}

var ErrNotImplemented = errors.New("Not Implemented")

// Can only encode nil
func (enc *Encoder) Encode(v interface{}) error {
    switch v.(type) {
    case nil:
        var _, err = enc.w.Write([]byte{0xf6})
        return err
    }
    return ErrNotImplemented
}
