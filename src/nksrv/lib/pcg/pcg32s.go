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

const (
	pcg32sInitState = 0x4d595df4d0f33173 // 5573589319906701683
	pcg32sIncrement = 0x14057b7ef767814f // 1442695040888963407
)

type PCG32s struct {
	state uint64
}

func NewPCG32s() PCG32s {
	return PCG32s{pcg32sInitState}
}

func (p *PCG32s) Seed(state uint64) {
	p.state = (state+pcg32sIncrement)*pcg32Multiplier + pcg32sIncrement
}

func (p *PCG32s) Random() (r uint32) {
	// Confuse and permute 32-bit output from old state
	r = bits.RotateLeft32(uint32(((p.state>>18)^p.state)>>27), -int(p.state>>59))

	// Advance 64-bit linear congruential generator to new state
	p.state = p.state*pcg32Multiplier + pcg32sIncrement

	return
}

func (p *PCG32s) Bounded(bound uint32) uint32 {
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
func (p *PCG32s) FastBounded(bound uint32) uint32 {
	v := p.Random()
	prod := uint64(v) * uint64(bound)
	low := uint32(prod)
	if low < bound {
		thresh := -bound % bound
		for low < thresh {
			v = p.Random()
			prod = uint64(v) * uint64(bound)
			low = uint32(prod)
		}
	}
	return uint32(prod >> 32)
}

func (p *PCG32s) Advance(delta uint64) {
	p.state = advanceLCG64(p.state, delta, pcg32Multiplier, pcg32sIncrement)
}

func (p *PCG32s) Retreat(delta uint64) {
	p.Advance(-delta)
}
