package xface

import (
	"errors"
	"image/color"
	"math/big"
)

func getbyteandshift(b *big.Int) (x uint8) {
	bits := b.Bits()
	if len(bits) == 0 {
		return 0
	}
	x = uint8(bits[0])
	b.Rsh(b, 8)
	return
}

func pop_integer(b *big.Int, prs []probRange) uint32 {
	/* extract the last byte into r, and shift right b by 8 bits */
	r := getbyteandshift(b)

	// find relevant range
	i := uint32(0)
	for r < prs[i].p_offset ||
		uint32(r) >= uint32(prs[i].p_range)+uint32(prs[i].p_offset) {

		i++
	}
	var x big.Int
	b.Mul(b, x.SetUint64(uint64(prs[i].p_range)))
	b.Add(b, x.SetUint64(uint64(r-prs[i].p_offset)))

	return i
}

func pop_greys(b *big.Int, w, h uint32, bitmap []byte) {
	if w > 3 {
		w /= 2
		h /= 2
		pop_greys(b, w, h, bitmap[               :])
		pop_greys(b, w, h, bitmap[              w:])
		pop_greys(b, w, h, bitmap[h*xface_width  :])
		pop_greys(b, w, h, bitmap[h*xface_width+w:])
	} else {
		w = pop_integer(b, xface_probranges_2x2[:])
		// XXX could we avoid ifs there?
		if w&1 != 0 {
			bitmap[0] = 1
		}
		if w&2 != 0 {
			bitmap[1] = 1
		}
		if w&4 != 0 {
			bitmap[xface_width] = 1
		}
		if w&8 != 0 {
			bitmap[xface_width+1] = 1
		}
	}
}

func decode_block(b *big.Int, level, w, h uint32, bitmap []byte) {
	switch pop_integer(b, xface_probranges_per_level[level][:]) {
	case xface_color_white:
		// do nothing
	case xface_color_black:
		pop_greys(b, w, h, bitmap)
	default:
		w /= 2
		h /= 2
		level++
		decode_block(b, level, w, h, bitmap[               :])
		decode_block(b, level, w, h, bitmap[              w:])
		decode_block(b, level, w, h, bitmap[h*xface_width  :])
		decode_block(b, level, w, h, bitmap[h*xface_width+w:])
	}
}

func xface_read(in string) (b *big.Int) {
	var x big.Int
	for i := 0; i < len(in); i++ {
		c := in[i]
		if c < xface_first_print || c > xface_last_print {
			// invalid digit
			if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
				// allow common whitespace chars
				continue
			}
			// but not anything else
			return nil
		}
		if b != nil {
			b.Mul(b, xface_prints_big)
			b.Add(b, x.SetUint64(uint64(c - xface_first_print)))
		} else {
			b = new(big.Int).SetUint64(uint64(c - xface_first_print))
		}
	}
	return
}

var palWB = [2]color.Color{
	color.RGBA{0xff, 0xff, 0xff, 0xff},
	color.RGBA{0x00, 0x00, 0x00, 0xff},
}

func xface_decode(bitmap *[xface_pixels]byte, b *big.Int) {
	decb := func(buf []byte) { decode_block(b, 0, 16, 16, buf) }

	decb(bitmap[  :])
	decb(bitmap[16:])
	decb(bitmap[32:])

	decb(bitmap[xface_width*16   :])
	decb(bitmap[xface_width*16+16:])
	decb(bitmap[xface_width*16+32:])

	decb(bitmap[xface_width*32   :])
	decb(bitmap[xface_width*32+16:])
	decb(bitmap[xface_width*32+32:])

	xface_generate_face(bitmap, bitmap)
}

func xface_decode_string(
	bitmap *[xface_pixels]byte, in string) (err error) {

	b := xface_read(in)
	if b == nil {
		err = errors.New("I have no face")
		return
	}

	// do decoding
	xface_decode(bitmap, b)
	return
}
