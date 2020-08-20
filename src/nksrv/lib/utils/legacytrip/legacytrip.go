package legacytrip

// legacy UNIX DES crypt(3) & Shift_JIS based imageboard tripcode stuff

import (
	ud "nksrv/lib/utils/unixdescrypt"

	ej "golang.org/x/text/encoding/japanese"
	tr "golang.org/x/text/transform"
)

func encodeSJIS(src, dst []byte) ([]byte, error) {
	enc := ej.ShiftJIS.NewEncoder()
	var ndst int
	for {
		var err error
		ndst, _, err = enc.Transform(dst, src, true)
		if err != nil {
			if err == tr.ErrShortDst {
				dst = make([]byte, 2*len(dst))
				enc.Reset()
				continue
			}
			return nil, err
		}
		return dst[:ndst], nil
	}
}

// assumption: len(old) == len(rep)
func myTr(buf []byte, old, rep string) {
	for i, ic := range buf {
		for j := 0; j < len(old); j++ {
			if ic == old[j] {
				buf[i] = rep[j]
				break
			}
		}
	}
}

func replaceNotInRangeWith(buf []byte, rst, red, x byte) {
	for i, c := range buf {
		if c < rst || c > red {
			buf[i] = x
		}
	}
}

var saltsuffix = []byte("H..")

//var saltregexp = regexp.MustCompile("[^.-z]")
//var saltreplacement = []byte{'.'}

func MakeLegacyTrip(src string) (string, error) {
	// encode in Shift_JIS
	var ss [16]byte
	trip, err := encodeSJIS([]byte(src), ss[:])
	if err != nil {
		return "", err
	}
	// """salt""" preparation
	var salt [2]byte
	copy(salt[:], append(trip, saltsuffix...)[1:3])
	replaceNotInRangeWith(salt[:], '.', 'z', '.')
	myTr(salt[:], ":;<=>?@[\\]^_`", `ABCDEFGabcdef`)
	// do crypt
	var buf [13]byte
	res := ud.CryptDES(trip, salt, buf[:0])
	// take last 10 bytes from that
	// this results in 64^10 variations which tbh isn't much
	return string(res[len(res)-10:]), nil
}
