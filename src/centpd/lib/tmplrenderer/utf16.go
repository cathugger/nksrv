package tmplrenderer

import (
	"bufio"
	"io"
	"net/http"
)

const (
	replacementChar = '\uFFFD'     // Unicode replacement character
	maxRune         = '\U0010FFFF' // Maximum valid Unicode code point.
	byteOrderMark   = '\uFEFF'
)

const (
	// 0xd800-0xdc00 encodes the high 10 bits of a pair.
	// 0xdc00-0xe000 encodes the low 10 bits of a pair.
	// the value is those 20 bits plus 0x10000.
	surr1 = 0xd800
	surr2 = 0xdc00
	surr3 = 0xe000

	surrSelf = 0x10000
)

type utf16lewr struct {
	w *bufio.Writer
}

func (u utf16lewr) writeWord(w uint16) error {
	var err error
	if err = u.w.WriteByte(byte(w)); err != nil {
		return err
	}
	if err = u.w.WriteByte(byte(w >> 8)); err != nil {
		return err
	}
	return nil
}

func (u utf16lewr) Write(b []byte) (int, error) {
	var err error
	for _, v := range string(b) {
		switch {
		case 0 <= v && v < surr1, surr3 <= v && v < surrSelf:
			// normal rune
			w := uint16(v)
			if err = u.writeWord(w); err != nil {
				return 0, err
			}
		case surrSelf <= v && v <= maxRune:
			// needs surrogate sequence
			v -= surrSelf
			w1, w2 := surr1+(v>>10)&0x3ff, surr2+v&0x3ff
			if err = u.writeWord(uint16(w1)); err != nil {
				return 0, err
			}
			if err = u.writeWord(uint16(w2)); err != nil {
				return 0, err
			}
		default:
			w := uint16(replacementChar)
			if err = u.writeWord(w); err != nil {
				return 0, err
			}
		}
	}
	return len(b), nil
}

func (u utf16lewr) Close() error {
	return u.w.Flush()
}

type utf16bewr struct {
	w *bufio.Writer
}

func (u utf16bewr) writeWord(w uint16) error {
	var err error
	if err = u.w.WriteByte(byte(w >> 8)); err != nil {
		return err
	}
	if err = u.w.WriteByte(byte(w)); err != nil {
		return err
	}
	return nil
}

func (u utf16bewr) Write(b []byte) (int, error) {
	var err error
	for _, v := range string(b) {
		switch {
		case 0 <= v && v < surr1, surr3 <= v && v < surrSelf:
			// normal rune
			w := uint16(v)
			if err = u.writeWord(w); err != nil {
				return 0, err
			}
		case surrSelf <= v && v <= maxRune:
			// needs surrogate sequence
			v -= surrSelf
			w1, w2 := surr1+(v>>10)&0x3ff, surr2+v&0x3ff
			if err = u.writeWord(uint16(w1)); err != nil {
				return 0, err
			}
			if err = u.writeWord(uint16(w2)); err != nil {
				return 0, err
			}
		default:
			w := uint16(replacementChar)
			if err = u.writeWord(w); err != nil {
				return 0, err
			}
		}
	}
	return len(b), nil
}

func (u utf16bewr) Close() error {
	return u.w.Flush()
}

func utf16leWCCreator(w http.ResponseWriter) io.WriteCloser {
	ww := utf16lewr{bufio.NewWriter(w)}
	ww.writeWord(uint16(byteOrderMark))
	return ww
}

func utf16leNBOMWCCreator(w http.ResponseWriter) io.WriteCloser {
	ww := utf16lewr{bufio.NewWriter(w)}
	return ww
}

func utf16beWCCreator(w http.ResponseWriter) io.WriteCloser {
	ww := utf16bewr{bufio.NewWriter(w)}
	ww.writeWord(uint16(byteOrderMark))
	return ww
}

func utf16beNBOMWCCreator(w http.ResponseWriter) io.WriteCloser {
	ww := utf16bewr{bufio.NewWriter(w)}
	return ww
}
