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
	quality   = 85
	longLimit = 1000
)

// ResizeLimit resizes the image if it's long side bigger than limit.
func ResizeLimit(imgFile []byte) (*bytes.Buffer, error) {
	r := bytes.NewReader(imgFile)
	src, err := imaging.Decode(r, imaging.AutoOrientation(true))
	if err != nil {
		r.Reset(imgFile)
		if src, err = webp.Decode(r); err != nil {
			return nil, err
		}
	}
	w, h := limitWidthHeight(src.Bounds())
	small := imaging.Resize(src, w, h, imaging.NearestNeighbor)

	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, small, &jpeg.Options{Quality: quality})
	return buf, err
}

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

func limitWidthHeight(bounds image.Rectangle) (limitWidth, limitHeight int) {
	w, h := bounds.Dx(), bounds.Dy()
	// 先限制宽度
	if w > longLimit {
		w = longLimit
		h *= longLimit / w
	}
	// 缩小后的高度仍有可能超过限制，因此要再判断一次
	if h > longLimit {
		h = longLimit
		w *= longLimit / h
	}
	return w, h
}
