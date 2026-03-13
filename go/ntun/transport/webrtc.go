package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/pion/webrtc/v3"
)

const DataChannelName = "DC"

type WebRTCTransport struct {
	iceServer *webrtc.ICEServer
	peer      *webrtc.PeerConnection
	dc        *webrtc.DataChannel
}

func NewWebRTCTransport() *WebRTCTransport {
	return &WebRTCTransport{}
}

func (w *WebRTCTransport) Transport() error {

	return nil
}

func (w *WebRTCTransport) CreatePeer(iceServer *webrtc.ICEServer) error {
	w.iceServer = iceServer

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{*w.iceServer},
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

type OfferInfo struct {
	Session   *webrtc.SessionDescription
	IceServer *webrtc.ICEServer
	Candidate *webrtc.ICECandidateInit
}

func (w *WebRTCTransport) CreateOffer(iceServer *webrtc.ICEServer) ([]byte, error) {
	err := w.CreatePeer(iceServer)
	if err != nil {
		return nil, err
	}

	dc, err := w.peer.CreateDataChannel(DataChannelName, nil)
	if err != nil {
		return nil, err
	}

	w.handleDataChannel(dc)

	var candidate *webrtc.ICECandidateInit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	w.peer.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil &&
			c.Typ == webrtc.ICECandidateTypeRelay {
			slog.Debug(fmt.Sprintf("ICECandidate %s", c.String()))
			if candidate == nil {
				candidateInit := c.ToJSON()
				candidate = &candidateInit
				cancel()
			}
		}
	})

	w.peer.OnICEGatheringStateChange(func(s webrtc.ICEGathererState) {
		slog.Debug(fmt.Sprintf("ICEGathererState %s", s))
	})

	offer, err := w.peer.CreateOffer(nil)
	w.peer.SetLocalDescription(offer)

	<-ctx.Done()

	w.peer.OnICECandidate(nil)
	w.peer.OnICEGatheringStateChange(nil)

	err = ctx.Err()
	if err == context.DeadlineExceeded {
		return nil, err
	}

	offerInfo := &OfferInfo{Session: w.peer.LocalDescription(), IceServer: w.iceServer, Candidate: candidate}

	infoBuf, err := json.Marshal(offerInfo)
	if err != nil {
		return nil, err
	}

	return infoBuf, err
}

func (w *WebRTCTransport) CreateAnswer(infoBuf []byte) (sessionBuf []byte, err error) {
	var offerInfo OfferInfo
	if err := json.Unmarshal(infoBuf, &offerInfo); err != nil {
		return sessionBuf, err
	}

	err = w.CreatePeer(offerInfo.IceServer)
	if err != nil {
		return sessionBuf, err
	}

	// ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	w.peer.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil &&
			c.Typ == webrtc.ICECandidateTypeRelay {
			slog.Debug(fmt.Sprintf("ICECandidate %s", c.String()))
		}
	})

	// w.peer.OnICEGatheringStateChange(func(s webrtc.ICEGathererState) {
	// 	slog.Debug(fmt.Sprintf("ICEGathererState %s", s))

	// 	if s == webrtc.ICEGathererStateComplete {
	// 		cancel()
	// 	}
	// })

	if err := w.peer.SetRemoteDescription(*offerInfo.Session); err != nil {
		return sessionBuf, err
	}

	w.peer.AddICECandidate(*offerInfo.Candidate)

	answer, err := w.peer.CreateAnswer(nil)
	w.peer.SetLocalDescription(answer)

	// <-ctx.Done()

	w.peer.OnICECandidate(nil)
	// w.peer.OnICEGatheringStateChange(nil)

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

	w.dc.OnOpen(w.handleDataChannelOnOpen)
	w.dc.OnClose(w.handleDataChannelOnClose)
	w.dc.OnMessage(w.handleDataChannelOnMessage)
}

func (w *WebRTCTransport) handleDataChannelOnOpen() {
	fmt.Println("Канал открыт")

	w.dc.Send([]byte("Привет от пира1!"))
}

func (w *WebRTCTransport) handleDataChannelOnClose() {
	fmt.Println("Канал открыт")

	w.dc.Send([]byte("Привет от пира1!"))
}

func (w *WebRTCTransport) handleDataChannelOnMessage(msg webrtc.DataChannelMessage) {
	fmt.Printf("Пир1 получил: %s\n", string(msg.Data))
}
