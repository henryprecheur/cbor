package cbor

import (
	"bytes"
	"fmt"
	"math"
	"testing"
)

// testEncoder test the CBOR encoder with the value v, and verify that err, and
// expected match what's returned and written by the encoder.
func testEncoder(t *testing.T, v interface{}, expected []byte) {
	// buffer is where we write the CBOR encoded values
	var buffer = bytes.Buffer{}
	// create a new encoder writing to buffer, and encode v with it
	var e = NewEncoder(&buffer).Encode(v)

	if e != nil {
		t.Fatalf("err: %#v != nil with %#v", e, v)
	}

	if !bytes.Equal(buffer.Bytes(), expected) {
		t.Fatalf(
			"(%#v) %#v != %#v", v, buffer.Bytes(), expected,
		)
	}
}

func TestNil(t *testing.T) {
	testEncoder(t, nil, []byte{0xf6})
}

func TestNilTyped(t *testing.T) {
	var i *int = nil
	testEncoder(t, i, []byte{0xf6})

	var v interface{} = nil
	testEncoder(t, v, []byte{0xf6})
}

func TestPointer(t *testing.T) {
	i := uint(10)
	pi := &i // pi is a *uint

	// should output the number 10
	testEncoder(t, pi, []byte{0x0a})
}

func TestBool(t *testing.T) {
	testEncoder(t, false, []byte{0xf4})
	testEncoder(t, true, []byte{0xf5})
}

func TestIntSmall(t *testing.T) {
	for i := 0; i <= 23; i++ {
		var expected = []byte{byte(i)}
		testEncoder(t, uint64(i), expected)
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
			testEncoder(t, uint64(c.Value), c.Expected)
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
			testEncoder(t, c.Value, c.Expected)
		})
	}

	// for arrays
	t.Run("array", func(t *testing.T) {
		a := [...]byte{1, 2}
		testEncoder(t, &a, []byte{0x42, 1, 2})
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
			testEncoder(t, c.Value, c.Expected)
		})
	}
}

func TestNegativeIntegers(t *testing.T) {
	var cases = []struct {
		Value    int64
		Expected []byte
	}{
		{Value: -1, Expected: []byte{0x20}},
		{Value: -10, Expected: []byte{0x29}},
		{Value: -100, Expected: []byte{0x38, 0x63}},
		{Value: -1000, Expected: []byte{0x39, 0x03, 0xe7}},
		{
			Value: math.MinInt64,
			Expected: []byte{
				0x3b, 0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			},
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%d", c.Value), func(t *testing.T) {
			testEncoder(t, c.Value, c.Expected)
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
			testEncoder(t, c.Value, c.Expected)
		})
	}
}

func TestMap(t *testing.T) {
	// {}
	t.Run("{}", func(t *testing.T) {
		testEncoder(t, map[struct{}]struct{}{}, []byte{0xa0})
	})
	// ["a", {"b": "c"}]
	t.Run("{\"a\", {\"b\": \"c\"}", func(t *testing.T) {
		testEncoder(
			t,
			[]interface{}{"a", map[string]string{"b": "c"}},
			[]byte{0x82, 0x61, 0x61, 0xa1, 0x61, 0x62, 0x61, 0x63},
		)
	})

	var cases = []struct {
		Value    interface{}
		Expected [][]byte
	}{
		{
			Value: map[int]int{1: 2, 3: 4},
			Expected: [][]byte{
				[]byte{0x01, 0x02}, // {1: 2}
				[]byte{0x03, 0x04}, // {3: 4}
			},
		},
		{
			Value: map[string]interface{}{"a": 1, "b": []int{2, 3}},
			Expected: [][]byte{
				[]byte{0x61, 0x61, 0x01},             // {"a": 1}
				[]byte{0x61, 0x62, 0x82, 0x02, 0x03}, // {"b": [2, 3]}
			},
		},
		{
			Value: map[string]string{
				"a": "A", "b": "B", "c": "C", "d": "D", "e": "E",
			},
			Expected: [][]byte{
				[]byte{0x61, 0x61, 0x61, 0x41}, // {"a": "A"}
				[]byte{0x61, 0x62, 0x61, 0x42}, // {"b": "B"}
				[]byte{0x61, 0x63, 0x61, 0x43}, // {"c": "C"}
				[]byte{0x61, 0x64, 0x61, 0x44}, // {"d": "D"}
				[]byte{0x61, 0x65, 0x61, 0x45}, // {"e": "E"}
			},
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%v", c.Value), func(t *testing.T) {
			var buffer bytes.Buffer

			if err := NewEncoder(&buffer).Encode(c.Value); err != nil {
				t.Fatalf("err: %#v != %#v with %#v", err, nil, c.Value)
			}

			var (
				header     = buffer.Bytes()[0]
				result     = buffer.Bytes()[1:]
				lengthMask = ^uint8(0) >> 3 // bit mask to extract the length
				length     = header & lengthMask
			)
			if header>>5 != majorMap {
				t.Fatalf("invalid major type: %#v", header)
			}

			if int(length) != len(c.Expected) {
				t.Fatalf("invalid length: %#v != %#v", length, len(c.Expected))
			}

			for _, kv := range c.Expected {
				if !bytes.Contains(result, kv) {
					t.Fatalf("key/value %#v not found in result", kv)
				}
				// remove the value from the result
				result = bytes.Replace(result, kv, []byte{}, 1)
			}

			// ensure there's left-over data
			if len(result) > 0 {
				t.Fatalf("leftover in result: %#v", result)
			}
		})
	}
}

func TestStruct(t *testing.T) {
	var cases = []struct {
		Value    interface{}
		Expected []byte
	}{
		{Value: struct{}{}, Expected: []byte{0xa0}},
		{
			Value: struct {
				a int
				b []int
			}{a: 1, b: []int{2, 3}},
			Expected: []byte{
				0xa2, 0x61, 0x61, 0x01, 0x61, 0x62, 0x82, 0x02, 0x03,
			},
		},
		{
			Value: struct {
				a string
				b string
				c string
				d string
				e string
			}{"A", "B", "C", "D", "E"},
			Expected: []byte{
				0xa5, 0x61, 0x61, 0x61, 0x41, 0x61, 0x62, 0x61, 0x42, 0x61,
				0x63, 0x61, 0x43, 0x61, 0x64, 0x61, 0x44, 0x61, 0x65, 0x61,
				0x45,
			},
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%v", c.Value), func(t *testing.T) {
			testEncoder(t, c.Value, c.Expected)
		})
	}
}

func TestStructTag(t *testing.T) {
	testEncoder(t,
		struct {
			AField int   `cbor:"a"`
			BField []int `cbor:"b"`
			Omit1  int   `cbor:"c,omitempty"`
			Omit2  int   `cbor:",omitempty"`
			Ignore int   `cbor:"-"`
		}{AField: 1, BField: []int{2, 3}, Ignore: 12345},
		[]byte{0xa2, 0x61, 0x61, 0x01, 0x61, 0x62, 0x82, 0x02, 0x03},
	)
}
