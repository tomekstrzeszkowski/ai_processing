package web_rtc

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"strzcam.com/broadcaster/connection"
	"strzcam.com/broadcaster/video"
)

var iceServers = []webrtc.ICEServer{
	{
		URLs: []string{
			"stun:stun.l.google.com:19302",
			"stun:stun2.l.google.com:19302",
			"stun:stun3.l.google.com:19302",
			"stun:stun.1und1.de:3478",
			"stun:stun.avigora.com:3478",
			"stun:stun.avigora.fr:3478",
		},
	},
	{
		URLs:       []string{"turn:global.turn.twilio.com:3478?transport=udp"},
		Username:   "dc2d2894d5a9023620c467b0e71cfa6a35457e6679785ed6ae9856fe5bdfa269",
		Credential: "tE2DajzSbc123",
	},
	{
		URLs:       []string{"turn:openrelay.metered.ca:80", "turn:openrelay.metered.ca:443"},
		Username:   "openrelayproject",
		Credential: "openrelayproject",
	},
	{
		URLs:       []string{"turn:openrelay.metered.ca:443?transport=tcp"},
		Username:   "openrelayproject",
		Credential: "openrelayproject",
	},
}

type Offeror struct {
	pc               *webrtc.PeerConnection
	dataChannel      *webrtc.DataChannel
	wsClient         *websocket.Conn
	videoTrack       *VideoTrack
	staticVideoTrack *StaticVideoTrack
	savedVideoPath   string
}

func NewOfferor(wsClient *websocket.Conn, savedVideoPath string) (Offeror, error) {
	return Offeror{wsClient: wsClient, savedVideoPath: savedVideoPath}, nil
}

func (o *Offeror) CreatePeerConnection(videoTrack *VideoTrack) (*webrtc.PeerConnection, error) {
	pc, error := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: iceServers,
	})
	o.pc = pc

	if error != nil {
		log.Fatal(error)
	}
	if videoTrack != nil {
		o.videoTrack = videoTrack
	}
	o.HandlePeerConnection()
	return o.pc, error
}

func (o *Offeror) Close() {
	o.pc.Close()
}

func (o *Offeror) HandlePeerConnection() {
	if o.videoTrack != nil {
		o.HandleVideoTrack()
	}
	dataChannel, err := o.CreateDataChannel()
	if err != nil {
		log.Fatal(err)
	}
	o.dataChannel = dataChannel
	o.pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		fmt.Printf("Connection state: %s\n", state.String())
		switch state {
		case webrtc.PeerConnectionStateFailed, webrtc.PeerConnectionStateDisconnected:
			connectionState, _ := json.Marshal(map[string]string{"type": "failed"})
			if err := o.wsClient.WriteMessage(websocket.TextMessage, connectionState); err != nil {
				log.Fatal(err)
			}
		}
	})
	o.pc.OnNegotiationNeeded(func() {
		state := o.pc.ConnectionState()
		switch state {
		case webrtc.PeerConnectionStateNew, webrtc.PeerConnectionStateConnected, webrtc.PeerConnectionStateConnecting:
			return
		}
		fmt.Printf("Negotiation needed, create and send a new offer connection state: %s", state)
		o.CreateAndSendOffer()
	})
}

func (o *Offeror) SendFlushMessageToSignaling() {
	flushMessage, err := json.Marshal(map[string]string{"type": "flush"})
	if err != nil {
		log.Fatal(err)
	}
	if err := o.wsClient.WriteMessage(websocket.TextMessage, flushMessage); err != nil {
		log.Fatal(err)
	}
}

