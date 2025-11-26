package web_rtc

import "strzcam.com/broadcaster/video"

// signaling message used by websocket
type SignalingMessage struct {
	Type string                   `json:"type"`
	Sdp  string                   `json:"sdp,omitempty"`
	Ice  []map[string]interface{} `json:"ice,omitempty"`
}

// data channel incomming messages
type DataChannelMessage struct {
	Type      string  `json:"type"`
	Sdp       string  `json:"sdp,omitempty"`
	StartDate string  `json:"startDate,omitempty"`
	EndDate   string  `json:"endDate,omitempty"`
	VideoName string  `json:"videoName,omitempty"`
	Seek      float64 `json:"seek,omitempty"`
	IsForward bool    `json:"isForward,omitempty"`
}

// data channel outgouing messages
type VideoListMessage struct {
	Type      string        `json:"type"`
	VideoList []video.Video `json:"videoList"`
}
type StatusSeekMessage struct {
	Type string  `json:"type"`
	Seek float64 `json:"seek"`
}
type StatusIsPlayingMessage struct {
	Type      string `json:"type"`
	IsPlaying bool   `json:"isPlaying"`
}
type StatusIsLoopMessage struct {
	Type   string `json:"type"`
	IsLoop bool   `json:"isLoop"`
}
type StatusLoadVideoMessage struct {
	Type      string  `json:"type"`
	IsPlaying bool    `json:"isPlaying"`
	IsLoop    bool    `json:"isLoop"`
	Duration  float64 `json:"duration,omitempty"`
}
