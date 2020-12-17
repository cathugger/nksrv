package pcg

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

import (
	"fmt"
	"testing"
	"unsafe"
)

//
// TESTS
//

const count64 = 256

// Does advancing work?
func TestAdvance64(t *testing.T) {
	pcg := NewPCG64()
	pcg.Seed(1, 1, 0, 1)
	values := make([]uint64, count64)
	for i := range values {
		values[i] = pcg.Random()
	}

	for skip := 1; skip < count64; skip++ {
		pcg.Seed(1, 1, 0, 1)
		pcg.Advance(0, uint64(skip))
		result := pcg.Random()
		expect := values[skip]
		if result != expect {
			t.Errorf("Advance(%d) is %d; want %d", skip, result, expect)
		}
	}
}

// Does retreating work?
func TestRetreat64(t *testing.T) {
	pcg := NewPCG64()
	pcg.Seed(1, 1, 0, 1)
	expect := pcg.Random()

	for skip := 1; skip < count64; skip++ {
		pcg.Seed(1, 1, 0, 1)
		for i := 0; i < skip; i++ {
			_ = pcg.Random()
		}
		pcg.Retreat(0, uint64(skip))
		result := pcg.Random()
		if result != expect {
			t.Errorf("Retreat(%d) is %d; want %d", skip, result, expect)
		}
	}
}

//
// BENCHMARKS
//

// Measure the time it takes to generate a 64-bit generator
func BenchmarkNew64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		pcg := NewPCG64()
		pcg.Seed(1, 1, 0, 1)
	}
}

// Measure the time it takes to generate random values
func BenchmarkRandom64(b *testing.B) {
	b.StopTimer()
	pcg := NewPCG64()
	pcg.Seed(1, 1, 0, 1)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		_ = pcg.Random()
	}
}

// Measure the time it takes to generate bounded random values
func BenchmarkSlowBounded64(b *testing.B) {
	b.StopTimer()
	pcg := NewPCG64()
	pcg.Seed(1, 1, 0, 1)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		_ = pcg.slowBounded(uint64(i) & 0xff) // 0..255
		// _ = pcg.SlowBounded(1e18)
		// _ = pcg.SlowBounded(6)             // roll of die
		// _ = pcg.SlowBounded(52)            // deck of cards
		// _ = pcg.SlowBounded(365)           // day of year
	}
}

// Measure the time it takes to generate bounded random values
func BenchmarkFastBounded64(b *testing.B) {
	b.StopTimer()
	pcg := NewPCG64()
	pcg.Seed(1, 1, 0, 1)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		_ = pcg.Bounded(uint64(i) & 0xff) // 0..255
		// _ = pcg.Bounded(1e18)
		// _ = pcg.Bounded(6)             // roll of die
		// _ = pcg.Bounded(52)            // deck of cards
		// _ = pcg.Bounded(365)           // day of year
	}
}

//
// EXAMPLES
//

