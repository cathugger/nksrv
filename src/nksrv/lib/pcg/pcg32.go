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
	pcg32InitState     = 0x853c49e6748fea9b //  9600629759793949339
	pcg32InitIncrement = 0xda3e39cb94b95bdb // 15726070495360670683
	pcg32Multiplier    = 0x5851f42d4c957f2d //  6364136223846793005
)

type PCG32 struct {
	state uint64
	inc   uint64
}

func NewPCG32() PCG32 {
	return PCG32{pcg32InitState, pcg32InitIncrement}
}

func (p *PCG32) Seed(state, seq uint64) {
	p.inc = (seq << 1) | 1
	p.state = (state+p.inc)*pcg32Multiplier + p.inc
}

func (p *PCG32) Random() (r uint32) {
	// Confuse and permute 32-bit output from old state
	r = bits.RotateLeft32(uint32(((p.state>>18)^p.state)>>27), -int(p.state>>59))

	// Advance 64-bit linear congruential generator to new state
	p.state = p.state*pcg32Multiplier + p.inc

	return
}

func (p *PCG32) slowBounded(bound uint32) uint32 {
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

func (p *PCG32) Bounded(bound uint32) uint32 {
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

func (p *PCG32) Advance(delta uint64) {
	p.state = advanceLCG64(p.state, delta, pcg32Multiplier, p.inc)
}

func (p *PCG32) Retreat(delta uint64) {
	p.Advance(-delta)
}
