package xface

import "math/big"

var xface_prints_big = new(big.Int).SetUint64(xface_prints)

func firstword(w []big.Word) big.Word {
	if len(w) != 0 {
		return w[0]
	} else {
		return 0
	}
}
