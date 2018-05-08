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

func TestNilTyped(t *testing.T) {
	var i *int = nil
	testEncoder(t, i, nil, []byte{0xf6})

	var v interface{} = nil
	testEncoder(t, v, nil, []byte{0xf6})
}

func TestPointer(t *testing.T) {
	i := uint(10)
	pi := &i // pi is a *uint

	// should output the number 10
	testEncoder(t, pi, nil, []byte{0x0a})
}

func TestBool(t *testing.T) {
	testEncoder(t, false, nil, []byte{0xf4})
	testEncoder(t, true, nil, []byte{0xf5})
}

func TestIntSmall(t *testing.T) {
	for i := 0; i <= 23; i++ {
		var expected = []byte{byte(i)}
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

func TestByteString(t *testing.T) {
	var cases = []struct {
		Value    []byte
		Expected []byte
	}{
		{Value: []byte{}, Expected: []byte{0x40}},
		{Value: []byte{1, 2, 3, 4}, Expected: []byte{0x44, 0x01, 0x02, 0x03, 0x04}},
		{
			Value:    []byte("hello"),
			Expected: []byte{0x45, 0x68, 0x65, 0x6c, 0x6c, 0x6f},
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%v", c.Value), func(t *testing.T) {
			testEncoder(t, c.Value, nil, c.Expected)
		})
	}

	// for arrays
	t.Run("array", func(t *testing.T) {
		a := [...]byte{1, 2}
		testEncoder(t, &a, nil, []byte{0x42, 1, 2})
	})
}

func TestUnicodeString(t *testing.T) {
	var cases = []struct {
		Value    string
		Expected []byte
	}{
		{Value: "", Expected: []byte{0x60}},
		{Value: "IETF", Expected: []byte{0x64, 0x49, 0x45, 0x54, 0x46}},
		{Value: "\"\\", Expected: []byte{0x62, 0x22, 0x5c}},
		{Value: "\u00fc", Expected: []byte{0x62, 0xc3, 0xbc}},
		{Value: "\u6c34", Expected: []byte{0x63, 0xe6, 0xb0, 0xb4}},
		// Invalid unicode codepoint can't be represented in Go string
		// {Value: "\ud800\udd51", Expected: []byte{0x64, 0xf0, 0x90, 0x85, 0x91}},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%s", c.Value), func(t *testing.T) {
			testEncoder(t, c.Value, nil, c.Expected)
		})
	}
}

func TestArray(t *testing.T) {
	var cases = []struct {
		Value    []interface{}
		Expected []byte
	}{
		{Value: []interface{}{}, Expected: []byte{0x80}},
		{Value: []interface{}{1, 2, 3}, Expected: []byte{0x83, 0x1, 0x2, 0x3}},
		{
			Value:    []interface{}{1, []interface{}{2, 3}, []interface{}{4, 5}},
			Expected: []byte{0x83, 0x01, 0x82, 0x02, 0x03, 0x82, 0x04, 0x05},
		},
		{
			Value: []interface{}{
				1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18,
				19, 20, 21, 22, 23, 24, 25,
			},
			Expected: []byte{
				0x98, 0x19, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
				0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12,
				0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x18, 0x18, 0x19,
			},
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%v", c.Value), func(t *testing.T) {
			testEncoder(t, c.Value, nil, c.Expected)
		})
	}
}
