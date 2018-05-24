package cbor

import (
	"bytes"
	"testing"
)

func TestNil(t *testing.T) {
	var buffer = bytes.Buffer{}
	var err = NewEncoder(&buffer).Encode(nil)

	if !(err == nil && bytes.Equal(buffer.Bytes(), []byte{0xf6})) {
		t.Fatalf(
			"%#v != %#v or %#v != %#v",
			err, nil, buffer.Bytes(), []byte{0xf6},
		)
	}
}
