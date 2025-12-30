package frame

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
)

func YuvToJpeg(yuvData []byte, width, height int) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	ySize := width * height
	uSize := ySize / 4

	yPlane := yuvData[:ySize]
	uPlane := yuvData[ySize : ySize+uSize]
	vPlane := yuvData[ySize+uSize:]

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			yIndex := y*width + x
			uvIndex := (y/2)*(width/2) + (x / 2)

			yVal := float64(yPlane[yIndex])
			uVal := float64(uPlane[uvIndex]) - 128
			vVal := float64(vPlane[uvIndex]) - 128

			r := yVal + 1.402*vVal
			g := yVal - 0.344*uVal - 0.714*vVal
			b := yVal + 1.772*uVal

			img.SetRGBA(x, y, color.RGBA{
				R: uint8(clamp(r, 0, 255)),
				G: uint8(clamp(g, 0, 255)),
				B: uint8(clamp(b, 0, 255)),
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func clamp(val, min, max float64) float64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
func BytesToYCbCr(data []byte, width, height int) *image.YCbCr {
	ySize := width * height
	cSize := ySize / 4

	return &image.YCbCr{
		Y:              data[:ySize],
		Cb:             data[ySize : ySize+cSize],
		Cr:             data[ySize+cSize:],
		YStride:        width,
		CStride:        width / 2,
		SubsampleRatio: image.YCbCrSubsampleRatio420,
		Rect:           image.Rect(0, 0, width, height),
	}
}
