package captcha

import "errors"

func packBCD(out []byte, in []byte) ([]byte, error) {
	il := len(in)
	bl := (il + 1) >> 1

	b := make([]byte, bl)

	ii, bi := il-1, bl-1
	for ii > 0 {
		xh, xl := in[ii-1], in[ii]
		if xh >= 10 || xl >= 10 {
			return nil, errors.New("invalid BCD")
		}

		b[bi] = (xh << 4) | xl

		ii -= 2
		bi--
	}
	if ii >= 0 {
		// last unaligned decimal
		xl := in[ii]
		if xl >= 10 {
			return nil, errors.New("invalid BCD")
		}
		b[bi] = xl
	}

	out = append(out, b...)
	return out, nil
}

func unpackBCD(out []byte, in []byte, xlen int) ([]byte, error) {
	if len(in)*2 < xlen {
		return nil, errors.New("invalid length")
	}

	b := make([]byte, xlen)

	ii := len(in) - 1
	bi := xlen - 1
	for bi > 0 {
		b[bi-1], b[bi] = in[ii]>>4, in[ii]&15

		ii--
		bi -= 2
	}
	if bi >= 0 {
		// last unaligned decimal
		b[bi] = in[ii] & 15
	}

	out = append(out, b...)
	return out, nil
}
