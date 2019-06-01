// Copyright 2011 Dmitry Chestnykh. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package captcha

import "testing"

type byteCounter struct {
	n int64
}

func (bc *byteCounter) Write(b []byte) (int, error) {
	bc.n += int64(len(b))
	return len(b), nil
}

func BenchmarkNewImage(b *testing.B) {
	b.StopTimer()
	d := RandomDigits(5)
	var seed [16]byte
	copy(seed[:], randomBytes(16))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		NewImage(d, seed, StdWidth, StdHeight)
	}
}

func BenchmarkImageWriteTo(b *testing.B) {
	b.StopTimer()
	d := RandomDigits(5)
	var seed [16]byte
	copy(seed[:], randomBytes(16))
	b.StartTimer()
	counter := &byteCounter{}
	for i := 0; i < b.N; i++ {
		img := NewImage(d, seed, StdWidth, StdHeight)
		img.WriteTo(counter)
		b.SetBytes(counter.n)
		counter.n = 0
	}
}
