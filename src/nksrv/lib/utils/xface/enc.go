package xface

import (
	"bytes"
	"math/big"
)

func allSame(w, h uint32, bitmap []byte) bool {
	var x uint32
	val := bitmap[0]
	for h != 0 {
		h--
		for x = 0; x < w; x++ {
			if bitmap[x] != val {
				return false
			}
		}
		bitmap = bitmap[xfaceWidth:]
	}
	return true
}

func allBlack(w, h uint32, bitmap []byte) bool {
	if w > 3 {
		w /= 2
		h /= 2
		return true &&
			allBlack(w, h, bitmap[                 :]) &&
			allBlack(w, h, bitmap[                w:]) &&
			allBlack(w, h, bitmap[h*xfaceWidth    :]) &&
			allBlack(w, h, bitmap[h*xfaceWidth + w:])
	} else {
		/* at least one pixel in the 2x2 grid is non-zero */
		return false ||
			bitmap[            0] != 0 ||
			bitmap[            1] != 0 ||
			bitmap[xfaceWidth+0] != 0 ||
			bitmap[xfaceWidth+1] != 0
	}
}

func allWhite(w, h uint32, bitmap []byte) bool {
	return bitmap[0] == 0 && allSame(w, h, bitmap)
}

func pushGreys(pq []probRange, w, h uint32, bitmap []byte) []probRange {
	if w > 3 {
		w /= 2
		h /= 2
		pq = pushGreys(pq, w, h, bitmap[                 :])
		pq = pushGreys(pq, w, h, bitmap[                w:])
		pq = pushGreys(pq, w, h, bitmap[h*xfaceWidth    :])
		pq = pushGreys(pq, w, h, bitmap[h*xfaceWidth + w:])
	} else {
		p := probRanges2x2[
			1*bitmap[            0]+
			2*bitmap[            1]+
			4*bitmap[xfaceWidth+0]+
			8*bitmap[xfaceWidth+1]]
		pq = append(pq, p)
	}
	return pq
}

func encodeBlock(
	pq []probRange, level uint32, w, h uint32, bitmap []byte) []probRange {

	if allWhite(w, h, bitmap) {
		pq = append(pq, probRangesPerLevel[level][xfaceColorWhite])
	} else if allBlack(w, h, bitmap) {
		pq = append(pq, probRangesPerLevel[level][xfaceColorBlack])
		pq = pushGreys(pq, w, h, bitmap)
	} else {
		pq = append(pq, probRangesPerLevel[level][xfaceColorGrey])
		w /= 2
		h /= 2
		level++
		pq = encodeBlock(pq, level, w, h, bitmap[                :])
		pq = encodeBlock(pq, level, w, h, bitmap[               w:])
		pq = encodeBlock(pq, level, w, h, bitmap[h*xfaceWidth    :])
		pq = encodeBlock(pq, level, w, h, bitmap[h*xfaceWidth + w:])
	}
	return pq
}

func pushInteger(b *big.Int, prange probRange) {
	var m, x big.Int

	b.QuoRem(b, x.SetUint64(uint64(prange.pRange)), &m)
	b.Lsh(b, 8)
	b.Add(b, x.SetUint64(uint64(firstword(m.Bits())+big.Word(prange.pOffset))))
}

func xfaceEncode(b *big.Int, bitmap []byte) {
	var srcbmp [xfacePixels]byte
	copy(srcbmp[:], bitmap)
	generateFace(bitmap, srcbmp[:])

	var pq []probRange

	encb := func(buf []byte) { pq = encodeBlock(pq, 0, 16, 16, buf) }

	encb(bitmap[  :])
	encb(bitmap[16:])
	encb(bitmap[32:])

	encb(bitmap[xfaceWidth*16   :])
	encb(bitmap[xfaceWidth*16+16:])
	encb(bitmap[xfaceWidth*16+32:])

	encb(bitmap[xfaceWidth*32   :])
	encb(bitmap[xfaceWidth*32+16:])
	encb(bitmap[xfaceWidth*32+32:])

	for i := len(pq) - 1; i >= 0; i-- {
		pushInteger(b, pq[i])
	}
}

func xfaceWrite(b *big.Int) string {
	var buf bytes.Buffer
	/* write the inverted big integer in b to intbuf */
	var m big.Int
	for {
		// at least one character
		b.QuoRem(b, xfacePrintsBig, &m)
		c := byte(firstword(m.Bits()) + xfaceFirstPrint)
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

func xfaceEncodeString(bitmap []byte) string {
	var b big.Int
	xfaceEncode(&b, bitmap)
	return xfaceWrite(&b)
}
