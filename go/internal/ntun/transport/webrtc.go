package transport

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"ntun/internal/log"
	"ntun/internal/utils"
	"os"
	"sync"
	"time"

	"github.com/pion/logging"
	"github.com/pion/webrtc/v3"
)

const (
	webRtcLogs = false

	iceGatheringTimeout = 60 * time.Second
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

	// ConnectCh    chan struct{}
	DisconnectCh chan struct{}
}

func NewWebRTCTransport() *WebRTCTransport {
	return &WebRTCTransport{
		transportCh: make(chan struct{}),
		dcOpenCh:    make(chan struct{}),
		dcCloseCh:   make(chan struct{}),
		dcWriteCh:   make(chan []byte),
		// ConnectCh:    make(chan struct{}),
		DisconnectCh: make(chan struct{}),
	}
}

func (w *WebRTCTransport) Transport() (net.Conn, error) {
	if _, ok := <-w.transportCh; !ok {
		return nil, errors.New("transport closed")
	}

	return w.wconn, nil
}

func (w *WebRTCTransport) createPeer(iceServer *webrtc.ICEServer) error {
	w.iceServer = iceServer

	settingEngine := webrtc.SettingEngine{}

	if webRtcLogs {
		loggerFactory := logging.NewDefaultLoggerFactory()
		loggerFactory.DefaultLogLevel = logging.LogLevelTrace
		loggerFactory.Writer = os.Stderr
		settingEngine.LoggerFactory = loggerFactory
	}

	config := webrtc.Configuration{
		ICEServers:         []webrtc.ICEServer{*w.iceServer},
		ICETransportPolicy: webrtc.ICETransportPolicyRelay,
	}

	api := webrtc.NewAPI(webrtc.WithSettingEngine(settingEngine))

	peer, err := api.NewPeerConnection(config)
	if err != nil {
		return err
	}

	w.peer = peer

	w.peer.OnICEConnectionStateChange(func(s webrtc.ICEConnectionState) {
		slog.Debug(fmt.Sprintf("%s: ICEConnectionState %s", log.ObjName(w), s))

		if s == webrtc.ICEConnectionStateConnected ||
			s == webrtc.ICEConnectionStateCompleted {

			stats := w.peer.GetStats()

			var pair webrtc.ICECandidatePairStats
			candidates := map[string]webrtc.ICECandidateStats{}

			for _, s := range stats {
				switch v := s.(type) {
				case webrtc.ICECandidateStats:
					candidates[v.ID] = v
				case webrtc.ICECandidatePairStats:
					if v.State == "succeeded" &&
						v.Nominated {
						pair = v
					}
				}
			}

			local := candidates[pair.LocalCandidateID]
			remote := candidates[pair.RemoteCandidateID]

			iceCandidateTypeToStr := func(candidateType webrtc.ICECandidateType) string {
				switch candidateType {
				case webrtc.ICECandidateTypeHost:
					return "host"
				case webrtc.ICECandidateTypeSrflx:
					return "srflx"
				case webrtc.ICECandidateTypePrflx:
					return "prflx"
				case webrtc.ICECandidateTypeRelay:
					return "relay"
				default:
					return "unknown"
				}
			}

			slog.Debug(fmt.Sprintf("%s: connected %s:%s:%d:%s -> %s:%s:%d:%s", log.ObjName(w), local.RelayProtocol, local.IP, local.Port, iceCandidateTypeToStr(local.CandidateType), remote.RelayProtocol, remote.IP, remote.Port, iceCandidateTypeToStr(remote.CandidateType)))
		}
	})

	w.peer.OnDataChannel(w.handleDataChannel)

	return nil
}

type sessionCreator func() (webrtc.SessionDescription, error)

func (w *WebRTCTransport) createSession(sessionCreator sessionCreator) error {
	w.peer.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}

		slog.Debug(fmt.Sprintf("%s: ICECandidate %s", log.ObjName(w), c.String()))
	})

	session, err := sessionCreator()
	if err != nil {
		return err
	}

	if err := w.peer.SetLocalDescription(session); err != nil {
		return err
	}

	gatherComplete := webrtc.GatheringCompletePromise(w.peer)

	select {
	case <-gatherComplete:
	case <-time.After(iceGatheringTimeout):
		return errors.New("ice gathering timeout")
	}

	w.peer.OnICECandidate(nil)
	w.peer.OnICEGatheringStateChange(nil)

	return nil
}

type OfferInfo struct {
	Session   *webrtc.SessionDescription
	IceServer *webrtc.ICEServer
}

func (w *WebRTCTransport) CreateOffer(iceServer *webrtc.ICEServer) ([]byte, error) {
	if err := w.createPeer(iceServer); err != nil {
		return nil, err
	}

	dataChannelName := utils.RandShortString()

	dc, err := w.peer.CreateDataChannel(dataChannelName, nil)
	if err != nil {
		return nil, err
	}

	w.handleDataChannel(dc)

	if err := w.createSession(func() (webrtc.SessionDescription, error) { return w.peer.CreateOffer(nil) }); err != nil {
		return nil, err
	}

	offerInfo := &OfferInfo{Session: w.peer.LocalDescription(), IceServer: w.iceServer}

	infoBuf, err := json.Marshal(offerInfo)
	if err != nil {
		return nil, err
	}

	return infoBuf, err
}

type AnswerInfo struct {
	Session *webrtc.SessionDescription
}

func (w *WebRTCTransport) CreateAnswer(infoBuf []byte) ([]byte, error) {
	var offerInfo OfferInfo
	if err := json.Unmarshal(infoBuf, &offerInfo); err != nil {
		return nil, err
	}

	if err := w.createPeer(offerInfo.IceServer); err != nil {
		return nil, err
	}

	if err := w.peer.SetRemoteDescription(*offerInfo.Session); err != nil {
		return nil, err
	}

	if err := w.createSession(func() (webrtc.SessionDescription, error) { return w.peer.CreateAnswer(nil) }); err != nil {
		return nil, err
	}

	answerInfo := &AnswerInfo{Session: w.peer.LocalDescription()}

	sessionBuf, err := json.Marshal(answerInfo)
	if err != nil {
		return nil, err
	}

	return sessionBuf, err
}

func (w *WebRTCTransport) SetAnswer(infoBuf []byte) error {
	var answerInfo AnswerInfo
	if err := json.Unmarshal(infoBuf, &answerInfo); err != nil {
		return err
	}

	if err := w.peer.SetRemoteDescription(*answerInfo.Session); err != nil {
		return err
	}

	return nil
}

func (w *WebRTCTransport) Close() error {
	if w.peer == nil {
		return nil
	}

	return w.peer.Close()
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
				// w.ConnectCh <- struct{}{}
				w.transportCh <- struct{}{}

				slog.Debug(fmt.Sprintf("%s: connected via %s", log.ObjName(w), w.iceServer.URLs[0]))
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

				w.DisconnectCh <- struct{}{}

				slog.Debug(fmt.Sprintf("%s: disconnected", log.ObjName(w)))

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
