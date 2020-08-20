package pcg

import "math/bits"

type xuint128 struct {
	hi uint64
	lo uint64
}

func (x *xuint128) add(y xuint128) {
	var carry uint64
	x.lo, carry = bits.Add64(x.lo, y.lo, 0)
	x.hi, _ = bits.Add64(x.hi, y.hi, carry)
}

func (x *xuint128) multiply(y xuint128) {
	hi, lo := bits.Mul64(x.lo, y.lo)
	hi += x.hi * y.lo
	hi += x.lo * y.hi
	x.hi = hi
	x.lo = lo
}
