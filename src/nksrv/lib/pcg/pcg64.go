package pcg

import "math/bits"

type PCG64 struct {
	state xuint128
	inc   xuint128
}

const (
	maxUint64 = (1 << 64) - 1

	pcg64Multiplier = 47026247687942121848144207491837523525
	pcg64MulHi      = pcg64Multiplier >> 64
	pcg64MulLo      = pcg64Multiplier & maxUint64

	// initial state
	pcg64InitStHi = 0x979c9a98d8462005
	pcg64InitStLo = 0x7d3e9cb6cfe0549b
	// initial increment
	pcg64InitInHi = 0x0000000000000001
	pcg64InitInLo = 0xda3e39cb94b95bdb
)

func NewPCG64() PCG64 {
	return PCG64{
		state: xuint128{pcg64InitStHi, pcg64InitStLo},
		inc:   xuint128{pcg64InitInHi, pcg64InitInLo},
	}
}

func (p *PCG64) Seed(stateHi, stateLo, seqHi, seqLo uint64) {
	//p.state = (state+p.inc)*pcg64Multiplier + p.inc
	//p.inc = (seq << 1) | 1
	p.state = xuint128{stateHi, stateLo}
	p.inc = xuint128{(seqHi << 1) | (seqLo >> 63), (seqLo << 1) | 1}
	p.add()
	p.multiply()
	p.add()
}

func (p *PCG64) Random() uint64 {
	// Advance 64-bit linear congruential generator to new state
	p.multiply()
	p.add()

	// Confuse and permute 64-bit output from old state
	return bits.RotateLeft64(p.state.hi^p.state.lo, -int(p.state.hi>>58))
}

func (p *PCG64) add() {
	p.state.add(p.inc)
}

func (p *PCG64) multiply() {
	p.state.multiply(xuint128{pcg64MulHi, pcg64MulLo})
}

func (p *PCG64) Bounded(bound uint64) uint64 {
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
func (p *PCG64) FastBounded(bound uint64) uint64 {
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

func (p *PCG64) Advance(deltaHi, deltaLo uint64) {
	p.state = advanceLCG128(
		p.state,
		xuint128{deltaHi, deltaLo},
		xuint128{pcg64MulHi, pcg64MulLo},
		p.inc)
}

func (p *PCG64) Retreat(deltaHi, deltaLo uint64) {
	// -x = ~x + 1
	t := xuint128{^deltaHi, ^deltaLo}
	t.add(xuint128{0, 1})
	p.Advance(t.hi, t.lo)
}
