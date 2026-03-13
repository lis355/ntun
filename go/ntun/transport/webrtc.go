package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/pion/webrtc/v3"
)

// TODO optimize json.Marshal(w.peer.LocalDescription()) do a  Trickle ICE send preferred candidate from sender

const DataChannelName = "DC"

type TurnServerInfo struct {
	URL      string
	Username string
	Password string
}

type WebRTCTransport struct {
	iceServerInfo *TurnServerInfo
	peer          *webrtc.PeerConnection
	dc            *webrtc.DataChannel
}

func NewWebRTCTransport(iceServerInfo *TurnServerInfo) *WebRTCTransport {
	return &WebRTCTransport{
		iceServerInfo: iceServerInfo,
	}
}

func (w *WebRTCTransport) Transport() error {

	return nil
}

func (w *WebRTCTransport) CreatePeer() error {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{
			URLs:           []string{w.iceServerInfo.URL},
			Username:       w.iceServerInfo.Username,
			Credential:     w.iceServerInfo.Password,
			CredentialType: webrtc.ICECredentialTypePassword,
		}},
	}

	peer, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return err
	}

	w.peer = peer

	w.peer.OnICEConnectionStateChange(func(s webrtc.ICEConnectionState) {
		slog.Debug(fmt.Sprintf("ICEConnectionState %s", s))

		if s == webrtc.ICEConnectionStateConnected {
			//
		}
	})

	w.peer.OnDataChannel(w.handleDataChannel)

	return nil
}

func (w *WebRTCTransport) CreateOffer() (sessionBuf []byte, err error) {
	err = w.CreatePeer()
	if err != nil {
		return sessionBuf, err
	}

	dc, err := w.peer.CreateDataChannel(DataChannelName, nil)
	if err != nil {
		return sessionBuf, err
	}

	w.handleDataChannel(dc)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	w.peer.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil &&
			c.Typ == webrtc.ICECandidateTypeRelay {
			slog.Debug(fmt.Sprintf("ICECandidate %s", c.String()))
		}
	})

	w.peer.OnICEGatheringStateChange(func(s webrtc.ICEGathererState) {
		slog.Debug(fmt.Sprintf("ICEGathererState %s", s))

		if s == webrtc.ICEGathererStateComplete {
			cancel()
		}
	})

	offer, err := w.peer.CreateOffer(nil)
	w.peer.SetLocalDescription(offer)

	<-ctx.Done()

	w.peer.OnICECandidate(nil)
	w.peer.OnICEGatheringStateChange(nil)

	err = ctx.Err()
	if err == context.DeadlineExceeded {
		return sessionBuf, err
	}

	sessionBuf, err = json.Marshal(w.peer.LocalDescription())
	if err != nil {
		return sessionBuf, err
	}

	return sessionBuf, err
}

func (w *WebRTCTransport) CreateAnswer(offerBuf []byte) (sessionBuf []byte, err error) {
	var offer webrtc.SessionDescription
	if err := json.Unmarshal(offerBuf, &offer); err != nil {
		return sessionBuf, fmt.Errorf("failed to unmarshal offer: %w", err)
	}

	err = w.CreatePeer()
	if err != nil {
		return sessionBuf, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	w.peer.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil &&
			c.Typ == webrtc.ICECandidateTypeRelay {
			slog.Debug(fmt.Sprintf("ICECandidate %s", c.String()))
		}
	})

	w.peer.OnICEGatheringStateChange(func(s webrtc.ICEGathererState) {
		slog.Debug(fmt.Sprintf("ICEGathererState %s", s))

		if s == webrtc.ICEGathererStateComplete {
			cancel()
		}
	})

	if err := w.peer.SetRemoteDescription(offer); err != nil {
		return sessionBuf, err
	}

	answer, err := w.peer.CreateAnswer(nil)
	w.peer.SetLocalDescription(answer)

	<-ctx.Done()

	w.peer.OnICECandidate(nil)
	w.peer.OnICEGatheringStateChange(nil)

	sessionBuf, err = json.Marshal(w.peer.LocalDescription())
	if err != nil {
		return sessionBuf, err
	}

	return sessionBuf, err
}

func (w *WebRTCTransport) SetAnswer(answerBuf []byte) error {
	var answer webrtc.SessionDescription
	if err := json.Unmarshal(answerBuf, &answer); err != nil {
		return fmt.Errorf("failed to unmarshal answer: %w", err)
	}

	if err := w.peer.SetRemoteDescription(answer); err != nil {
		return err
	}

	return nil
}

func (w *WebRTCTransport) handleDataChannel(dc *webrtc.DataChannel) {
	w.dc = dc

	w.dc.OnOpen(w.handleDataChannelOpen)
	w.dc.OnMessage(w.handleDataChannelOnMessage)
}

func (w *WebRTCTransport) handleDataChannelOpen() {
	fmt.Println("Канал открыт")

	w.dc.Send([]byte("Привет от пира1!"))
}

func (w *WebRTCTransport) handleDataChannelOnMessage(msg webrtc.DataChannelMessage) {
	fmt.Printf("Пир1 получил: %s\n", string(msg.Data))
}
