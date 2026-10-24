package web_rtc

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

type FFProbeOutput struct {
	Streams []struct {
		RFrameRate string `json:"r_frame_rate"`
	} `json:"streams"`
}

func GetFrameDuration(filePath string) (time.Duration, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-select_streams", "v:0",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe failed: %w", err)
	}

	var probe FFProbeOutput
	if err := json.Unmarshal(output, &probe); err != nil {
		return 0, err
	}

	if len(probe.Streams) == 0 {
		return 0, fmt.Errorf("no video stream found")
	}

	// Parse "30/1" format
	var num, den int
	fmt.Sscanf(probe.Streams[0].RFrameRate, "%d/%d", &num, &den)
	fps := float64(num) / float64(den)

	return time.Duration(float64(time.Second) / fps), nil
}
