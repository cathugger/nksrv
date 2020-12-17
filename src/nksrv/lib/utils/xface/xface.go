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
