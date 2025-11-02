package web_rtc

type SignalingMessage struct {
	Type string                   `json:"type"`
	Sdp  string                   `json:"sdp,omitempty"`
	Ice  []map[string]interface{} `json:"ice,omitempty"`
}
