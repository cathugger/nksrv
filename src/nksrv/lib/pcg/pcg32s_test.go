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

const count32s = 256

// Does advancing work?
func TestAdvance32s(t *testing.T) {
	pcg := NewPCG32s()
	pcg.Seed(1)
	values := make([]uint32, count32s)
	for i := range values {
		values[i] = pcg.Random()
	}

	for skip := 1; skip < count32s; skip++ {
		pcg.Seed(1)
		pcg.Advance(uint64(skip))
		result := pcg.Random()
		expect := values[skip]
		if result != expect {
			t.Errorf("Advance(%d) is %d; want %d", skip, result, expect)
		}
	}
}

// Does retreating work?
func TestRetreat32s(t *testing.T) {
	pcg := NewPCG32s()
	pcg.Seed(1)
	expect := pcg.Random()

	for skip := 1; skip < count32s; skip++ {
		pcg.Seed(1)
		for i := 0; i < skip; i++ {
			_ = pcg.Random()
		}
		pcg.Retreat(uint64(skip))
		result := pcg.Random()
		if result != expect {
			t.Errorf("Retreat(%d) is %d; want %d", skip, result, expect)
		}
	}
}

//
// BENCHMARKS
//

// Measure the time it takes to generate a 32-bit generator
func BenchmarkNew32s(b *testing.B) {
	for i := 0; i < b.N; i++ {
		pcg := NewPCG32s()
		pcg.Seed(1)
	}
}

// Measure the time it takes to generate random values
func BenchmarkRandom32s(b *testing.B) {
	b.StopTimer()
	pcg := NewPCG32s()
	pcg.Seed(1)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		_ = pcg.Random()
	}
}

// Measure the time it takes to generate bounded random values
func BenchmarkBounded32s(b *testing.B) {
	b.StopTimer()
	pcg := NewPCG32s()
	pcg.Seed(1)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		_ = pcg.Bounded(uint32(i) & 0xff) // 0..255
		//_ = pcg.Bounded(1e9)
		// _ = pcg.Bounded(6)             // roll of die
		// _ = pcg.Bounded(52)            // deck of cards
		// _ = pcg.Bounded(365)           // day of year
	}
}

// Measure the time it takes to generate bounded random values
func BenchmarkBounded32sFast(b *testing.B) {
	b.StopTimer()
	pcg := NewPCG32s()
	pcg.Seed(1)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		_ = pcg.FastBounded(uint32(i) & 0xff) // 0..255
		//_ = pcg.FastBounded(1e9)
		// _ = pcg.Bounded(6)                 // roll of die
		// _ = pcg.Bounded(52)                // deck of cards
		// _ = pcg.Bounded(365)               // day of year
	}
}

//
// EXAMPLES
//

