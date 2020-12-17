package xface

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
