package srndtrip

import (
	"encoding/hex"
	"strings"
)

// this package implements unicode tripcode

func MakeSRNdTrip(pubkey string, length int) string {
	var b strings.Builder

	data, err := hex.DecodeString(pubkey)
	if err != nil {
		panic(err)
	}

	if length <= 0 || length > len(data) {
		length = len(data)
	}

	// originally srnd (and srndv2) used 9600==0x2580
	// however, range shifted by 0x10 looks better to me
	// (instead of `▀▁▂▃▄▅▆▇█▉▊▋▌▍▎▏` it'll use `⚀⚁⚂⚃⚄⚅⚆⚇⚈⚉⚊⚋⚌⚍⚎⚏`)
	// and display equaly good both in torbrowser+DejaVuSans and phone
	// since jeff ack'd it, I'll just use it
	const rstart = 0x2590

	// logic (same as in srnd):
	// it first writes length/2 chars of begining
	// and then length/2 chars of ending
	// if length==len(data), that essentially means just using whole
	i := 0
	for ; i < length/2; i++ {
		b.WriteRune(rstart + rune(data[i]))
	}
	for ; i < length; i++ {
		b.WriteRune(rstart + rune(data[len(data)-length+i]))
	}

	return b.String()
}
