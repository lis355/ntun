package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/pion/webrtc/v3"
)

const (
	DataChannelName     = "DC"
	ICEGatheringTimeout = 30 * time.Second
)

type WebRTCTransport struct {
	iceServer   *webrtc.ICEServer
	peer        *webrtc.PeerConnection
	dc          *webrtc.DataChannel
	transportCh chan struct{}
	dcOpenCh    chan struct{}
	dcCloseCh   chan struct{}
	dcWriteCh   chan []byte
	wconn       *webRTCConn
}

func NewWebRTCTransport() *WebRTCTransport {
	return &WebRTCTransport{
		transportCh: make(chan struct{}),
		dcOpenCh:    make(chan struct{}),
		dcCloseCh:   make(chan struct{}),
		dcWriteCh:   make(chan []byte),
	}
}

func (w *WebRTCTransport) Transport() (net.Conn, error) {
	<-w.transportCh
	return w.wconn, nil
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
		slog.Debug(fmt.Sprintf("[WebRTCTransport]: %p ICEConnectionState %s", w, s))
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

	ctx, cancel := context.WithTimeout(context.Background(), ICEGatheringTimeout)

	w.peer.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil &&
			c.Typ == webrtc.ICECandidateTypeRelay {
			slog.Debug(fmt.Sprintf("[WebRTCTransport]: %p ICECandidate %s", w, c.String()))
			if candidate == nil {
				candidateInit := c.ToJSON()
				candidate = &candidateInit
				cancel()
			}
		}
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

	if err := w.peer.SetRemoteDescription(*offerInfo.Session); err != nil {
		return sessionBuf, err
	}

	w.peer.AddICECandidate(*offerInfo.Candidate)

	answer, err := w.peer.CreateAnswer(nil)
	w.peer.SetLocalDescription(answer)

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

func (w *WebRTCTransport) Close() {
	w.peer.Close()
}

func (w *WebRTCTransport) handleDataChannel(dc *webrtc.DataChannel) {
	w.dc = dc
	w.wconn = &webRTCConn{
		webrtc:  w,
		readCh:  make(chan []byte),
		readBuf: make([]byte, 0),
	}

	w.dc.OnOpen(w.handleDataChannelOnOpen)
	w.dc.OnClose(w.handleDataChannelOnClose)
	w.dc.OnMessage(w.handleDataChannelOnMessage)

	go func() {
		for {
			select {
			case buf, ok := <-w.dcWriteCh:
				if !ok {
					w.dcWriteCh = nil
					return
				}

				if err := w.dc.Send(buf); err != nil {
					w.Close()

					return
				}
			case _, ok := <-w.dcOpenCh:
				if !ok {
					w.dcOpenCh = nil
					return
				}
				w.dcWriteCh = make(chan []byte)
				w.transportCh <- struct{}{}
			case _, ok := <-w.dcCloseCh:
				if !ok {
					w.dcCloseCh = nil
					return
				}

				w.wconn.lock.Lock()
				close(w.dcWriteCh)
				w.dcWriteCh = nil

				close(w.wconn.readCh)
				w.wconn.readCh = nil

				w.wconn.closed = true
				w.wconn.lock.Unlock()

				w.wconn = nil

				w.dc.OnOpen(nil)
				w.dc.OnClose(nil)
				w.dc.OnMessage(nil)

				w.dc = nil

				return
			}
		}
	}()
}

func (w *WebRTCTransport) handleDataChannelOnOpen() {
	w.dcOpenCh <- struct{}{}
}

func (w *WebRTCTransport) handleDataChannelOnClose() {
	w.dcCloseCh <- struct{}{}
}

func (w *WebRTCTransport) handleDataChannelOnMessage(msg webrtc.DataChannelMessage) {
	w.wconn.lock.Lock()
	if w.wconn.closed {
		w.wconn.lock.Unlock()
		return
	}
	w.wconn.lock.Unlock()

	w.wconn.readCh <- msg.Data
}

type webRTCConn struct {
	lock    sync.Mutex
	closed  bool
	webrtc  *WebRTCTransport
	readCh  chan []byte
	readBuf []byte
}

func (w *webRTCConn) Read(b []byte) (n int, err error) {
	if len(b) == 0 {
		return 0, nil
	}

	w.lock.Lock()
	if w.closed {
		w.lock.Unlock()

		return 0, io.ErrClosedPipe
	}

	if len(w.readBuf) > 0 {
		n = copy(b, w.readBuf)
		w.readBuf = w.readBuf[n:]
		w.lock.Unlock()

		return n, nil
	}
	w.lock.Unlock()

	buf, ok := <-w.readCh
	if !ok {
		return 0, io.EOF
	}

	w.lock.Lock()
	w.readBuf = append(w.readBuf, buf...)

	n = copy(b, w.readBuf)
	w.readBuf = w.readBuf[n:]
	w.lock.Unlock()

	return n, nil
}

func (w *webRTCConn) Write(b []byte) (n int, err error) {
	w.lock.Lock()
	if w.closed {
		w.lock.Unlock()

		return 0, io.ErrClosedPipe
	}
	w.lock.Unlock()

	w.webrtc.dcWriteCh <- b

	return len(b), nil
}

func (w *webRTCConn) Close() error {
	w.lock.Lock()
	if w.closed {
		w.lock.Unlock()
		return nil
	}

	w.closed = true
	w.lock.Unlock()

	w.webrtc.Close()

	return nil
}

func (w *webRTCConn) LocalAddr() net.Addr {
	return &net.UnixAddr{Net: "webrtc", Name: "webrtc"}
}

func (w *webRTCConn) RemoteAddr() net.Addr {
	return &net.UnixAddr{Net: "webrtc", Name: "webrtc"}
}

func (w *webRTCConn) SetDeadline(t time.Time) error {
	return nil
}

func (w *webRTCConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (w *webRTCConn) SetWriteDeadline(t time.Time) error {
	return nil
}
