package pcg

import "math/bits"

type PCG64s struct {
	state xuint128
}

const (
	pcg64sIncrement = 117397592171526113268558934119004209487
	pcg64sIncHi     = pcg64sIncrement >> 64
	pcg64sIncLo     = pcg64sIncrement & maxUint64

	pcg64sInitializer = 245720598905631564143578724636268694099
	pcg64sInitHi      = pcg64sInitializer >> 64
	pcg64sInitLo      = pcg64sInitializer & maxUint64
)

func NewPCG64s() PCG64s {
	return PCG64s{
		state: xuint128{pcg64sInitHi, pcg64sInitLo},
	}
}

func (p *PCG64s) Seed(stateHi, stateLo uint64) {
	//p.state = (state+pcg64sIncrement)*pcg64Multiplier + pcg64sIncrement
	p.state = xuint128{stateHi, stateLo}
	p.add()
	p.multiply()
	p.add()
}

func (p *PCG64s) Random() uint64 {
	// Advance 64-bit linear congruential generator to new state
	p.multiply()
	p.add()

	// Confuse and permute 64-bit output from old state
	return bits.RotateLeft64(p.state.hi^p.state.lo, -int(p.state.hi>>58))
}

func (p *PCG64s) add() {
	p.state.add(xuint128{pcg64sIncHi, pcg64sIncLo})
}

func (p *PCG64s) multiply() {
	p.state.multiply(xuint128{pcg64MulHi, pcg64MulLo})
}

func (p *PCG64s) Bounded(bound uint64) uint64 {
	if bound == 0 {
		return 0
	}
	threshold := -bound % bound
	for {
		r := p.Random()
		if r >= threshold {
			return r % bound
		}
	}
}

// as in int31n, go/src/math/rand/rand.go
// this function uses a single division in the worst case
func (p *PCG64s) FastBounded(bound uint64) uint64 {
	v := p.Random()
	high, low := bits.Mul64(v, bound)
	if low < bound {
		thresh := -bound
		if thresh >= bound {
			thresh -= bound
			if thresh >= bound {
				thresh %= bound
			}
		}
		for low < thresh {
			v = p.Random()
			high, low = bits.Mul64(v, bound)
		}
	}
	return high
}

func (p *PCG64s) Advance(deltaHi, deltaLo uint64) {
	p.state = advanceLCG128(
		p.state,
		xuint128{deltaHi, deltaLo},
		xuint128{pcg64MulHi, pcg64MulLo},
		xuint128{pcg64sIncHi, pcg64sIncLo})
}

func (p *PCG64s) Retreat(deltaHi, deltaLo uint64) {
	// -x = ~x + 1
	t := xuint128{^deltaHi, ^deltaLo}
	t.add(xuint128{0, 1})
	p.Advance(t.hi, t.lo)
}
