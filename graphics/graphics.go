package graphics

import (
	"bytes"
	"image"
	"image/jpeg"
	"os"

	"github.com/disintegration/imaging"
	"golang.org/x/image/webp"
)

const (
	thumbSize = 128
	quality   = 80
)

// Thumbnail create a thumbnail of imgFile.
func Thumbnail(imgFile string) (*bytes.Buffer, error) {
	src, err := imaging.Open(imgFile, imaging.AutoOrientation(true))
	if err != nil {
		file, err := os.Open(imgFile)
		if err != nil {
			return nil, err
		}
		src, err = webp.Decode(file)
		if err != nil {
			return nil, err
		}
	}
	side := shortSide(src.Bounds())
	src = imaging.CropCenter(src, side, side)
	src = imaging.Resize(src, thumbSize, 0, imaging.NearestNeighbor)
	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, src, &jpeg.Options{Quality: quality})
	return buf, err
}

func shortSide(bounds image.Rectangle) int {
	if bounds.Dx() < bounds.Dy() {
		return bounds.Dx()
	}
	return bounds.Dy()
}
