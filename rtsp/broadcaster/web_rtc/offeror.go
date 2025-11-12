package web_rtc

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"strzcam.com/broadcaster/connection"
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
	pc          *webrtc.PeerConnection
	dataChannel *webrtc.DataChannel
	wsClient    *websocket.Conn
	videoTrack  *VideoTrack
}

func NewOfferor(wsClient *websocket.Conn) (Offeror, error) {
	return Offeror{wsClient: wsClient}, nil
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
		if state == webrtc.PeerConnectionStateFailed {
			connectionState, _ := json.Marshal(map[string]string{"type": "failed"})
			if err := o.wsClient.WriteMessage(websocket.TextMessage, connectionState); err != nil {
				log.Fatal(err)
			}
		}
	})
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
			disconnectedMessage, err := json.Marshal(map[string]string{"type": "disconnected"})
			if err := o.wsClient.WriteMessage(websocket.TextMessage, disconnectedMessage); err != nil {
				log.Fatal(err)
			}

			if err := o.wsClient.WriteMessage(websocket.TextMessage, offerData); err != nil {
				log.Fatal(err)
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

func (o *Offeror) CreateAndSendOffer() {
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

	if err := o.wsClient.WriteMessage(websocket.TextMessage, offerData); err != nil {
		log.Fatal(err)
	}
}
