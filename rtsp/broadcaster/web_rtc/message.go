package web_rtc

type SignalingMessage struct {
	Type string                   `json:"type"`
	Sdp  string                   `json:"sdp,omitempty"`
	Ice  []map[string]interface{} `json:"ice,omitempty"`
}

type DataChannelMessage struct {
	Type      string `json:"type"`
	DateRange string `json:"dateRange,omitempty"`
	VideoName string `json:"videoName,omitempty"`
	Seek      int    `json:"seek,omitempty"`
}
