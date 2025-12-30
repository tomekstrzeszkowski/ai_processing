package frame

import (
	"bytes"
	"image"
	"image/jpeg"
)

func DecodeRawFrame(frame Frame) (image.Image, error) {
	reader := bytes.NewReader(frame.Data)
	return jpeg.Decode(reader)
}
