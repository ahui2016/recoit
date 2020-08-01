package graphics

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/jpeg"
	"os"

	"github.com/disintegration/imaging"
	"golang.org/x/image/webp"
)

// Thumbnail create a thumbnail of imgFile, and encodes it to base64 string.
func Thumbnail(imgFile string) (string, error) {
	src, err := imaging.Open(imgFile, imaging.AutoOrientation(true))
	if err != nil {
		file, err := os.Open(imgFile)
		if err != nil {
			return "", err
		}
		src, err = webp.Decode(file)
		if err != nil {
			return "", err
		}
	}
	side := shortSide(src.Bounds())
	src = imaging.CropCenter(src, side, side)
	src = imaging.Resize(src, 96, 0, imaging.NearestNeighbor)
	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, src, &jpeg.Options{Quality: 80})
	blob := base64.StdEncoding.EncodeToString(buf.Bytes())
	return blob, nil
}

func shortSide(bounds image.Rectangle) int {
	if bounds.Dx() < bounds.Dy() {
		return bounds.Dx()
	}
	return bounds.Dy()
}
