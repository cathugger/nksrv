package captcha

func packBCD(out []byte, in []byte) ([]byte, error) {
	l := len(in)
	bl := (l+1)>>1
	var b [bl]byte
	i, bi := l - 1, bl - 1
	for ; i > 0; i -= 2, bi-- {
		xh, xl := in[i], in[i-1]
		if xh >= 10 || xl >= 10 {
			return nil, errors.New("invalid BCD")
		}
		b[bi] = (xh << 4) | xl
	}
	if i >= 0 {
		// last unaligned decimal
		xl := in[i]
		if xl >= 10 {
			return nil, errors.New("invalid BCD")
		}
		b[bi] = l
	}
	out = append(out, b)
	return out, nil
}
