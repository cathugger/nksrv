package xface

// based on ffmpeg xface impl (LGPL) which was based on original compface
// TODO proper legal shit

/* define the face size - 48x48x1 */
const (
	xface_width  = 48
	xface_height = 48
	xface_pixels = xface_width * xface_height
)

/* compressed output uses the full range of printable characters.
 * In ASCII these are in a contiguous block so we just need to know
 * the first and last. The total number of printables is needed too. */
const (
	xface_first_print = '!'
	xface_last_print  = '~'
	xface_prints      = xface_last_print - xface_first_print + 1
)

/* Each face is encoded using 9 octrees of 16x16 each. Each level of the
 * trees has varying probabilities of being white, grey or black.
 * The table below is based on sampling many faces */
const (
	xface_color_black uint32 = iota
	xface_color_grey
	xface_color_white
)

/* Data of varying probabilities are encoded by a value in the range 0 - 255.
 * The probability of the data determines the range of possible encodings.
 * Offset gives the first possible encoding of the range. */
type probRange struct {
	p_range, p_offset uint8
}

var xface_probranges_per_level = [4][3]probRange{
	//  black      grey       white
	{ {  1, 255}, {251, 0}, {  4, 251} }, /* Top of tree almost always grey */
	{ {  1, 255}, {200, 0}, { 55, 200} },
	{ { 33, 223}, {159, 0}, { 64, 159} },
	{ {131,   0}, {  0, 0}, {125, 131} }, /* Grey disallowed at bottom */
}

var xface_probranges_2x2 = [16]probRange{
	{ 0,   0}, {38,   0}, {38,  38}, {13, 152},
	{38,  76}, {13, 165}, {13, 178}, { 6, 230},
	{38, 114}, {13, 191}, {13, 204}, { 6, 236},
	{13, 217}, { 6, 242}, { 5, 248}, { 3, 253},
}

/*
 * The "guess the next pixel" tables follow. Normally there are 12
 * neighbour pixels used to give 1<<12 cases as we get closer to the
 * upper left corner lesser numbers of neighbours are available.
 *
 * Each byte in the tables represents 8 boolean values starting from
 * the most significant bit.
 */

