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
		// Examples from CBOR spec
		{Value: 0, Expected: []byte{0x00}},
		{Value: 1, Expected: []byte{0x01}},
		{Value: 10, Expected: []byte{0x0a}},
		{Value: 23, Expected: []byte{0x17}},
		{Value: 24, Expected: []byte{0x18, 0x18}},
		{Value: 25, Expected: []byte{0x18, 0x19}},
		{Value: 100, Expected: []byte{0x18, 0x64}},
		{Value: 1000, Expected: []byte{0x19, 0x03, 0xe8}},
		{Value: 1000000, Expected: []byte{0x1a, 0x00, 0x0f, 0x42, 0x40}},
		{
			Value: 1000000000000,
			Expected: []byte{
				0x1b, 0x00, 0x00, 0x00, 0xe8, 0xd4, 0xa5, 0x10, 0x00,
			},
		},
		{
			Value: 18446744073709551615,
			Expected: []byte{
				0x1b, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			},
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%d", c.Value), func(t *testing.T) {
			testEncoder(t, uint64(c.Value), nil, c.Expected)
		})
	}
}
