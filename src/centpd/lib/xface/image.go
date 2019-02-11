package xface

import "image"

func XFaceStringToImg(in string) (img image.Image, err error) {
	var bitmap [xface_pixels]byte

	err = xface_decode_string(&bitmap, in)
	if err != nil {
		return
	}

	// make actual image out of it
	// 0=white 1=black
	img = &image.Paletted{
		Pix:     bitmap[:],
		Stride:  xface_width,
		Rect:    image.Rect(0, 0, xface_width, xface_height),
		Palette: palWB[:],
	}
	return
}
