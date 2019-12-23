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

const count64s = 256

// Does advancing work?
func TestAdvance64s(t *testing.T) {
	pcg := NewPCG64s()
	pcg.Seed(0, 1)
	values := make([]uint64, count64s)
	for i := range values {
		values[i] = pcg.Random()
	}

	for skip := 1; skip < count64s; skip++ {
		pcg.Seed(0, 1)
		pcg.Advance(0, uint64(skip))
		result := pcg.Random()
		expect := values[skip]
		if result != expect {
			t.Errorf("Advance(%d) is %d; want %d", skip, result, expect)
		}
	}
}

// Does retreating work?
func TestRetreat64s(t *testing.T) {
	pcg := NewPCG64s()
	pcg.Seed(0, 1)
	expect := pcg.Random()

	for skip := 1; skip < count64s; skip++ {
		pcg.Seed(0, 1)
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
func BenchmarkNew64s(b *testing.B) {
	for i := 0; i < b.N; i++ {
		pcg := NewPCG64s()
		_ = pcg.Seed(0, 1)
	}
}

// Measure the time it takes to generate random values
func BenchmarkRandom64s(b *testing.B) {
	b.StopTimer()
	pcg := NewPCG64s()
	pcg.Seed(0, 1)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		_ = pcg.Random()
	}
}

// Measure the time it takes to generate bounded random values
func BenchmarkBounded64s(b *testing.B) {
	b.StopTimer()
	pcg := NewPCG64s()
	pcg.Seed(0, 1)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		_ = pcg.Bounded(uint64(i) & 0xff) // 0..255
		//_ = pcg.Bounded(1e9)
		// _ = pcg.Bounded(6)             // roll of die
		// _ = pcg.Bounded(52)            // deck of cards
		// _ = pcg.Bounded(365)           // day of year
	}
}

// Measure the time it takes to generate bounded random values
func BenchmarkBounded64sFast(b *testing.B) {
	b.StopTimer()
	pcg := NewPCG64s()
	pcg.Seed(0, 1)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		_ = pcg.FastBounded(uint64(i) & 0xff) // 0..255
		//_ = pcg.FastBounded(1e9)
		// _ = pcg.Bounded(6)                 // roll of die
		// _ = pcg.Bounded(52)                // deck of cards
		// _ = pcg.Bounded(365)               // day of year
	}
}

//
// EXAMPLES
//