func (o *Offeror) CreateDataChannel() (*webrtc.DataChannel, error) {
	ordered := false
	maxRetransmits := uint16(0)
	dataChannel, err := o.pc.CreateDataChannel(connection.WebRtcDataChannel, &webrtc.DataChannelInit{
		Ordered:        &ordered,
		MaxRetransmits: &maxRetransmits,
	})
	if err != nil {
		return nil, err
	}
	dataChannel.OnOpen(func() {
		fmt.Println("Data channel opened")
		dataChannel.SendText("hello from server")
		// reset offert so it can't be reused
		o.SendFlushMessageToSignaling()
	})
	dataChannel.OnClose(func() {
		fmt.Println("Data channel closed")
	})
	dataChannel.OnMessage(func(dataChannelMessage webrtc.DataChannelMessage) {
		fmt.Printf("Message from data channel: %s\n", string(dataChannelMessage.Data))
		var message DataChannelMessage
		if err := json.Unmarshal(dataChannelMessage.Data, &message); err != nil {
			log.Fatal("Can not parse message in data channel")
			return
		}
		switch message.Type {
		case "close":
			// is it needed?
			//recreate offer
			offer, err := o.pc.CreateOffer(nil)
			if err != nil {
				log.Fatal(err)
			}

			if err := o.pc.SetLocalDescription(offer); err != nil {
				log.Fatal(err)
			}
			offerData, err := json.Marshal(offer)
			if err != nil {
				log.Fatal(err)
			}
			o.SendFlushMessageToSignaling()
			if err := o.wsClient.WriteMessage(websocket.TextMessage, offerData); err != nil {
				log.Fatal(err)
			}
		case "videoList":
			start, _ := time.Parse("2006-01-02", message.StartDate)
			end, _ := time.Parse("2006-01-02", message.EndDate)
			videoList, _ := video.GetVideoByDateRange(o.savedVideoPath, start, end)
			videoListMessage := VideoListMessage{Type: "videoList", VideoList: videoList}
			if responseMessage, err := json.Marshal(videoListMessage); err == nil {
				log.Printf("Sending %s", responseMessage)
				dataChannel.Send(responseMessage)
			}
		case "video":
			// when adding video re-negotation is needed, exchange offer via dataChannel
			fmt.Printf("Adding video track %s", message.VideoName)
			filePath := filepath.Join(o.savedVideoPath, message.VideoName)
			if o.staticVideoTrack == nil {
				staticVideoTrack, err := NewStaticVideoTrack()
				o.staticVideoTrack = staticVideoTrack
				if err := staticVideoTrack.LoadVideo(filePath); err != nil {
					log.Fatal(err)
					return
				}
				// videoBytes, _ := video.GetVideoByPath(filePath)
				rtpSender, err := o.pc.AddTrack(staticVideoTrack.track)
				staticVideoTrack.Play()
				if err != nil {
					log.Fatal(err)
					return
				}
				go func() {
					rtcpBuf := make([]byte, 1500)
					for {
						if _, _, err := rtpSender.Read(rtcpBuf); err != nil {
							return
						}
					}
				}()
				if err != nil {
					log.Fatal(err)
					return
				}
				offer, err := o.PrepareOffer()
				if err != nil {
					return
				}
				dataChannel.Send(offer)
			} else {
				o.staticVideoTrack.LoadVideo(filePath)
			}
		case "answer":
			answer := webrtc.SessionDescription{
				Type: webrtc.SDPTypeAnswer,
				SDP:  message.Sdp,
			}
			if err := o.pc.SetRemoteDescription(answer); err != nil {
				log.Printf("Error setting remote description: %v", err)
			}
		}

	})
	return dataChannel, nil
}

func (o *Offeror) HandleVideoTrack() error {
	rtpSender, err := o.pc.AddTrack(o.videoTrack.track)
	if err != nil {
		log.Fatal(err)
		return err
	}
	// prevent video clogging and flush the buffer
	go func() {
		rtcpBuf := make([]byte, 1500)
		for {
			if _, _, err := rtpSender.Read(rtcpBuf); err != nil {
				return
			}
		}
	}()
	return nil
}
func (o *Offeror) PrepareOffer() ([]byte, error) {
	offer, err := o.pc.CreateOffer(nil)
	if err != nil {
		log.Fatal(err)
	}

	if err := o.pc.SetLocalDescription(offer); err != nil {
		log.Fatal(err)
	}
	offerData, err := json.Marshal(offer)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	return offerData, nil
}
func (o *Offeror) CreateAndSendOffer() {
	offerData, err := o.PrepareOffer()
	if err != nil {
		log.Fatal(err)
		return
	}

	if err := o.wsClient.WriteMessage(websocket.TextMessage, offerData); err != nil {
		log.Fatal(err)
	}
}
