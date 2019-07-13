package xface

import (
	"bytes"
	"math/big"
)

func all_same(w, h uint32, bitmap []byte) bool {
	var x uint32

	val := bitmap[0]
	for h != 0 {
		h--
		for x = 0; x < w; x++ {
			if bitmap[x] != val {
				return false
			}
		}
		bitmap = bitmap[xface_width:]
	}
	return true
}

func all_black(w, h uint32, bitmap []byte) bool {
	if w > 3 {
		w /= 2
		h /= 2
		return true &&
			all_black(w, h, bitmap[               :]) &&
			all_black(w, h, bitmap[              w:]) &&
			all_black(w, h, bitmap[h*xface_width  :]) &&
			all_black(w, h, bitmap[h*xface_width+w:])
	} else {
		/* at least one pixel in the 2x2 grid is non-zero */
		return false ||
			bitmap[            0] != 0 ||
			bitmap[            1] != 0 ||
			bitmap[xface_width+0] != 0 ||
			bitmap[xface_width+1] != 0
	}
}

func all_white(w, h uint32, bitmap []byte) bool {
	return bitmap[0] == 0 && all_same(w, h, bitmap)
}

func push_greys(pq []probRange, w, h uint32, bitmap []byte) []probRange {
	if w > 3 {
		w /= 2
		h /= 2
		pq = push_greys(pq, w, h, bitmap[               :])
		pq = push_greys(pq, w, h, bitmap[              w:])
		pq = push_greys(pq, w, h, bitmap[h*xface_width  :])
		pq = push_greys(pq, w, h, bitmap[h*xface_width+w:])
	} else {
		p := xface_probranges_2x2[
			1*bitmap[            0]+
			2*bitmap[            1]+
			4*bitmap[xface_width+0]+
			8*bitmap[xface_width+1]]
		pq = append(pq, p)
	}
	return pq
}

func encode_block(
	pq []probRange, level uint32, w, h uint32, bitmap []byte) []probRange {

	if all_white(w, h, bitmap) {
		pq = append(pq, xface_probranges_per_level[level][xface_color_white])
	} else if all_black(w, h, bitmap) {
		pq = append(pq, xface_probranges_per_level[level][xface_color_black])
		pq = push_greys(pq, w, h, bitmap)
	} else {
		pq = append(pq, xface_probranges_per_level[level][xface_color_grey])
		w /= 2
		h /= 2
		level++
		pq = encode_block(pq, level, w, h, bitmap[               :])
		pq = encode_block(pq, level, w, h, bitmap[              w:])
		pq = encode_block(pq, level, w, h, bitmap[h*xface_width  :])
		pq = encode_block(pq, level, w, h, bitmap[h*xface_width+w:])
	}
	return pq
}

func push_integer(b *big.Int, prange probRange) {
	var m, x big.Int

	b.QuoRem(b, x.SetUint64(uint64(prange.p_range)), &m)
	b.Lsh(b, 8)
	b.Add(b, x.SetUint64(uint64(firstword(m.Bits())+big.Word(prange.p_offset))))
}

func xface_encode(b *big.Int, bitmap []byte) {
	var srcbmp [xface_pixels]byte
	copy(srcbmp[:], bitmap)
	xface_generate_face(bitmap, srcbmp[:])

	var pq []probRange

	encb := func(buf []byte) { pq = encode_block(pq, 0, 16, 16, buf) }

	encb(bitmap[  :])
	encb(bitmap[16:])
	encb(bitmap[32:])

	encb(bitmap[xface_width*16   :])
	encb(bitmap[xface_width*16+16:])
	encb(bitmap[xface_width*16+32:])

	encb(bitmap[xface_width*32   :])
	encb(bitmap[xface_width*32+16:])
	encb(bitmap[xface_width*32+32:])

	for i := len(pq) - 1; i >= 0; i-- {
		push_integer(b, pq[i])
	}
}

func xface_write(b *big.Int) string {
	var buf bytes.Buffer
	/* write the inverted big integer in b to intbuf */
	var m big.Int
	for {
		// at least one character
		b.QuoRem(b, xface_prints_big, &m)
		c := byte(firstword(m.Bits()) + xface_first_print)
		buf.WriteByte(c)
		// no more?
		if len(b.Bits()) == 0 {
			break
		}
	}
	bb := buf.Bytes()
	// invert
	for i := 0; i < len(bb)/2; i++ {
		bb[i], bb[len(bb)-i-1] = bb[len(bb)-i-1], bb[i]
	}
	return string(bb)
}

func xface_encode_string(bitmap []byte) string {
	var b big.Int
	xface_encode(&b, bitmap)
	return xface_write(&b)
}
