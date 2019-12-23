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

// Basic sanity test: is first known value determined properly?
func TestSanity32x2(t *testing.T) {
	pcg := NewPCG32x2()
	result := pcg.Seed(1, 1, 2, 1).Random()
	expect := uint64(1107300197865787281)
	if result != expect {
		t.Errorf("NewPCG32x2().Seed(1, 1, 2, 1).Random() is %q; want %q", result, expect)
	}
}

var sumTests32x2 = []struct {
	state    uint64 // PCG seed value for state
	sequence uint64 // PCG seed value for sequence
	count    int    // number of values to sum
	sum      uint64 // sum of the first count values
}{
	{1, 1, 10, 8034187309725975364},
	{1, 1, 100, 14328956917741108809},
	{1, 1, 1000, 15814724732829753998},
	{1, 1, 10000, 8547922387302793844},
}

// Are the sums of the first few values consistent with expectation?
func TestSum32x2(t *testing.T) {
	for i, a := range sumTests32x2 {
		pcg := NewPCG32x2()
		pcg.Seed(a.state, a.state, a.sequence+1, a.sequence)
		sum := uint64(0)
		for j := 0; j < a.count; j++ {
			sum += uint64(pcg.Random())
		}
		if sum != a.sum {
			t.Errorf("#%d, sum of first %d values = %d; want %d", i, a.count, sum, a.sum)
		}
	}
}

const count32x2 = 256

// Does advancing work?
func TestAdvance32x2(t *testing.T) {
	pcg := NewPCG32x2()
	pcg.Seed(1, 1, 1, 2)
	values := make([]uint64, count32x2)
	for i := range values {
		values[i] = pcg.Random()
	}

	for skip := 1; skip < count32x2; skip++ {
		pcg.Seed(1, 1, 1, 2)
		pcg.Advance(uint64(skip))
		result := pcg.Random()
		expect := values[skip]
		if result != expect {
			t.Errorf("Advance(%d) is %d; want %d", skip, result, expect)
		}
	}
}

// Does retreating work?
func TestRetreat32x2(t *testing.T) {
	pcg := NewPCG32x2()
	pcg.Seed(1, 1, 1, 2)
	expect := pcg.Random()

	for skip := 1; skip < count32x2; skip++ {
		pcg.Seed(1, 1, 1, 2)
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

// Measure the time it takes to generate a 64-bit generator
func BenchmarkNew32x2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		pcg := NewPCG32x2()
		_ = pcg.Seed(1, 1, 1, 2)
	}
}

// Measure the time it takes to generate random values
func BenchmarkRandom32x2(b *testing.B) {
	b.StopTimer()
	pcg := NewPCG32x2()
	pcg.Seed(1, 1, 1, 2)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		_ = pcg.Random()
	}
}

// Measure the time it takes to generate bounded random values
func BenchmarkBounded32x2(b *testing.B) {
	b.StopTimer()
	pcg := NewPCG32x2()
	pcg.Seed(1, 1, 1, 2)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		_ = pcg.Bounded(uint64(i) & 0xff) // 0..255
		//_ = pcg.Bounded(1e19)
		// _ = pcg.Bounded(6)             // roll of die
		// _ = pcg.Bounded(52)            // deck of cards
		// _ = pcg.Bounded(365)           // day of year
	}
}

// Measure the time it takes to generate bounded random values
func BenchmarkBounded32x2Fast(b *testing.B) {
	b.StopTimer()
	pcg := NewPCG32x2()
	pcg.Seed(1, 1, 1, 2)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		_ = pcg.FastBounded(uint64(i) & 0xff) // 0..255
		//_ = pcg.FastBounded(1e19)
		// _ = pcg.Bounded(6)                 // roll of die
		// _ = pcg.Bounded(52)                // deck of cards
		// _ = pcg.Bounded(365)               // day of year
	}
}

//
// EXAMPLES
//

