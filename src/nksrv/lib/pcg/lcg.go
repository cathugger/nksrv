package pcg

func advanceLCG64(state, delta, curMult, curPlus uint64) uint64 {
	accMult := uint64(1)
	accPlus := uint64(0)
	for delta != 0 {
		if delta&1 != 0 {
			accMult *= curMult
			accPlus = accPlus*curMult + curPlus
		}
		curPlus *= curMult + 1
		curMult *= curMult
		delta /= 2
	}
	return accMult*state + accPlus
}

func advanceLCG128(state, delta, curMult, curPlus xuint128) xuint128 {
	zero := xuint128{0, 0}
	accMult := xuint128{0, 1}
	accPlus := xuint128{0, 0}
	for delta != zero {
		if delta.lo&1 != 0 {
			//accMult *= curMult
			accMult.multiply(curMult)
			//accPlus = accPlus*curMult + curPlus
			accPlus.multiply(curMult)
			accPlus.add(curPlus)
		}
		//curPlus *= curMult + 1
		t := curMult
		t.add(xuint128{0, 1})
		curPlus.multiply(t)
		//curMult *= curMult
		curMult.multiply(curMult)
		//delta /= 2
		delta = xuint128{delta.hi >> 1, (delta.hi << 63) | (delta.lo >> 1)}
	}
	//return accMult*state + accPlus
	accMult.multiply(state)
	accMult.add(accPlus)
	return accMult
}
