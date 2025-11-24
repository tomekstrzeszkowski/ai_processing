package web_rtc

import "strzcam.com/broadcaster/video"

type SignalingMessage struct {
	Type string                   `json:"type"`
	Sdp  string                   `json:"sdp,omitempty"`
	Ice  []map[string]interface{} `json:"ice,omitempty"`
}

type DataChannelMessage struct {
	Type      string  `json:"type"`
	StartDate string  `json:"startDate,omitempty"`
	EndDate   string  `json:"endDate,omitempty"`
	VideoName string  `json:"videoName,omitempty"`
	Seek      float64 `json:"seek,omitempty"`
	Sdp       string  `json:"sdp,omitempty"`
}

type VideoListMessage struct {
	Type      string        `json:"type"`
	VideoList []video.Video `json:"videoList"`
}

type SeekMessage struct {
	Type string  `json:"type"`
	Seek float64 `json:"seek"`
}
