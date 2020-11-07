package xface

func generateFace(dst, src []uint8) {
	var h, i, j, l, m int32
	var k uint32

	// NOTE:
	// this filter loop actually doesn't match up with technique described in comments,
	// and is off-by-one both horizontally and vertically;
	// g3* aren't even reached, while border pixels fall into more common tables.
	// too late to change that now (it'd essentially break format),
	// just describing it there to help anyone else trying to read into this.

	for j = 0; j < xfaceHeight; j++ {
		for i = 0; i < xfaceWidth; i++ {
			h = i + j*xfaceWidth
			k = 0

			/*
			        Compute k, encoding the bits *before* the current one, contained in the
			        image buffer. That is, given the grid:

			         l      i
			         |      |
			         v      v
			        +--+--+--+--+--+
			   m -> | 1| 2| 3| 4| 5|
			        +--+--+--+--+--+
			        | 6| 7| 8| 9|10|
			        +--+--+--+--+--+
			   j -> |11|12| *|  |  |
			        +--+--+--+--+--+

			        the value k for the pixel marked as "*" will contain the bit encoding of
			        the values in the matrix marked from "1" to "12". In case the pixel is
			        near the border of the grid, the number of values contained within the
			        grid will be lesser than 12.
			*/

			for l = i - 2; l <= i+2; l++ {
				for m = j - 2; m <= j; m++ {
					if l <= 0 || (l >= i && m == j) {
						continue
					}
					if l <= xfaceWidth && m > 0 {
						k = 2*k + uint32(src[l+m*xfaceWidth])
					}
				}
			}

			/*
			      Use the guess for the given position and the computed value of k.

			      The following table shows the number of digits in k, depending on
			      the position of the pixel, and shows the corresponding guess table
			      to use:

			         i=1  i=2  i=3       i=w-1 i=w
			       +----+----+----+ ... +----+----+
			   j=1 |  0 |  1 |  2 |     |  2 |  2 |
			       |g22 |g12 |g02 |     |g42 |g32 |
			       +----+----+----+ ... +----+----+
			   j=2 |  3 |  5 |  7 |     |  6 |  5 |
			       |g21 |g11 |g01 |     |g41 |g31 |
			       +----+----+----+ ... +----+----+
			   j=3 |  5 |  9 | 12 |     | 10 |  8 |
			       |g20 |g10 |g00 |     |g40 |g30 |
			       +----+----+----+ ... +----+----+
			*/

			gen := func(table []uint8) { dst[h] ^= (table[k>>3] >> (7 - (k & 7))) & 1 }

			switch i {
			case 1:
				switch j {
				case 1:
					gen(g22[:])
				case 2:
					gen(g21[:])
				default:
					gen(g20[:])
				}
			case 2:
				switch j {
				case 1:
					gen(g12[:])
				case 2:
					gen(g11[:])
				default:
					gen(g10[:])
				}
			case xfaceWidth - 1:
				switch j {
				case 1:
					gen(g42[:])
				case 2:
					gen(g41[:])
				default:
					gen(g40[:])
				}
			case xfaceWidth:
				switch j {
				case 1:
					gen(g32[:])
				case 2:
					gen(g31[:])
				default:
					gen(g30[:])
				}
			default:
				switch j {
				case 1:
					gen(g02[:])
				case 2:
					gen(g01[:])
				default:
					gen(g00[:])
				}
			}
		}
	}
}
