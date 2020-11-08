package xface

import (
	"errors"
	"image"
	"image/color"
	"image/draw"
)

var palWB = [2]color.Color{
	color.RGBA{0xff, 0xff, 0xff, 0xff},
	color.RGBA{0x00, 0x00, 0x00, 0xff},
}

func paleq(a, b color.Palette) bool {
	if &a[0] == &b[0] {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		xr, xg, xb, xa := a[i].RGBA()
		yr, yg, yb, ya := b[i].RGBA()
		if xr != yr || xg != yg || xb != yb || xa != ya {
			return false
		}
	}
	return true
}

func XFaceStringToImg(in string) (img image.Image, err error) {
	var bitmap [xfacePixels]byte

	err = xfaceDecodeString(&bitmap, in)
	if err != nil {
		return
	}

	// make actual image out of it
	// 0=white 1=black
	img = &image.Paletted{
		Pix:     bitmap[:],
		Stride:  xfaceWidth,
		Rect:    image.Rect(0, 0, xfaceWidth, xfaceHeight),
		Palette: palWB[:],
	}
	return
}

func XFaceImgToString(img image.Image) (s string, err error) {
	ib := img.Bounds()
	if ib.Dx() != 48 || ib.Dy() != 48 {
		err = errors.New("image isn't 48x48")
		return
	}

	pimg, ok := img.(*image.Paletted)
	if !ok || !paleq(palWB[:], pimg.Palette) {
		// need conversion probably
		pimg = image.NewPaletted(ib, palWB[:])
		draw.FloydSteinberg.Draw(pimg, ib, img, image.Point{})
	}
	// all gucci
	s = xfaceEncodeString(pimg.Pix)
	return
}