func ExampleReport64() {
	// Print report
	rng := NewPCG64()
	rng.Seed(0, 42, 0, 54)

	fmt.Printf("pcg64 random:\n"+
		"      -  result:      64-bit unsigned int (uint64)\n"+
		"      -  period:      2^128   (* 2^127 streams)\n"+
		"      -  state type:  PCG64 (%d bytes)\n"+
		"      -  output func: XSL-RR\n"+
		"\n",
		unsafe.Sizeof(rng))

	for round := 1; round <= 5; round++ {
		fmt.Printf("Round %d:\n", round)

		/* Make some 64-bit numbers */
		fmt.Printf("  64bit:")
		for i := 0; i < 6; i++ {
			if i > 0 && i%3 == 0 {
				fmt.Printf("\n\t")
			}
			fmt.Printf(" 0x%016x", rng.Random())
		}
		fmt.Println()

		fmt.Printf("  Again:")
		// rng.Advance(-6 & (1<<64 - 1))
		// rng.Advance(1<<64 - 6)
		rng.Retreat(0, 6)
		for i := 0; i < 6; i++ {
			if i > 0 && i%3 == 0 {
				fmt.Printf("\n\t")
			}
			fmt.Printf(" 0x%016x", rng.Random())
		}
		fmt.Println()

		/* Toss some coins */
		fmt.Printf("  Coins: ")
		for i := 0; i < 65; i++ {
			fmt.Printf("%c", "TH"[rng.slowBounded(2)])
		}
		fmt.Println()

		/* Roll some dice */
		fmt.Printf("  Rolls:")
		for i := 0; i < 33; i++ {
			fmt.Printf(" %d", rng.slowBounded(6)+1)
		}
		fmt.Println()

		/* Deal some cards */
		const (
			SUITS = 4
			CARDS = 52
		)
		var cards [CARDS]int
		for i := range cards {
			cards[i] = i
		}
		for i := uint64(CARDS); i > 1; i-- {
			chosen := rng.slowBounded(i)
			cards[chosen], cards[i-1] = cards[i-1], cards[chosen]
		}

		fmt.Printf("  Cards:")
		for i, c := range cards {
			fmt.Printf(" %c%c", "A23456789TJQK"[c/SUITS], "hcds"[c%SUITS])
			if (i+1)%22 == 0 {
				fmt.Printf("\n\t")
			}
		}
		fmt.Println()

		fmt.Println()
	}

	// Output:
	// pcg64 random:
	//       -  result:      64-bit unsigned int (uint64)
	//       -  period:      2^128   (* 2^127 streams)
	//       -  state type:  PCG64 (32 bytes)
	//       -  output func: XSL-RR
	//
	// Round 1:
	//   64bit: 0x86b1da1d72062b68 0x1304aa46c9853d39 0xa3670e9e0dd50358
	// 	 0xf9090e529a7dae00 0xc85b9fd837996f2c 0x606121f8e3919196
	//   Again: 0x86b1da1d72062b68 0x1304aa46c9853d39 0xa3670e9e0dd50358
	// 	 0xf9090e529a7dae00 0xc85b9fd837996f2c 0x606121f8e3919196
	//   Coins: TTTHHHTTTHHHTTTTHHTTHHTHTHTTHHTHTTTTHHTTTHTHHTHTTTTHHTTTHHHTTTHTT
	//   Rolls: 6 4 1 5 1 5 5 3 6 3 4 6 2 3 6 5 5 5 1 5 3 6 2 6 1 4 4 3 5 2 6 3 2
	//   Cards: 3d 7d 3h Qd 9d 8c Ts Ad 9s 6c Jh Ac 5s 4c 2c 7s Kh Kd 7h Qh 6d Qc
	// 	 8d Qs 6s Js 4d Kc 9h 3c 2h Td 5d 5h 9c 4s 5c 7c 3s 4h As Th 6h Jc
	// 	 2s Jd Tc Ah 2d Ks 8h 8s
	//
	// Round 2:
	//   64bit: 0x1773ba241e7a792a 0xe41aed7117b0bc10 0x36bac8d9432af525
	// 	 0xe0c78e2f3c850a38 0xe3ad939c1c7ce70d 0xa302fdced8c79e93
	//   Again: 0x1773ba241e7a792a 0xe41aed7117b0bc10 0x36bac8d9432af525
	// 	 0xe0c78e2f3c850a38 0xe3ad939c1c7ce70d 0xa302fdced8c79e93
	//   Coins: TTTTHTHTHHTHTHTTTTTHHTTHHHHTHTHHHHHHHTHHHTHHTHTTTHHHHTTHHTTTHTHTH
	//   Rolls: 6 1 1 5 4 1 5 6 3 2 4 2 2 4 6 2 1 5 2 6 2 3 1 5 1 1 5 4 4 2 3 6 3
	//   Cards: As 2h 4d 7d Ad Qc 9s 7h Kh Jc 7c 3d 8c Th 9c Qd 9h Td 6d 8d Qs 5c
	// 	 6s 8s Ac Kd 2d 3h Qh Tc Jh Ah 3s 4h 9d 8h Jd 4s 2s Ts 5s Kc 4c 5d
	// 	 3c 6h 2c 6c 7s Js 5h Ks
	//
	// Round 3:
	//   64bit: 0xc96006593aed3b62 0xf04d5afa3f197bf1 0xce6f729cc913a50f
	// 	 0x98b5fc4fbb1e4aea 0x802dce1b410fc8c3 0xe3bac0a14f6e5033
	//   Again: 0xc96006593aed3b62 0xf04d5afa3f197bf1 0xce6f729cc913a50f
	// 	 0x98b5fc4fbb1e4aea 0x802dce1b410fc8c3 0xe3bac0a14f6e5033
	//   Coins: HTTHTHTTTTTHTTTHHTHTHHTHHHHHHHHHTTTHTHTHTHHTTTTTTHHHHTHTTTTHHHHHH
	//   Rolls: 5 6 4 3 3 1 4 5 2 3 2 1 1 3 2 3 4 5 4 6 4 3 6 2 2 6 3 2 2 4 5 2 5
	//   Cards: 5c 5d 9d 4s Qs Kh 2c 3h Ac 2s 7s 4c 6s 8h 9c 6d 2h 4d 3c 5h 6h Ad
	// 	 7c Js Jd 6c 2d 3d 4h Kd 9s Th Kc 7h 8s Tc Qc Qd Jh Ks 8d Ts Ah Jc
	// 	 5s As Qh 8c 3s Td 7d 9h
	//
	// Round 4:
	//   64bit: 0x68da679de81de48a 0x7ee3c031fa0aa440 0x6eb1663983530403
	// 	 0xfec4d7a9a7aec823 0xbce221c255ee9467 0x460a42a962b8a2f9
	//   Again: 0x68da679de81de48a 0x7ee3c031fa0aa440 0x6eb1663983530403
	// 	 0xfec4d7a9a7aec823 0xbce221c255ee9467 0x460a42a962b8a2f9
	//   Coins: HHHTTTTHHHHHTTTTTTTHHHTHHHHTTHTTTHTTTTHTHHHHTHHTTTHHHTHHTTHHHTHTH
	//   Rolls: 3 5 6 3 6 4 5 6 5 6 1 1 6 6 5 5 5 1 6 4 6 4 5 1 1 4 4 4 3 5 6 1 6
	//   Cards: 7c Kh 2d Qc Jh Js Kc Ks Kd 3d 8d 4s Jc 8c 9d 5c 9c Qh As Qd 3s Ac
	// 	 3h 3c Ad 9h 6h Th Jd 5s 6s 7h 7d 7s 2c 2h 2s 6d 8h 4d Ts Tc 4h 5h
	// 	 4c Ah 9s Td 8s 5d 6c Qs
	//
	// Round 5:
	//   64bit: 0x9e0d084cff42fe2f 0x63cd8347aae338ea 0x112aae00540d3fa1
	// 	 0x53968bc829afd6ec 0x1b9900eb6c5b6d90 0xe89ed17ea33cb420
	//   Again: 0x9e0d084cff42fe2f 0x63cd8347aae338ea 0x112aae00540d3fa1
	// 	 0x53968bc829afd6ec 0x1b9900eb6c5b6d90 0xe89ed17ea33cb420
	//   Coins: HTTTTTHTHTHHHTHTTTHTHHTHHTHTTTHHTTHHHTTTTHTTHHTHHTHHHTTHHTHTHHHHH
	//   Rolls: 6 6 5 1 1 4 5 5 3 1 2 6 5 2 4 6 4 2 6 4 4 3 2 5 3 3 6 5 3 4 5 1 2
	//   Cards: Jd Qh 8s 9h Kh 3c Ts Th Kc Kd 4s Ah 5h 4d Jc 7d 9c Ac 8c Ks 6s 2d
	// 	 Td Qc 2s 8h Tc 6c 3d 3h 4h 6h 7s Qs As 5d 3s 5c 6d 4c Js 5s 8d 9d
	// 	 2c 9s 7h Qd Jh Ad 2h 7c
	//
}
