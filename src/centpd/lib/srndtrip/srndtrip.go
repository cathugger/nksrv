package srndtrip

import (
	"encoding/hex"
	"strings"
)

// this package implements unicode tripcode originally used in srnd (and srnd2)

func MakeSRNdTrip(pubkey string, length int) string {
	var b strings.Builder

	data, err := hex.DecodeString(pubkey)
	if err != nil {
		panic(err)
	}

	if length <= 0 || length > len(data) {
		length = len(data)
	}

	// logic:
	// it first writes length/2 chars of begining
	// and then length/2 chars of ending
	// if length==len(data), that essentially means just using whole
	i := 0
	for ; i < length/2; i++ {
		b.WriteRune(9600 + rune(data[i]))
	}
	for ; i < length; i++ {
		b.WriteRune(9600 + rune(data[len(data)-length+i]))
	}

	return b.String()
}