func ExampleReport32x2() {
	// Print report
	rng := NewPCG32x2()
	rng.Seed(42, 42, 54, 54)

	fmt.Printf("pcg32x2 random:\n"+
		"      -  result:      64-bit unsigned int (uint64)\n"+
		"      -  period:      2^64   (* ~2^126 streams)\n"+
		"      -  state type:  PGC32x2 (%d bytes)\n"+
		"      -  output func: XSH-RR (x 2)\n"+
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
		rng.Retreat(6)
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
	// pcg32x2 random:
	//       -  result:      64-bit unsigned int (uint64)
	//       -  period:      2^64   (* ~2^126 streams)
	//       -  state type:  PGC32x2 (32 bytes)
	//       -  output func: XSH-RR (x 2)
	//
	// Round 1:
	//   64bit: 0xa15c02b71a410f65 0x7b47f409e0b09a53 0xba1d333011fba8ac
	// 	 0x83d2f293452993e9 0xbfa4784b36082c12 0xcbed606ef5934191
	//   Again: 0xa15c02b71a410f65 0x7b47f409e0b09a53 0xba1d333011fba8ac
	// 	 0x83d2f293452993e9 0xbfa4784b36082c12 0xcbed606ef5934191
	//   Coins: TTTTHTHTTHHHHTHTTTTTTHHTHHTTHHHTHHHTHTHHHHHTHTHHHTHTTTHTHHTHTTTHT
	//   Rolls: 6 2 1 3 5 5 6 3 4 2 3 1 2 5 5 2 1 1 1 6 6 3 6 4 3 6 6 6 2 5 6 3 5
	//   Cards: As Qc 8c Ts 3s Kd 4s Js 6h Jc 9h 9s Ad 5d 9d 2s Ah 6s 8s Ks 4h 3d
	// 	 2c 7c 4d 5s 7h 5c Td Ac 8h 8d Kc Qd 5h 9c Tc 7s 2h 3c Th Kh Qs Jd
	// 	 7d 6c 4c 6d 3h 2d Jh Qh
	//
	// Round 2:
	//   64bit: 0x74ab93ad2c59d334 0x1c1da00063e4d9a8 0x494ff896d80ced60
	// 	 0x34462f2fabb62df0 0xd308a3e56ceea41f 0x0fa83bab7c561bd7
	//   Again: 0x74ab93ad2c59d334 0x1c1da00063e4d9a8 0x494ff896d80ced60
	// 	 0x34462f2fabb62df0 0xd308a3e56ceea41f 0x0fa83bab7c561bd7
	//   Coins: TTHHHTTTTHTTHHHTTHHHTTTHTTTTHHTHTHHHTHHHHTTTHHHTHTTHHTTTTHTHTHHTH
	//   Rolls: 5 5 6 2 2 3 1 2 1 1 6 4 2 1 3 2 3 3 6 6 2 1 5 6 3 4 4 6 3 3 2 5 5
	//   Cards: 3d Th 2c Jd Qh 2s Ac Jc 8h 3s 3h 5d 6c 9h 9c Js Qd Qc Qs Td 8d 9d
	// 	 4s Tc 7h 6h 8s Kh 7c Kc 8c Ah 9s 4c 6d 7d 5s 4h 5h Ad 6s 2d Kd 3c
	// 	 2h Ks Ts 4d 5c 7s As Jh
	//
	// Round 3:
	//   64bit: 0x39af5f9f70880799 0x04196b1831198a88 0xc3c3eb2826859955
	// 	 0xc076c60c822c3b19 0xc693e1351c41c1e0 0xf8f639327255b24d
	//   Again: 0x39af5f9f70880799 0x04196b1831198a88 0xc3c3eb2826859955
	// 	 0xc076c60c822c3b19 0xc693e1351c41c1e0 0xf8f639327255b24d
	//   Coins: THTTHHHHHTTTHTTHHTHTHTTHHHTHHHTHHTTTHTHHTHHTHHTTTHTTTHHHTTHTTTHHT
	//   Rolls: 6 6 5 3 4 4 2 3 1 1 5 6 1 6 2 5 4 2 2 1 1 3 6 2 6 2 1 5 2 2 5 2 4
	//   Cards: Jd Jc 8s Ks Ad Tc 6c Kd 8h 4s 2c 3s 6d 5c 2h 9d Qh Th 5h 3d 9c 8d
	// 	 Kc 4c Js 9s 7h Qd Qs 7s 4h 2d 2s 4d 3h 7d Qc 5d Jh 7c Ts 6s Ac Td
	// 	 As 8c 6h 9h Ah Kh 3c 5s
	//
	// Round 4:
	//   64bit: 0x55ce6851e2d99679 0x97a7726d339ab6aa 0x17e108157638b420
	// 	 0x58007d43742b5198 0x962fb1486083910e 0xb9bb55bddcb7a611
	//   Again: 0x55ce6851e2d99679 0x97a7726d339ab6aa 0x17e108157638b420
	// 	 0x58007d43742b5198 0x962fb1486083910e 0xb9bb55bddcb7a611
	//   Coins: TTHTTTTHHTTTHTHHHHHHTHHHTHHHHTHHTHTTHTHTHHTHHHHHTHTHHTTTTTHHTHHHT
	//   Rolls: 3 3 6 6 4 4 3 2 5 2 4 4 2 3 6 5 5 5 2 2 6 4 2 3 5 3 1 5 3 6 4 2 2
	//   Cards: Js Qd 4h 4d 2c 9h 8h Th Qs As 6d 5h 6c Ad 9s 7s Ts 2s 6h Qc 3c 5d
	// 	 Ac 7d Jc Jh Qh 3s 9c Kh Ah 8s 4c Tc 6s 8d Ks 3d 5c Jd 9d Kd 7c 4s
	// 	 Td Kc 8c 5s 2d 3h 7h 2h
	//
	// Round 5:
	//   64bit: 0xfcef7cd67f85ce72 0x1b488b5addfc8630 0xd0daf7eac38e77db
	// 	 0x1d9a70f727df7675 0x241a37cf77011f4d 0x9a3857b7be702c2b
	//   Again: 0xfcef7cd67f85ce72 0x1b488b5addfc8630 0xd0daf7eac38e77db
	// 	 0x1d9a70f727df7675 0x241a37cf77011f4d 0x9a3857b7be702c2b
	//   Coins: HHTHTTTTTTTHTHTTTTTHHHHTTTHHHHHHHHTHTHTHTTHTTTTTTHTHTHHHTTHHHTHTT
	//   Rolls: 3 4 6 3 5 4 2 4 2 3 3 1 5 5 5 4 6 3 2 3 2 5 5 6 3 2 4 4 6 5 5 6 2
	//   Cards: Ah 2s 8c Qh Jd As 3d Ks 4s Kc Qd Js Kd 7c 9s 8d 5d Kh 2h 9d Jh 3s
	// 	 7s Qs Qc Td 4h Th 6d 8s 6s Ad 9c 7d 6c Jc 2d 7h Tc Ac 8h 5c 3c 4d
	// 	 5h 5s 6h Ts 3h 9h 4c 2c
}
