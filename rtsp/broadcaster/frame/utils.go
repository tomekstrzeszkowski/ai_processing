package frame

import (
	"image"
	"image/color"
)

func DecodeRawFrame(frame Frame) (image.Image, error) {
	width := int(frame.Width)
	height := int(frame.Height)
	bgrData := frame.Data
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := (y*width + x) * 3
			img.Set(x, y, color.RGBA{
				R: bgrData[i+2], // BGR -> RGB
				G: bgrData[i+1],
				B: bgrData[i],
				A: 255,
			})
		}
	}

	return img, nil
}
