package xface

// based on ffmpeg xface impl (LGPL) which was based on original compface
// TODO proper legal shit

/* define the face size - 48x48x1 */
const (
	xfaceWidth  = 48
	xfaceHeight = 48
	xfacePixels = xfaceWidth * xfaceHeight
)

/* compressed output uses the full range of printable characters.
 * In ASCII these are in a contiguous block so we just need to know
 * the first and last. The total number of printables is needed too. */
const (
	xfaceFirstPrint = '!'
	xfaceLastPrint  = '~'
	xfacePrints     = xfaceLastPrint - xfaceFirstPrint + 1
)

/* Each face is encoded using 9 octrees of 16x16 each. Each level of the
 * trees has varying probabilities of being white, grey or black.
 * The table below is based on sampling many faces */
const (
	xfaceColorBlack uint32 = iota
	xfaceColorGrey
	xfaceColorWhite
)

/* Data of varying probabilities are encoded by a value in the range 0 - 255.
 * The probability of the data determines the range of possible encodings.
 * Offset gives the first possible encoding of the range. */
type probRange struct {
	pRange  uint8
	pOffset uint8
}

var probRangesPerLevel = [4][3]probRange{
	//   black       grey        white
	{ {  1, 255}, {251,   0}, {  4, 251} }, /* Top of tree almost always grey */
	{ {  1, 255}, {200,   0}, { 55, 200} },
	{ { 33, 223}, {159,   0}, { 64, 159} },
	{ {131,   0}, {  0,   0}, {125, 131} }, /* Grey disallowed at bottom */
}

var probRanges2x2 = [16]probRange{
	{  0,   0}, { 38,   0}, { 38,  38}, { 13, 152},
	{ 38,  76}, { 13, 165}, { 13, 178}, {  6, 230},
	{ 38, 114}, { 13, 191}, { 13, 204}, {  6, 236},
	{ 13, 217}, {  6, 242}, {  5, 248}, {  3, 253},
}
