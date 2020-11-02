package xface

/*
 * The "guess the next pixel" tables follow. Normally there are 12
 * neighbour pixels used to give 1<<12 cases as we get closer to the
 * upper left corner lesser numbers of neighbours are available.
 *
 * Each byte in the tables represents 8 boolean values starting from
 * the most significant bit.
 */

 var g00 = [...]uint8{
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
var g01 = [...]uint8{
	0x37, 0x73, 0x00, 0x19, 0x57, 0x7f, 0xf5, 0xfb, 0x70, 0x33,
	0xf0, 0xf9, 0x7f, 0xff, 0xff, 0xff,
}
var g02 = [...]uint8{
	0x50,
}
var g10 = [...]uint8{
	0x00, 0x00, 0x00, 0x00, 0x50, 0x00, 0xf3, 0x5f, 0x84, 0x04,
	0x17, 0x9f, 0x04, 0x23, 0x05, 0xff, 0x00, 0x00, 0x00, 0x02,
	0x03, 0x03, 0x33, 0xd7, 0x05, 0x03, 0x5f, 0x3f, 0x17, 0x33,
	0xff, 0xff, 0x00, 0x80, 0x02, 0x04, 0x12, 0x00, 0x11, 0x57,
	0x05, 0x25, 0x05, 0x03, 0x35, 0xbf, 0x9f, 0xff, 0x07, 0x6f,
	0x20, 0x40, 0x17, 0x06, 0xfa, 0xe8, 0x01, 0x07, 0x1f, 0x9f,
	0x1f, 0xff, 0xff, 0xff,
}
var g20 = [...]uint8{
	0x04, 0x00, 0x01, 0x01, 0x43, 0x2e, 0xff, 0x3f,
}
var g30 = [...]uint8{
	0x11, 0x11, 0x11, 0x11, 0x51, 0x11, 0x13, 0x11, 0x11, 0x11,
	0x13, 0x11, 0x11, 0x11, 0x33, 0x11, 0x13, 0x11, 0x13, 0x13,
	0x13, 0x13, 0x31, 0x31, 0x11, 0x01, 0x11, 0x11, 0x71, 0x11,
	0x11, 0x75,
}
var g40 = [...]uint8{
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
var g11 = [...]uint8{
	0x01, 0x13, 0x03, 0x7f,
}
var g21 = [...]uint8{
	0x17,
}
var g31 = [...]uint8{
	0x55, 0x57, 0x57, 0x7f,
}
var g41 = [...]uint8{
	0x01, 0x01, 0x01, 0x1f, 0x03, 0x1f, 0x3f, 0xff,
}
var g12 = [...]uint8{
	0x40,
}
var g22 = [...]uint8{
	0x00,
}
var g32 = [...]uint8{
	0x10,
}
var g42 = [...]uint8{
	0x10,
}