Go CBOR encoder: Episode 9, floating point numbers
==================================================

This is a tutorial on how to write a [CBOR][] encoder in
[Go](https://golang.org/), where we’ll learn more about [reflection][] and type
introspection.

Read the previous episodes, each episode builds on the previous one:

[CBOR]: http://cbor.io/
[reflection]: https://godoc.org/reflect

- [Episode 1, getting started][ep1]
- [Episode 2, booleans][ep2]
- [Episode 3, positive integers][ep3]
- [Episode 4, reflect and pointers][ep4]
- [Episode 5, strings][ep5]
- [Episode 6, negative integers and arrays][ep6]
- [Episode 7, maps][ep7]
- [Episode 8, structs][ep8]

[ep1]: http://henry.precheur.org/scratchpad/2018-03-19T15%3A50%3A41-07%3A00
[ep2]: http://henry.precheur.org/scratchpad/20180402_181348
[ep3]: http://henry.precheur.org/scratchpad/20180420_110255
[ep4]: http://henry.precheur.org/scratchpad/20180504_101405
[ep5]: http://henry.precheur.org/scratchpad/20180513_094706
[ep6]: http://henry.precheur.org/scratchpad/20180520_094032
[ep7]: http://henry.precheur.org/scratchpad/20180527_114309
[ep8]: http://henry.precheur.org/scratchpad/20180604_083306
[rfc7049]: https://tools.ietf.org/html/rfc7049

----

Floating points numbers come in three varieties in CBOR:

- [float16](https://en.wikipedia.org/wiki/Half-precision_floating-point_format)
- [float32](https://en.wikipedia.org/wiki/Single-precision_floating-point_format)
- [float64](https://en.wikipedia.org/wiki/Double-precision_floating-point_format)

Golang only supports float32 & float64 natively, but we can build the smaller 16
bits floating point numbers ourselves.

We’ll do what we did with integers: we’ll minimize the size of the output by
encoding the information as tightly as possible. This means we’ll use 16
bits floats by default, and fall back to float 32 & 64 bits as needed. The only
criteria: don’t lose information / precision, we want our numbers to be exact.
Go uses [IEEE 754][ieee754_repr] floating point arithmetic, those numbers have
three parts:

[ieee754_repr]: https://en.wikipedia.org/wiki/IEEE_754#Representation_and_encoding_in_memory

- sign bit
- exponent
- fractional

You get the real number with the formula:

(-1)^s . 1.C . 2^exp

To represent 1.0 using this notation we’d have:

sign bit = 0
exponent = 0
mantissa = 00...0

(-1)^0 . 1.00000 . 2^0 = 1.0

To represent 1.5 it would be:

sign bit = 0
exponent = 0
fraction = 10...0

The mantissa would be 1.1 in binary which corresponds to 1.5 in decimal.

(-1)^0 . 1.5 . 2^0 = 1.5

Exponents are represented with biased integers: a bias is added so that the
smallest representable exponent is represented as 1, with 0 used for subnormal
numbers. This means that after we read the exponent in memory we’ll substract
the bias to get the real value of the exponent. For example 16bits floats’
exponent bias is 15, this means that if the exponent’ value in memory is 01111 =
15, its real value will be 0; if its value is 00001 = 1 the real value will be
-14.

For the fractional part we can trim off the excess zeros at the end without
losing precision. So what we’ll do is count how many trailing zeros we have in
the fractional part and choose the smallest possible type that can accomodate
the fractional part.

For example if you fractional part is 1.10111110101 we can use a float 32bits to
store it, but not a float 16 bits because we only have 10 bits for the
fractional part and the example before need 12 bits.

As a first step we’ll handle float 32 bits & float 64 bits first, and then
implement 16 bits floating point numbers.

Just like we did for integers we’ll minimize the size of our output by using the
smallest type possible without loosing precision.

Here’s 1.0 represented as a binary half precision floating point number:

    0 01111 0000000000

01111 is 15 in decimal, the exponent bias on a float16 is 15 therefor to get the
real number we calculate:

    (-1)^0 . 1.000000

All right now how are we going to decide when to use a type or another? We’ll
look at the exponent and the mantissa and depending of their value we’ll pick on
type over the other.

For example 16 floating numbers exponent must be between -14 and 15, therefor we
can’t encode numbers with an exponent out of this range. For the mantissa we’ll
have to verify if we can keep all the significant bits when converting.

When we truncate the mantissa we want to ensure we’re not chopping off any 1’s,
to acheive this we’ll count the number of trailing zeros. If we have enough
trailing zeros we can then encode the floating point number in this particular
type.

For example we :




----

We’ll start small and add a few tests at each step as we add support for more
features. Let’s get started with something easy: 32 & 64 floats, that Go has
native support for. As usual we look at the example in the [spec][rfc7049] and
see that: 100,000.0 can be encoded exactly with a float32, while 1.1 can only
be represented by a float64. We’ll start with those two:

    func TestFloat(t *testing.T) {
        var cases = []struct {
            Value    float64
            Expected []byte
        }{
            {
                Value:    1.1,
                Expected: []byte{0xfb, 0x3f, 0xf1, 0x99, 0x99, 0x99, 0x99, 0x99, 0x9a},
            },
            {Value: 100000.0, Expected: []byte{0xfa, 0x47, 0xc3, 0x50, 0x00}},
        }

        for _, c := range cases {
            t.Run(fmt.Sprintf("%v", c.Value), func(t *testing.T) {
                testEncoder(t, c.Value, c.Expected)
            })
        }
    }

We’ll add a new function writeFloat. To decide whether to use float32 or float64
we will convert the original float64 value into a float32 and back. If the value
is still the same we can safely encode the number as a float32:

    const (
        // floating point types
        minorFloat16 = 25
        minorFloat32 = 26
        minorFloat64 = 27
    )

    func (e *Encoder) writeFloat(input float64) error {

        if float64(float32(input)) == input {
            if err := e.writeHeader(majorSimpleValue, minorFloat32); err != nil {
                return err
            }
            return binary.Write(e.w, binary.BigEndian, float32(input))
        } else {
            if err := e.writeHeader(majorSimpleValue, minorFloat64); err != nil {
                return err
            }
            return binary.Write(e.w, binary.BigEndian, input)
        }
    }

`go test` confirms TestFloat currently works.

The next step is to add support for 16bits floats. As mentioned before Go
doesn’t natively support 16bits floats, so we’ll have to generate the binary
value ourselves. What kind of number can we store in a 16bits float anyway?

A 16bits float looks like this:

    SEEEEEFFFFFFFFFF

S is the sign bit, 0 positive, 1 negative. EEEEE is the 5 bits exponent, and
finally FFFFFFFFFF is the 10 bits fractional part.

The 5 bits exponent’s range is -14 to 15, so as long as the number’s exponents
are with those limits we can encode it as a 16 bits float.

The 10 bits fractional is quite a bit smaller than the 23 bits of the 32 bits
floats’ exponent. We may lose precision when we chop off the end of a number’s
fractional part: all the long bits will be dropped, if there’s any 1’s in those
dropped bits we lose precision. Therefor we will use a [bit mask][] to ensure
we’re not dropping any bits.

[bit mask]: https://en.wikipedia.org/wiki/Mask_(computing)

As a first step we’ll add the function unpackFloat64 that decomposes a float64
into its sign bit, exponent, and fractional. We’ll also add a bunch of constants
that we’ll use for bit mask and shifting operations on floats:

    const (
        float16ExpBits  = 5
        float16FracBits = 10
        float16ExpBias  = 15
        float32ExpBits  = 8
        float32FracBits = 23
        float64ExpBits  = 11
        float64ExpBias  = 1023
        float64FracBits = 52

        // Minimum number of trailing zeros needed in the fractional
        float16MinZeros = float64FracBits - float16FracBits
        float32MinZeros = float64FracBits - float32FracBits

        expMask  = (1 << float64ExpBits) - 1
        fracMask = (1 << float64FracBits) - 1
    )

    func (e *Encoder) writeFloat16(negative bool, exp uint16, frac uint64) error {
        if err := e.writeHeader(majorSimpleValue, minorFloat16); err != nil {
            return err
        }
        var output uint16
        if negative {
            output = 1 << 15
        }
        output |= exp << float16FracBits
        output |= uint16(frac >> (float64FracBits - float16FracBits))
        return binary.Write(e.w, binary.BigEndian, output)
    }

    func unpackFloat64(f float64) (exp int, frac uint64) {
        var r = math.Float64bits(f)
        exp = int(r>>float64FracBits&expMask) - float64ExpBias
        frac = r & fracMask
        return
    }

unpackFloat64 works with 64 bits float since it’s the float type with the most
precision. the r variable is a uint64 with f’s raw binary value. We extract the
exponent by shifting r by float64FracBits and masking is with expMask to trim
off the bit sign. The result is then converted into an integer and we subtract
the exponent’s bias: float64ExpBias which is 1023. That gives us the unbiased
exponent that we can use to determine what type we can use to encode the number.

FIXME rewrite

We’ll use unpackFloat64 to refactor writeFloat using bit masking instead of
converting the number to float32. The exponent range of 32 bits float is -126 to
127 and we need at least float32MinZeros = 23 - 10 = 13 trailing zeros at the end
of the fractional part. Here’s how we implement this:

FIXME rewrite

    func (e *Encoder) writeFloat(input float64) error {
        var (
            exp, frac     = unpackFloat64(input)
            trailingZeros = bits.TrailingZeros64(frac)
        )
        if trailingZeros > float64FracBits {
            trailingZeros = float64FracBits
        }
        switch {
        case (-14 <= exp) && (exp <= 15) && (trailingZeros >= float16MinZeros):
            // FIXME write float16 here
            return ErrNotImplemented
        case float64(float32(input)) == input:
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

Because we haven’t changed the behavior of writeFloat `go test` still works.

Let’s add support for 16 bits floats, and add a case in our tests to verify it
works. We’ll use 1.0 because it’s the easiest 16 bits number to start with, 0.0
is a special IEEE 754 number that we’ll handle later.

We add this to the test case list:

    ...
    {Value: 1.0, Expected: []byte{0xf9, 0x3c, 0x00}},
    ...

To write 16 bits floats we’ll add a new method writeFloat16 that’ll take all
three parameters needed to build a 16 bits float, turn them into a single 16
bits integer, and write this value to the out. writeFloat16 concatenate all the
bits together by using the binary or operator and bit shifting to put the bits
in the right order:

    func (e *Encoder) writeFloat16(negative bool, exp uint16, frac uint64) error {
        if err := e.writeHeader(majorSimpleValue, minorFloat16); err != nil {
            return err
        }
        var output uint16
        if negative {
            output = 1 << 15
        }
        output |= exp << float16FracBits
        output |= uint16(frac >> (float64FracBits - float16FracBits))
        return binary.Write(e.w, binary.BigEndian, output)
    }

We’ll hook up writeFloat16 to writeFloat with a switch case. We check the
exponent’s range and that we’re not dropping any 1’s at the end of the
fractional for float16 and float32, if none match we fall-back to float64:

    func (e *Encoder) writeFloat(input float64) error {
        var (
            exp, frac     = unpackFloat64(input)
            trailingZeros = bits.TrailingZeros64(frac)
        )
        switch {
        case (-14 <= exp) && (exp <= 15) && (trailingZeros >= float16MinZeros):
            return e.writeFloat16(math.Signbit(input), uint16(exp+float16ExpBias), frac)
        case float64(float32(input)) == input:
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

Our encoder handles float16, it looks like we’re done with floats right? We
handle all three types of floating numbers: float16, float32, and float64 it got
to be it.

Actually we aren’t clone to being done with floats: we can minimize the output
even more by handling special numbers: Zero, Infinity, NaN —Not A Number—, and
subnormal numbers. Right now the way the encoder works these special numbers are
all encoded as 64 bits values, but for most of them we can be encoded as 16 bits
numbers.

In a IEEE 754 floating point numbers there are two special exponents binary
values: all 0’s, and all 1’s. All 0’s corresponds to the Zero values, and
subnormal numbers, while all 1’s correspond to inifity and not-a-number values.

Zero is a special floating point value because it cannot be representted
precisely with the formula, when the exponent is zero the fractional part isn’t
prefixed by a 1, but by a 0:

[subnormal numbers]: https://en.wikipedia.org/wiki/Denormal_number

    1.fractional

Since the fractional part of a floating number is never zero, zero can’t be
represented. That’s why there’s a special value for zero: exponent and
fractional both set to 0. Note that this doesn’t include the sign bit: zero can
be either positive or negative. To support this we’ll have to copy the sign bit
from the original number. Let’s add two test to our test suite:

    ...
    {Value: 0.0, Expected: []byte{0xf9, 0x00, 0x00}},
    {Value: math.Copysign(0, -1), Expected: []byte{0xf9, 0x80, 0x00}},
    ...

To get a negative zero in Go we have to use the math.Copysign function since the
compiler turns the expression -0.0 into a positive zero. To add support for zero
values are 16 bits floats we just need to add a if statement at the beginning of
the writeFloat method:

    func (e *Encoder) writeFloat(input float64) error {
        if input == 0 {
            return e.writeFloat16(math.Signbit(input), 0, 0)
        }
        ...
    }

Note that we don’t check if the input equals -0.0 because -0.0 == 0.0.

Now we move onto infinite values. They are simple to detect with the math.IsInf
function. When we find one we write a float where the exponent is all 1’s and
the fractional part is all 0’s:

    func (e *Encoder) writeFloat(input float64) error {
        switch {
        case input == 0:
            return e.writeFloat16(math.Signbit(input), 0, 0)
        case math.IsInf(input, 0):
            return e.writeFloat16(math.Signbit(input), (1<<float16ExpBits)-1, 0)
        }
        ...
    }

Not a number or NaN is similar to infinites but with a non-zero fractional part.
The fractional part of a NaN carries some information, we’ll copy as is and just
chop off the end because all the important information is in the firts few bits.

We add the following the second switch statement:

    func (e *Encoder) writeFloat(input float64) error {
        switch {
        case input == 0:
            return e.writeFloat16(math.Signbit(input), 0, 0)
        case math.IsInf(input, 0):
            return e.writeFloat16(math.Signbit(input), (1<<float16ExpBits)-1, 0)
        }
        var (
            exp, frac     = unpackFloat64(input)
            trailingZeros = bits.TrailingZeros64(frac)
        )
        ...
        switch {
        case math.IsNaN(input):
                return e.writeFloat16(math.Signbit(input), (1<<float16ExpBits)-1, frac)
        ...
        }
    }

The last special numbers we have to handle are called “subnumbers”. Because when
the exponent is all zeros the formula changes, we can also represent some
other numbers than zero with this space. When exp = 00000 with a float16 this
means the the exponent == -14 and the leading bit isn’t a 1 but a 0! This means
we can encode integers with really low exponent from -15 to -24 as long as the
fractional part is a few 1’s, or with -24 a single 1. It turns out that the
smallest possible 16 bits subnumbers is part of 
