package cbor

import (
	"bytes"
	"testing"
)

// testEncoder test the CBOR encoder with the value v, and verify that err, and
// expected match what's returned and written by the encoder.
func testEncoder(t *testing.T, v interface{}, err error, expected []byte) {
	// buffer is where we write the CBOR encoded values
	var buffer = bytes.Buffer{}
	// create a new encoder writing to buffer, and encode v with it
	var e = NewEncoder(&buffer).Encode(v)

	if e != err {
		t.Fatalf("err: %#v != %#v with %#v", e, err, v)
	}

	if !bytes.Equal(buffer.Bytes(), expected) {
		t.Fatalf(
			"(%#v) %#v != %#v", v, buffer.Bytes(), expected,
		)
	}
}

func TestNil(t *testing.T) {
	testEncoder(t, nil, nil, []byte{0xf6})
}

func TestBool(t *testing.T) {
	testEncoder(t, false, nil, []byte{0xf4})
	testEncoder(t, true, nil, []byte{0xf5})
}