var g_00 = [...]uint8{
	0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0xe3, 0xdf, 0x05, 0x17,
	0x05, 0x0f, 0x00, 0x1b, 0x0f, 0xdf, 0x00, 0x04, 0x00, 0x00,
	0x0d, 0x0f, 0x03, 0x7f, 0x00, 0x00, 0x00, 0x01, 0x00, 0x1d,
	0x45, 0x2f, 0x00, 0x00, 0x00, 0x0d, 0x00, 0x0a, 0xff, 0xff,
	0x00, 0x04, 0x00, 0x05, 0x01, 0x3f, 0xcf, 0xff, 0x10, 0x01,
	0x80, 0xc9, 0x0f, 0x0f, 0xff, 0xff, 0x00, 0x00, 0x00, 0x00,
	0x1b, 0x1f, 0xff, 0xff, 0x4f, 0x54, 0x07, 0x1f, 0x57, 0x47,
	0xd7, 0x3d, 0xff, 0xff, 0x5f, 0x1f, 0x7f, 0xff, 0x7f, 0x7f,
	0x05, 0x0f, 0x01, 0x0f, 0x0f, 0x5f, 0x9b, 0xdf, 0x7f, 0xff,
	0x5f, 0x1d, 0x5f, 0xff, 0x0f, 0x1f, 0x0f, 0x5f, 0x03, 0x1f,
	0x4f, 0x5f, 0xf7, 0x7f, 0x7f, 0xff, 0x0d, 0x0f, 0xfb, 0xff,
	0xf7, 0xbf, 0x0f, 0x4f, 0xd7, 0x3f, 0x4f, 0x7f, 0xff, 0xff,
	0x67, 0xbf, 0x56, 0x25, 0x1f, 0x7f, 0x9f, 0xff, 0x00, 0x00,
	0x00, 0x05, 0x5f, 0x7f, 0x01, 0xdf, 0x14, 0x00, 0x05, 0x0f,
	0x07, 0xa2, 0x09, 0x0f, 0x00, 0x00, 0x00, 0x00, 0x0f, 0x5f,
	0x18, 0xd7, 0x94, 0x71, 0x00, 0x05, 0x1f, 0xb7, 0x0c, 0x07,
	0x0f, 0x0f, 0x00, 0x0f, 0x0f, 0x1f, 0x84, 0x8f, 0x05, 0x15,
	0x05, 0x0f, 0x4f, 0xff, 0x87, 0xdf, 0x05, 0x01, 0x10, 0x00,
	0x0f, 0x0f, 0x00, 0x08, 0x05, 0x04, 0x04, 0x01, 0x4f, 0xff,
	0x9f, 0x8f, 0x4a, 0x40, 0x5f, 0x5f, 0xff, 0xfe, 0xdf, 0xff,
	0x7f, 0xf7, 0xff, 0x7f, 0xff, 0xff, 0x7b, 0xff, 0x0f, 0xfd,
	0xd7, 0x5f, 0x4f, 0x7f, 0x7f, 0xdf, 0xff, 0xff, 0xff, 0xff,
	0xff, 0x77, 0xdf, 0x7f, 0x4f, 0xef, 0xff, 0xff, 0x77, 0xff,
	0xff, 0xff, 0x6f, 0xff, 0x0f, 0x4f, 0xff, 0xff, 0x9d, 0xff,
	0x0f, 0xef, 0xff, 0xdf, 0x6f, 0xff, 0xff, 0xff, 0x4f, 0xff,
	0xcd, 0x0f, 0x4f, 0xff, 0xff, 0xdf, 0x00, 0x00, 0x00, 0x0b,
	0x05, 0x02, 0x02, 0x0f, 0x04, 0x00, 0x00, 0x0c, 0x01, 0x06,
	0x00, 0x0f, 0x20, 0x03, 0x00, 0x00, 0x05, 0x0f, 0x40, 0x08,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x0c, 0x0f, 0x01, 0x00,
	0x80, 0x00, 0x00, 0x00, 0x80, 0x00, 0x00, 0x14, 0x01, 0x05,
	0x01, 0x15, 0xaf, 0x0f, 0x00, 0x01, 0x10, 0x00, 0x08, 0x00,
	0x46, 0x0c, 0x20, 0x00, 0x88, 0x00, 0x0f, 0x15, 0xff, 0xdf,
	0x02, 0x00, 0x00, 0x0f, 0x7f, 0x5f, 0xdb, 0xff, 0x4f, 0x3e,
	0x05, 0x0f, 0x7f, 0xf7, 0x95, 0x4f, 0x0d, 0x0f, 0x01, 0x0f,
	0x4f, 0x5f, 0x9f, 0xdf, 0x25, 0x0e, 0x0d, 0x0d, 0x4f, 0x7f,
	0x8f, 0x0f, 0x0f, 0xfa, 0x04, 0x4f, 0x4f, 0xff, 0xf7, 0x77,
	0x47, 0xed, 0x05, 0x0f, 0xff, 0xff, 0xdf, 0xff, 0x4f, 0x6f,
	0xd8, 0x5f, 0x0f, 0x7f, 0xdf, 0x5f, 0x07, 0x0f, 0x94, 0x0d,
	0x1f, 0xff, 0xff, 0xff, 0x00, 0x02, 0x00, 0x03, 0x46, 0x57,
	0x01, 0x0d, 0x01, 0x08, 0x01, 0x0f, 0x47, 0x6c, 0x0d, 0x0f,
	0x02, 0x00, 0x00, 0x00, 0x0b, 0x4f, 0x00, 0x08, 0x05, 0x00,
	0x95, 0x01, 0x0f, 0x7f, 0x0c, 0x0f, 0x01, 0x0e, 0x00, 0x00,
	0x0f, 0x41, 0x00, 0x00, 0x04, 0x24, 0x0d, 0x0f, 0x0f, 0x7f,
	0xcf, 0xdf, 0x00, 0x00, 0x00, 0x00, 0x04, 0x40, 0x00, 0x00,
	0x06, 0x26, 0xcf, 0x05, 0xcf, 0x7f, 0xdf, 0xdf, 0x00, 0x00,
	0x17, 0x5f, 0xff, 0xfd, 0xff, 0xff, 0x46, 0x09, 0x4f, 0x5f,
	0x7f, 0xfd, 0xdf, 0xff, 0x0a, 0x88, 0xa7, 0x7f, 0x7f, 0xff,
	0xff, 0xff, 0x0f, 0x04, 0xdf, 0x7f, 0x4f, 0xff, 0x9f, 0xff,
	0x0e, 0xe6, 0xdf, 0xff, 0x7f, 0xff, 0xff, 0xff, 0x0f, 0xec,
	0x8f, 0x4f, 0x7f, 0xff, 0xdf, 0xff, 0x0f, 0xcf, 0xdf, 0xff,
	0x6f, 0x7f, 0xff, 0xff, 0x03, 0x0c, 0x9d, 0x0f, 0x7f, 0xff,
	0xff, 0xff,
}
var g_01 = [...]uint8{
	0x37, 0x73, 0x00, 0x19, 0x57, 0x7f, 0xf5, 0xfb, 0x70, 0x33,
	0xf0, 0xf9, 0x7f, 0xff, 0xff, 0xff,
}
var g_02 = [...]uint8{
	0x50,
}
var g_10 = [...]uint8{
	0x00, 0x00, 0x00, 0x00, 0x50, 0x00, 0xf3, 0x5f, 0x84, 0x04,
	0x17, 0x9f, 0x04, 0x23, 0x05, 0xff, 0x00, 0x00, 0x00, 0x02,
	0x03, 0x03, 0x33, 0xd7, 0x05, 0x03, 0x5f, 0x3f, 0x17, 0x33,
	0xff, 0xff, 0x00, 0x80, 0x02, 0x04, 0x12, 0x00, 0x11, 0x57,
	0x05, 0x25, 0x05, 0x03, 0x35, 0xbf, 0x9f, 0xff, 0x07, 0x6f,
	0x20, 0x40, 0x17, 0x06, 0xfa, 0xe8, 0x01, 0x07, 0x1f, 0x9f,
	0x1f, 0xff, 0xff, 0xff,
}
var g_20 = [...]uint8{
	0x04, 0x00, 0x01, 0x01, 0x43, 0x2e, 0xff, 0x3f,
}
var g_30 = [...]uint8{
	0x11, 0x11, 0x11, 0x11, 0x51, 0x11, 0x13, 0x11, 0x11, 0x11,
	0x13, 0x11, 0x11, 0x11, 0x33, 0x11, 0x13, 0x11, 0x13, 0x13,
	0x13, 0x13, 0x31, 0x31, 0x11, 0x01, 0x11, 0x11, 0x71, 0x11,
	0x11, 0x75,
}
var g_40 = [...]uint8{
	0x00, 0x0f, 0x00, 0x09, 0x00, 0x0d, 0x00, 0x0d, 0x00, 0x0f,
	0x00, 0x4e, 0xe4, 0x0d, 0x10, 0x0f, 0x00, 0x0f, 0x44, 0x4f,
	0x00, 0x1e, 0x0f, 0x0f, 0xae, 0xaf, 0x45, 0x7f, 0xef, 0xff,
	0x0f, 0xff, 0x00, 0x09, 0x01, 0x11, 0x00, 0x01, 0x1c, 0xdd,
	0x00, 0x15, 0x00, 0xff, 0x00, 0x10, 0x00, 0xfd, 0x00, 0x0f,
	0x4f, 0x5f, 0x3d, 0xff, 0xff, 0xff, 0x4f, 0xff, 0x1c, 0xff,
	0xdf, 0xff, 0x8f, 0xff, 0x00, 0x0d, 0x00, 0x00, 0x00, 0x15,
	0x01, 0x07, 0x00, 0x01, 0x02, 0x1f, 0x01, 0x11, 0x05, 0x7f,
	0x00, 0x1f, 0x41, 0x57, 0x1f, 0xff, 0x05, 0x77, 0x0d, 0x5f,
	0x4d, 0xff, 0x4f, 0xff, 0x0f, 0xff, 0x00, 0x00, 0x02, 0x05,
	0x00, 0x11, 0x05, 0x7d, 0x10, 0x15, 0x2f, 0xff, 0x40, 0x50,
	0x0d, 0xfd, 0x04, 0x0f, 0x07, 0x1f, 0x07, 0x7f, 0x0f, 0xbf,
	0x0d, 0x7f, 0x0f, 0xff, 0x4d, 0x7d, 0x0f, 0xff,
}
var g_11 = [...]uint8{
	0x01, 0x13, 0x03, 0x7f,
}
var g_21 = [...]uint8{
	0x17,
}
var g_31 = [...]uint8{
	0x55, 0x57, 0x57, 0x7f,
}
var g_41 = [...]uint8{
	0x01, 0x01, 0x01, 0x1f, 0x03, 0x1f, 0x3f, 0xff,
}
var g_12 = [...]uint8{
	0x40,
}
var g_22 = [...]uint8{
	0x00,
}
var g_32 = [...]uint8{
	0x10,
}
var g_42 = [...]uint8{
	0x10,
}

