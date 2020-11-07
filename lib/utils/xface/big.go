package xface

import "math/big"

var xfacePrintsBig = new(big.Int).SetUint64(xfacePrints)

func firstword(w []big.Word) big.Word {
	if len(w) != 0 {
		return w[0]
	} else {
		return 0
	}
}
