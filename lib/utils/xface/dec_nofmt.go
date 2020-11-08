package xface

import (
	"errors"
	"math/big"
)

func getByteAndShift(b *big.Int) (x uint8) {
	bits := b.Bits()
	if len(bits) == 0 {
		return 0
	}
	x = uint8(bits[0])
	b.Rsh(b, 8)
	return
}

func popInteger(b *big.Int, prs []probRange) uint32 {
	/* extract the last byte into r, and shift right b by 8 bits */
	r := getByteAndShift(b)

	// find relevant range
	i := uint32(0)
	for r < prs[i].pOffset || uint32(r) >= uint32(prs[i].pRange)+uint32(prs[i].pOffset) {
		i++
	}
	var x big.Int
	b.Mul(b, x.SetUint64(uint64(prs[i].pRange)))
	b.Add(b, x.SetUint64(uint64(r-prs[i].pOffset)))

	return i
}

func popGreys(b *big.Int, w, h uint32, bitmap []byte) {
	if w > 3 {
		w /= 2
		h /= 2
		popGreys(b, w, h, bitmap[              :])
		popGreys(b, w, h, bitmap[             w:])
		popGreys(b, w, h, bitmap[h*xfaceWidth  :])
		popGreys(b, w, h, bitmap[h*xfaceWidth+w:])
	} else {
		w = popInteger(b, probRanges2x2[:])
		// XXX could we avoid ifs there?
		if w&1 != 0 {
			bitmap[0] = 1
		}
		if w&2 != 0 {
			bitmap[1] = 1
		}
		if w&4 != 0 {
			bitmap[xfaceWidth] = 1
		}
		if w&8 != 0 {
			bitmap[xfaceWidth+1] = 1
		}
	}
}

func decodeBlock(b *big.Int, level, w, h uint32, bitmap []byte) {
	switch popInteger(b, probRangesPerLevel[level][:]) {
	case xfaceColorWhite:
		// do nothing
	case xfaceColorBlack:
		popGreys(b, w, h, bitmap)
	default:
		w /= 2
		h /= 2
		level++
		decodeBlock(b, level, w, h, bitmap[              :])
		decodeBlock(b, level, w, h, bitmap[             w:])
		decodeBlock(b, level, w, h, bitmap[h*xfaceWidth  :])
		decodeBlock(b, level, w, h, bitmap[h*xfaceWidth+w:])
	}
}

func xfaceRead(in string) (b *big.Int) {
	var x big.Int
	for i := 0; i < len(in); i++ {
		c := in[i]
		if c < xfaceFirstPrint || c > xfaceLastPrint {
			// invalid digit
			if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
				// allow common whitespace chars
				continue
			}
			// but not anything else
			return nil
		}
		if b != nil {
			b.Mul(b, xfacePrintsBig)
			b.Add(b, x.SetUint64(uint64(c - xfaceFirstPrint)))
		} else {
			b = new(big.Int).SetUint64(uint64(c - xfaceFirstPrint))
		}
	}
	return
}

func xfaceDecode(bitmap *[xfacePixels]byte, b *big.Int) {

	decb := func(buf []byte) { decodeBlock(b, 0, 16, 16, buf) }

	decb(bitmap[  :])
	decb(bitmap[16:])
	decb(bitmap[32:])

	decb(bitmap[xfaceWidth*16   :])
	decb(bitmap[xfaceWidth*16+16:])
	decb(bitmap[xfaceWidth*16+32:])

	decb(bitmap[xfaceWidth*32   :])
	decb(bitmap[xfaceWidth*32+16:])
	decb(bitmap[xfaceWidth*32+32:])

	generateFace(bitmap[:], bitmap[:])
}

func xfaceDecodeString(
	bitmap *[xfacePixels]byte, in string) (err error) {

	b := xfaceRead(in)
	if b == nil {
		err = errors.New("i have no face")
		return
	}

	// do decoding
	xfaceDecode(bitmap, b)
	return
}