func ExampleReport64s() {
	// Print report
	rng := NewPCG64s()
	rng.Seed(0, 42)

	fmt.Printf("pcg64s random:\n"+
		"      -  result:      64-bit unsigned int (uint64)\n"+
		"      -  period:      2^128\n"+
		"      -  state type:  PGC64s (%d bytes)\n"+
		"      -  output func: XSL-RR\n"+
		"\n",
		unsafe.Sizeof(rng))

	for round := 1; round <= 5; round++ {
		fmt.Printf("Round %d:\n", round)

		/* Make some 64-bit numbers */
		fmt.Printf("  64bit:")
		for i := 0; i < 6; i++ {
			fmt.Printf(" 0x%016x", rng.Random())
			if i+1 == 3 {
				fmt.Printf("\n\t")
			}
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
		for i := uint64(CARDS); i > 1; i-- {
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
	// pcg64s random:
	//       -  result:      64-bit unsigned int (uint64)
	//       -  period:      2^128
	//       -  state type:  PGC64s (16 bytes)
	//       -  output func: XSL-RR
	//
	// Round 1:
	//   64bit: 0x287472e87ff5705a 0xbbd190b04ed0b545 0xb6cee3580db14880
	// 	 0xbf5f7d7e4c3d1864 0x734eedbe7e50bbc5 0xa5b6b5f867691c77
	//   Again: 0x287472e87ff5705a 0xbbd190b04ed0b545 0xb6cee3580db14880
	// 	 0xbf5f7d7e4c3d1864 0x734eedbe7e50bbc5 0xa5b6b5f867691c77
	//   Coins: HHTHHHTTHTHHTTTHHHTHHHTTTTTHHTHTTTTHHTHHTTTTTTHHTHTHTTTTTHTHHTTHT
	//   Rolls: 1 2 6 3 6 2 6 5 3 2 3 2 1 5 1 6 1 3 3 5 4 3 1 5 1 4 6 4 1 6 5 5 5
	//   Cards: 9c 5h 7d 7c 4c 8d 7h Qc Kh 2d 3h 2h Qd Ts 3d Kc 9h Jc 6h 6d 8c 4d
	// 	 Qh As Jh 8s Th 5s 2c 9d Ac 4h Kd 5d 9s 6c 3s Ks Js Jd 7s 2s 3c Tc
	// 	 Qs 4s 8h 5c Ah 6s Td Ad
	//
	// Round 2:
	//   64bit: 0x7d97ee72fb94fdf0 0xb35f07d53cc42b66 0x0854c5caec0c251f
	// 	 0xf37961a645554320 0x1d1d213622351b24 0x6edbb396c73fb49f
	//   Again: 0x7d97ee72fb94fdf0 0xb35f07d53cc42b66 0x0854c5caec0c251f
	// 	 0xf37961a645554320 0x1d1d213622351b24 0x6edbb396c73fb49f
	//   Coins: HHTTHHHTHHHHHTTHHHTHHTHTHTTTHHTHHHHHHTHTTHHHHHTHTTHTHHHTHHTTHHHHH
	//   Rolls: 5 4 2 2 5 3 2 2 2 4 3 1 2 5 6 1 6 5 3 1 1 3 5 6 4 2 5 3 1 2 2 4 1
	//   Cards: Qh Ah 3c 6d 8s 3h Jh 8c 6s 9h 8d 3s 9c Ts Qd Kc Kh 4d 7c 5h Th 2s
	// 	 2c 4c 6h Jd Td 2d Ks 5c 7h Qs 3d 4s 7d 5s 9d Ac 6c 9s 8h As Jc 7s
	// 	 2h Tc Kd 4h Ad 5d Qc Js
	//
	// Round 3:
	//   64bit: 0x187ee00430cec695 0x38efe3fb60c70613 0x3949bd01ef38c552
	// 	 0xd3f1543a45f3b48f 0xfb81a0482dc602cd 0xb48e4f661e4c7fc5
	//   Again: 0x187ee00430cec695 0x38efe3fb60c70613 0x3949bd01ef38c552
	// 	 0xd3f1543a45f3b48f 0xfb81a0482dc602cd 0xb48e4f661e4c7fc5
	//   Coins: THTTTHHTHHHHHHTHTTTTHHTHTHTHHHTHTTTTTHHTHTTTTTHTTTTTHHTHHHTTTTHHT
	//   Rolls: 1 5 6 2 1 3 2 5 5 6 3 4 4 1 5 3 5 1 4 1 2 4 4 1 3 5 6 5 4 5 5 6 3
	//   Cards: 9c 4s 9d Kh Js 6c 4d 8h Qd 8d 5s 3d 6h 7s 9s Ks 2s 7h Ac Ad Kc 4c
	// 	 5h Jd As Ah 8s 2h Jh 6s Th 2c Qc 8c 5d 9h 3h 4h Qh 3s Tc 5c 2d Qs
	// 	 3c Ts Kd Jc 7c 6d 7d Td
	//
	// Round 4:
	//   64bit: 0xd04c0a3a8cf6c571 0xbc94812fe9ec2c93 0x691f3e3aa2f42c77
	// 	 0xb7188d5162d89a1e 0x17fbf02e08fee28a 0x1aa17486e288664f
	//   Again: 0xd04c0a3a8cf6c571 0xbc94812fe9ec2c93 0x691f3e3aa2f42c77
	// 	 0xb7188d5162d89a1e 0x17fbf02e08fee28a 0x1aa17486e288664f
	//   Coins: THTHHHTTTHHHHHTTHHHHHTTHHTTHTHTTHHHHHTHTTHTTTTHTTTHTHHHHTTHHTTHTH
	//   Rolls: 3 6 5 6 2 3 5 6 2 5 1 5 3 5 6 2 3 1 1 3 1 6 6 4 4 5 4 4 4 2 4 6 5
	//   Cards: 6d 4c 9c 7s Qs Qh 5d 5h Jd 8d 3c 3d 9h Th 7d Tc As 7h 9s 2d 2s Td
	// 	 8s 4s 7c Qd Jc 5c 5s 2c 2h 6s 6c 4h Js Ks 8h 6h Jh Kh 4d 3s Ah 9d
	// 	 3h Kc 8c Qc Kd Ts Ad Ac
	//
	// Round 5:
	//   64bit: 0x10bda17a1292d5aa 0xf0cd1384e25b3497 0x8e592be49a6a6181
	// 	 0x5edc4faf5cda5865 0xb2ecea43437a3f8c 0x98dbb99c3550f0e4
	//   Again: 0x10bda17a1292d5aa 0xf0cd1384e25b3497 0x8e592be49a6a6181
	// 	 0x5edc4faf5cda5865 0xb2ecea43437a3f8c 0x98dbb99c3550f0e4
	//   Coins: HTHTHHHHHHTHHTTHHHTTTTTTHTTTHHHTHHHHHHHHHTTTTHHTTTHTHHHHTTHHHHTTH
	//   Rolls: 5 1 4 6 1 4 2 4 2 1 3 2 4 3 6 3 5 5 4 5 1 2 1 1 1 6 5 6 5 4 1 6 4
	//   Cards: 3d 7h Jd 3h 5d 2s 7d 5s 9d 8s 4c 8c Jc Ah Tc Js Ad 5h 6h Th 7s Kd
	// 	 Qd 9h As 8d 3c 2d 2c Kh 9s 7c Ac 6c 9c 2h Qs 4h Ts 6s Ks Jh 5c 8h
	// 	 3s 6d Kc 4d Td Qh Qc 4s
	//
}
