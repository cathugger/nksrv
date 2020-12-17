package exifhelper

import (
	"io"

	"github.com/rwcarlsen/goexif/exif"
)

func ExifOrient(r io.Reader) int {
	x, err := exif.Decode(r)
	if err == nil && x != nil {
		orient, err := x.Get(exif.Orientation)
		if err == nil && orient != nil && orient.Count != 0 {
			if i, err := orient.Int(0); err == nil {
				return i
			}
		}
	}
	return 1
}

func RotWH(orient int, w, h int) (int, int) {
	switch orient {
	case 1, 2, 3, 4:
		// nothing
	case 5, 6, 7, 8:
		w, h = h, w
	}
	return w, h
}
