package cbor

import (
	"bytes"
	"fmt"
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

func TestIntSmall(t *testing.T) {
	for i := 0; i <= 23; i++ {
		var expected = []byte{header(majorPositiveInteger, byte(i))}
		testEncoder(t, uint64(i), nil, expected)
	}
}

func TestIntBig(t *testing.T) {
	var cases = []struct {
		Value    uint64
		Expected []byte
	}{
		// smallest 8 bit value
		{
			Value:    24,
			Expected: []byte{header(majorPositiveInteger, positiveInt8), 24},
		},
		// biggest 8 bit value
		{
			Value:    0xff,
			Expected: []byte{header(majorPositiveInteger, positiveInt8), 0xff},
		},
		// smallest 16 bits value
		{
			Value:    0xff + 1,
			Expected: []byte{header(majorPositiveInteger, positiveInt16), 1, 0},
		},
		// biggest 16 bits value
		{
			Value:    0xffff,
			Expected: []byte{header(majorPositiveInteger, positiveInt16), 0xff, 0xff},
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%d", c.Value), func(t *testing.T) {
			testEncoder(t, uint64(c.Value), nil, c.Expected)
		})
	}
}
