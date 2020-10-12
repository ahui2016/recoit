package graphics

import (
	"bytes"
	"image"
	"image/jpeg"
	"math"

	"github.com/disintegration/imaging"
	"golang.org/x/image/webp"
)

const (
	thumbSize = 128
	quality   = 85
	longLimit = 1000
)

// ResizeLimit resizes the image if it's long side bigger than limit.
func ResizeLimit(img []byte) (*bytes.Buffer, error) {
	src, err := ReadImage(img)
	w, h := limitWidthHeight(src.Bounds())
	small := imaging.Resize(src, w, h, imaging.NearestNeighbor)

	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, small, &jpeg.Options{Quality: quality})
	return buf, err
}

// Thumbnail create a thumbnail of imgFile.
func Thumbnail(img []byte) (*bytes.Buffer, error) {
	src, err := ReadImage(img)
	side := shortSide(src.Bounds())
	src = imaging.CropCenter(src, side, side)
	src = imaging.Resize(src, thumbSize, 0, imaging.NearestNeighbor)
	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, src, &jpeg.Options{Quality: quality})
	return buf, err
}

// ReadImage .
func ReadImage(img []byte) (image.Image, error) {
	r := bytes.NewReader(img)
	src, err := imaging.Decode(r, imaging.AutoOrientation(true))
	if err != nil {
		r.Reset(img)
		if src, err = webp.Decode(r); err != nil {
			return nil, err
		}
	}
	return src, nil
}

func shortSide(bounds image.Rectangle) int {
	if bounds.Dx() < bounds.Dy() {
		return bounds.Dx()
	}
	return bounds.Dy()
}

func limitWidthHeight(bounds image.Rectangle) (limitWidth, limitHeight int) {
	w := float64(bounds.Dx())
	h := float64(bounds.Dy())
	// 先限制宽度
	if w > longLimit {
		h *= longLimit / w
		w = longLimit
	}
	// 缩小后的高度仍有可能超过限制，因此要再判断一次
	if h > longLimit {
		w *= longLimit / h
		h = longLimit
	}
	limitWidth = int(math.Round(w))
	limitHeight = int(math.Round(h))
}
