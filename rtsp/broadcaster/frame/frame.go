package frame

type Frame struct {
	Data     []byte
	Width    uint32
	Height   uint32
	Detected int
	Fps      float64
}