func xface_generate_face(dst, src []uint8) {
	var h, i, j, l, m int32
	var k uint32

	// NOTE:
	// this filter loop actually doesn't match up with technique described in comments,
	// and is off-by-one both horizontally and vertically;
	// g_3* aren't even reached, while border pixels fall into more common tables.
	// too late to change that now (it'd essentially break format),
	// just describing it there to help anyone else trying to read into this.

	for j = 0; j < xface_height; j++ {
		for i = 0; i < xface_width; i++ {
			h = i + j*xface_width
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
					if l <= xface_width && m > 0 {
						k = 2*k + uint32(src[l+m*xface_width])
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

			GEN := func(table []uint8) { dst[h] ^= (table[k>>3] >> (7 - (k & 7))) & 1 }

			switch i {
			case 1:
				switch j {
				case 1:
					GEN(g_22[:])
				case 2:
					GEN(g_21[:])
				default:
					GEN(g_20[:])
				}
			case 2:
				switch j {
				case 1:
					GEN(g_12[:])
				case 2:
					GEN(g_11[:])
				default:
					GEN(g_10[:])
				}
			case xface_width - 1:
				switch j {
				case 1:
					GEN(g_42[:])
				case 2:
					GEN(g_41[:])
				default:
					GEN(g_40[:])
				}
			case xface_width:
				switch j {
				case 1:
					GEN(g_32[:])
				case 2:
					GEN(g_31[:])
				default:
					GEN(g_30[:])
				}
			default:
				switch j {
				case 1:
					GEN(g_02[:])
				case 2:
					GEN(g_01[:])
				default:
					GEN(g_00[:])
				}
			}
		}
	}
}