func ExampleReport32s() {
	// Print report
	rng := NewPCG32s()
	rng.Seed(42)

	fmt.Printf("pcg32s random:\n"+
		"      -  result:      32-bit unsigned int (uint32)\n"+
		"      -  period:      2^64\n"+
		"      -  state type:  PCG32s (%d bytes)\n"+
		"      -  output func: XSH-RR\n"+
		"\n",
		unsafe.Sizeof(rng))

	for round := 1; round <= 5; round++ {
		fmt.Printf("Round %d:\n", round)

		/* Make some 32-bit numbers */
		fmt.Printf("  32bit:")
		for i := 0; i < 6; i++ {
			fmt.Printf(" 0x%08x", rng.Random())
		}
		fmt.Println()

		fmt.Printf("  Again:")
		// rng.Advance(-6 & (1<<64 - 1))
		// rng.Advance(1<<64 - 6)
		rng.Retreat(6)
		for i := 0; i < 6; i++ {
			fmt.Printf(" 0x%08x", rng.Random())
		}
		fmt.Println()

		/* Toss some coins */
		fmt.Printf("  Coins: ")
		for i := 0; i < 65; i++ {
			fmt.Printf("%c", "TH"[rng.Bounded(2)])
		}
		fmt.Println()

		/* Roll some dice */
		fmt.Printf("  Rolls:")
		for i := 0; i < 33; i++ {
			fmt.Printf(" %d", rng.Bounded(6)+1)
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
		for i := uint32(CARDS); i > 1; i-- {
			chosen := rng.Bounded(i)
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
	// pcg32s random:
	//       -  result:      32-bit unsigned int (uint32)
	//       -  period:      2^64
	//       -  state type:  PCG32s (8 bytes)
	//       -  output func: XSH-RR
	//
	// Round 1:
	//   32bit: 0xc2f57bd6 0x6b07c4a9 0x72b7b29b 0x44215383 0xf5af5ead 0x68beb632
	//   Again: 0xc2f57bd6 0x6b07c4a9 0x72b7b29b 0x44215383 0xf5af5ead 0x68beb632
	//   Coins: THTHHHTTHHTTHTTHTHHHTHTTTHTTHTTHTTTHHTTTTTHHTTTHTTHTHHTHHHTTHTTTH
	//   Rolls: 4 1 3 3 6 6 5 1 3 4 4 3 2 2 5 4 1 3 3 3 1 4 6 4 6 6 1 6 1 2 3 6 6
	//   Cards: 2d 5c 3h 6d Js 9c 4h Ts Qs 5d Ks 5h Ad Ac Qh Th Jd Kc Tc 7s Ah Kd
	// 	 7h 3c 4d 8s 2c 3d Kh 8h Jc 6h 4c 8d Qc 7c Td 2s 3s 4s 7d Qd Jh As
	// 	 6c 8c 5s 2h 6s 9d 9s 9h
	//
	// Round 2:
	//   32bit: 0x0573afcc 0x2cab16db 0x6af6f55a 0xe916bec2 0x1ca9b4a4 0xbb2778eb
	//   Again: 0x0573afcc 0x2cab16db 0x6af6f55a 0xe916bec2 0x1ca9b4a4 0xbb2778eb
	//   Coins: THHHTHTTTHHHTTTTTTHTTHTHTHHHTHHHTHTHTTHTTTTTHTHHTHHTTHHHHHTTTHTTH
	//   Rolls: 1 5 3 3 5 1 5 6 5 6 6 3 5 5 6 6 2 6 4 1 5 6 3 6 5 5 1 3 2 4 5 1 1
	//   Cards: 9c Ad 5d 7d Ah 8c Th Kd 5c Js 7c Kc Kh 6c Ks Tc Td 3d 7h 2d 5s 9s
	// 	 3h As 9d 8h 4s 6h Ts 2c Jh 3c 8s 4h 5h 6s Jd 8d 3s 6d 7s 4d Ac Qc
	// 	 4c 2h Qh 9h Qd 2s Qs Jc
	//
	// Round 3:
	//   32bit: 0x114306f3 0xb9bf0d91 0x1aed8e5e 0x587de8b7 0x7477c8bd 0xd853ec9d
	//   Again: 0x114306f3 0xb9bf0d91 0x1aed8e5e 0x587de8b7 0x7477c8bd 0xd853ec9d
	//   Coins: HTHHTHHHHTHTHTTHTHTHHTHTTHHHTTTTHHTTTTTTHTHTTTHTHTTTHTHHHHTTTTTTT
	//   Rolls: 1 5 4 2 1 4 6 3 2 1 6 3 6 4 3 1 4 4 2 5 5 3 3 2 6 1 6 3 2 6 5 6 3
	//   Cards: Ah 8d Ad Jd 2d 3h Jh 7c Kc Ks 3d As 4s 3s 8h Qc 7d Td 6c 8c 4d 5c
	// 	 9d Qh Js Ac Kd 5s 6d Ts 9h 9s 9c 2c 5h 3c 5d Th 4c 6s 7s Qd 7h 2h
	// 	 Tc 6h 4h 8s Qs Jc Kh 2s
	//
	// Round 4:
	//   32bit: 0xb982cd46 0x01cc6f94 0x0ad658ae 0xf6c6c97e 0xd1b772dd 0x0098599e
	//   Again: 0xb982cd46 0x01cc6f94 0x0ad658ae 0xf6c6c97e 0xd1b772dd 0x0098599e
	//   Coins: HTTHTTHHHHTHTHHHTTHTHTHTTTHTHTHHTHTHTTTTHHTTHHHTHTTHHTTTHHHTTHHHH
	//   Rolls: 4 4 5 4 2 1 4 2 2 5 2 5 6 6 2 1 6 6 2 6 6 3 6 2 1 4 1 1 1 1 5 1 5
	//   Cards: 6s Td 3h Js 7h Jh Ac Kh Th 4h 3c 6d Qs Ah 8h Kc Tc 2h 8c 2c Jd 2s
	// 	 Qh 4d 3d Ks 7s 9d 5d 2d 5s 5h Jc 3s 9s Qd Qc 7d 6h As 8s 4s 4c 8d
	// 	 9c 6c 5c Ad 7c 9h Kd Ts
	//
	// Round 5:
	//   32bit: 0xef3c7322 0xa1ff2188 0x3f564b42 0x91c90425 0x17711b95 0xf43aa1f7
	//   Again: 0xef3c7322 0xa1ff2188 0x3f564b42 0x91c90425 0x17711b95 0xf43aa1f7
	//   Coins: HTTHHHTTHTTTHTHHTHTHTHHTHHTTTHTTHTHHTHTTTTTHTHTTHHHHTHTHTHHTHHTHT
	//   Rolls: 4 1 6 3 3 2 5 6 3 2 6 5 3 1 5 5 4 6 4 4 2 5 5 4 1 5 2 4 5 5 5 3 5
	//   Cards: 6c 8d 4d Jc 9d As 9s 3c 9c Th Ks Qs 4c Js Ah Qc Ac Kd Td Qd Kh Kc
	// 	 Tc Jd 6s 5h 8c 8s Ad 5s 4s Ts 3h 3s 7h 7d 8h 2c 2d 5c 6h 2h 3d 7c
	// 	 9h 7s 4h 2s Jh 6d Qh 5d
	//
}
