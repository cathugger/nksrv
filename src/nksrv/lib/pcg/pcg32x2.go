package pcg

import "math/bits"

// PCG Random Number Generation
// Developed by Melissa O'Neill <oneill@pcg-random.org>
// Paper and details at http://www.pcg-random.org
// Ported to Go by Michael Jones <michael.jones@gmail.com>

// Copyright 2018 Michael T. Jones
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for
// the specific language governing permissions and limitations under the License.

type PCG32x2 struct {
	hi PCG32
	lo PCG32
}

func NewPCG32x2() PCG32x2 {
	return PCG32x2{NewPCG32(), NewPCG32()}
}

func (p *PCG32x2) Seed(stateHi, stateLo, sequenceHi, sequenceLo uint64) *PCG32x2 {
	mask := ^uint64(0) >> 1
	if sequenceHi&mask == sequenceLo&mask {
		sequenceLo = ^sequenceHi
	}
	p.hi.Seed(stateHi, sequenceHi)
	p.lo.Seed(stateLo, sequenceLo)
	return p
}

func (p *PCG32x2) Random() uint64 {
	return uint64(p.hi.Random())<<32 | uint64(p.lo.Random())
}

func (p *PCG32x2) Bounded(bound uint64) uint64 {
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
func (p *PCG32x2) FastBounded(bound uint64) uint64 {
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

func (p *PCG32x2) Advance(delta uint64) *PCG32x2 {
	p.lo.Advance(delta)
	p.hi.Advance(delta)
	return p
}

func (p *PCG32x2) Retreat(delta uint64) *PCG32x2 {
	return p.Advance(-delta)
}