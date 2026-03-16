package yandex

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/quotedprintable"
	"net"
	"ntun/internal/app"
	"ntun/internal/cfg"
	"ntun/internal/cipher"
	"ntun/internal/connections"
	"ntun/internal/log"
	"ntun/internal/messages/yandex"
	"ntun/internal/ntun/node"
	"ntun/internal/ntun/transport"
	"ntun/internal/utils"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/google/uuid"
	"github.com/pion/webrtc/v3"
	"go.yaml.in/yaml/v3"
)

// Логика "сигнального сервера" для WebRTC соединения выполняется посредством почты
// При старте удаляются все сообщения, которые старше десяти минут
// Также удаляются все сообщения, которые не расшифровались (ошибка)
// Сообщения удаляет тот, для кого они предназначались
// Тайм-аут ожидания offer/answer - 3 минуты

const (
	mailSubjectPrefix = "TUN_MSG_"
	oldMailAge        = 10 * time.Minute

	signalingMsgTypeOffer  = "offer"
	signalingMsgTypeAnswer = "answer"

	callingStateNone      = "none"
	callingStateOffer     = "offer"
	callingStateAnswer    = "answer"
	callingStateWaiting   = "waiting"
	callingStateConnected = "connected"
)

type signalingMsg struct {
	Version string
	Sender  uuid.UUID // TODO Id type
	Typ     string
	Buf     []byte
}

func GZipCipherBase64Encode(cipher *cipher.CipherAesGcm, buf []byte) ([]byte, error) {
	compressedBuf, err := utils.GZipEncode(buf)
	if err != nil {
		return nil, err
	}

	encryptedBuf, err := cipher.Encrypt(compressedBuf)
	if err != nil {
		return nil, err
	}

	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(encryptedBuf)))

	base64.StdEncoding.Encode(encoded, encryptedBuf)

	return encoded, nil
}

func GZipCipherBase64Decode(cipher *cipher.CipherAesGcm, encoded []byte) ([]byte, error) {
	decodedBuf := make([]byte, base64.StdEncoding.DecodedLen(len(encoded)))
	n, err := base64.StdEncoding.Decode(decodedBuf, encoded)
	if err != nil {
		return nil, err
	}

	encryptedBuf := decodedBuf[:n]

	compressedBuf, err := cipher.Decrypt(encryptedBuf)
	if err != nil {
		return nil, err
	}

	buf, err := utils.GZipDecode(compressedBuf)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

type YandexMailSignaling struct {
	yandex.YandexMail
	cipher *cipher.CipherAesGcm
	node   *node.Node
	mailCh chan *signalingMsg
}

func NewYandexMailSignaling(transportCfg *cfg.YandexWebRTCTransport, node *node.Node) (*YandexMailSignaling, error) {
	cipher, err := cipher.NewCipherAesGcm([]byte(node.Config.CipherKey))
	if err != nil {
		return nil, err
	}

	s := &YandexMailSignaling{
		cipher: cipher,
		node:   node,
		mailCh: make(chan *signalingMsg),
	}

	s.YandexMail = *yandex.NewYandexMail(transportCfg.MailUser, transportCfg.MailPass, s.processInbox)

	return s, nil
}

func (s *YandexMailSignaling) processInbox(start bool) error {
	searchCriteria := &imap.SearchCriteria{
		Header: []imap.SearchCriteriaHeaderField{
			{Key: "Subject", Value: mailSubjectPrefix},
		},
	}

	searchData, err := s.Client.Search(searchCriteria, nil).Wait()
	if err != nil {
		return err
	}

	if len(searchData.AllSeqNums()) == 0 {
		return nil
	}

	seqSet := imap.SeqSetNum(searchData.AllSeqNums()...)

	fetchOptions := &imap.FetchOptions{
		UID:          true,
		Envelope:     true,
		InternalDate: true,
		BodySection: []*imap.FetchItemBodySection{
			{Specifier: imap.PartSpecifierText},
		},
	}

	fetchCmd := s.Client.Fetch(seqSet, fetchOptions)

	mailUidsToDelete := make([]imap.UID, 0)

	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		var uid imap.UID
		var subj string
		var body []byte
		var date time.Time

		for {
			item := msg.Next()
			if item == nil {
				break
			}

			switch item := item.(type) {
			case imapclient.FetchItemDataUID:
				uid = item.UID
			case imapclient.FetchItemDataEnvelope:
				subj = item.Envelope.Subject
			case imapclient.FetchItemDataInternalDate:
				date = item.Time
			case imapclient.FetchItemDataBodySection:
				body, err = io.ReadAll(quotedprintable.NewReader(item.Literal))
				if err != nil {
					slog.Debug(fmt.Sprintf("%s: error reading msg body %v", log.ObjName(s), err))

					continue
				}
			}
		}

		delete := s.processMail(start, subj, date, []byte(body))
		if delete {
			// slog.Debug(fmt.Sprintf("%s: recieved mail %s %s %d bytes, deleting", log.ObjName(s), subj, date.Format(time.RFC3339), len(body)))
			mailUidsToDelete = append(mailUidsToDelete, uid)
		}
	}

	if len(mailUidsToDelete) > 0 {
		if err := s.DeleteMails(mailUidsToDelete); err != nil {
			slog.Error(fmt.Sprintf("%s: error deleting mails %v", log.ObjName(s), err))

			// dont break cycle
		}
	}

	if err := fetchCmd.Close(); err != nil {
		return err
	}

	return nil
}

func (s *YandexMailSignaling) processMail(start bool, subject string, date time.Time, buf []byte) bool {
	slog.Debug(fmt.Sprintf("%s: recieved mail %s %s %d bytes", log.ObjName(s), subject, date.Format(time.RFC3339), len(buf)))

	delete := false

	// При старте удаляются все сообщения, которые старше десяти минут
	// Также удаляются все сообщения, которые не расшифровались (ошибка)
	if start &&
		time.Since(date) > oldMailAge {
		delete = true
		slog.Debug(fmt.Sprintf("%s: recieved mail is old %s %s %d bytes", log.ObjName(s), subject, date.Format(time.RFC3339), len(buf)))
	}

	var decodedBuf []byte
	if !delete {
		buf, decodingErr := GZipCipherBase64Decode(s.cipher, buf)
		if decodingErr != nil {
			delete = true
			slog.Debug(fmt.Sprintf("%s: recieved mail error decoding %s %s %d bytes", log.ObjName(s), subject, date.Format(time.RFC3339), len(buf)))
		}

		decodedBuf = buf
	}

	if !delete {
		var msg signalingMsg
		if err := json.Unmarshal(decodedBuf, &msg); err != nil {
			delete = true
			slog.Debug(fmt.Sprintf("%s: recieved mail error decoding %s %s %d bytes", log.ObjName(s), subject, date.Format(time.RFC3339), len(buf)))
		} else if msg.Version != app.Version {
			delete = true
			slog.Debug(fmt.Sprintf("%s: recieved mail bad version %s app version %s", log.ObjName(s), msg.Version, app.Version))
		} else {
			// Сообщения удаляет тот, для кого они предназначались
			if msg.Sender == s.node.Config.Id {
				// пропускаем, это не нам сообщение (оно нами сделанное)
				delete = false
			} else if !s.node.HasAllowedToConnectNodeId(msg.Sender) {
				delete = true
				slog.Debug(fmt.Sprintf("%s: recieved mail from unknown node", log.ObjName(s)))
			} else {
				delete = true
				// slog.Debug(fmt.Sprintf("%s: recieved mail %s %s", log.ObjName(s), subject, date.Format(time.RFC3339)))

				s.mailCh <- &msg
			}
		}
	}

	return delete
}

func (s *YandexMailSignaling) sendMessage(subject string, typ string, data []byte) error {
	msg := &signalingMsg{
		Version: app.Version,
		Sender:  s.node.Config.Id,
		Typ:     typ,
		Buf:     data,
	}

	buf, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	body, err := GZipCipherBase64Encode(s.cipher, buf)
	if err != nil {
		return err
	}

	return s.SendMail(mailSubjectPrefix+subject, string(body))
}

func (s *YandexMailSignaling) recieveMessage() *signalingMsg {
	msg, ok := <-s.mailCh
	if !ok {
		return nil
	}

	return msg
}

type YandexWebRTCTransport struct {
	cfg              *cfg.YandexWebRTCTransport
	webRTCTransport  *transport.WebRTCTransport
	node             *node.Node
	signaling        *YandexMailSignaling
	callingState     string
	callingStartTime time.Time
}

func NewYandexWebRTCTransport(cfg *cfg.YandexWebRTCTransport, node *node.Node) (*YandexWebRTCTransport, error) {
	signaling, err := NewYandexMailSignaling(cfg, node)
	if err != nil {
		return nil, err
	}

	return &YandexWebRTCTransport{
		cfg:             cfg,
		webRTCTransport: transport.NewWebRTCTransport(),
		node:            node,
		signaling:       signaling,
		callingState:    callingStateNone,
	}, nil
}

func (y *YandexWebRTCTransport) Transport() (net.Conn, error) {
	conn, err := y.webRTCTransport.Transport()
	if err != nil {
		return nil, err
	}

	// conn = dev.NewSnifferHexDumpDebugConn(conn, fmt.Sprintf("[%s]", log.ObjName(y)), false, true, false, false)
	conn = connections.NewBufferedConn(conn, 4096, 10*time.Millisecond)

	y.callingState = callingStateConnected

	slog.Debug(fmt.Sprintf("%s: callingState %s %s", log.ObjName(y), y.callingState, y.callingStartTime.Format(time.RFC3339)))

	return conn, err
}

func (c *YandexWebRTCTransport) RateLimit() *cfg.Rate {
	return &c.cfg.RateLimit
}

func (y *YandexWebRTCTransport) Listen() error {
	if err := y.signaling.Listen(); err != nil {
		return err
	}

	go y.handleMessages()

	// DEBUG
	go func() {
		// NOTE пока что клиента (тот, что начинает webrtc) определяем как того, у кого есть Input
		if y.node.Input != nil {
			if err := y.callingCreateOffer(); err != nil {
				y.abort(err)

				return
			}
		}
	}()

	return nil
}

func (y *YandexWebRTCTransport) Close() error {
	if err := y.signaling.Listen(); err != nil {
		return err
	}

	if err := y.webRTCTransport.Close(); err != nil {
		return err
	}

	return nil
}

func (y *YandexWebRTCTransport) abort(err error) {
	slog.Debug(fmt.Sprintf("%s: aborting %v", log.ObjName(y), err))

	// TODO
}

func (y *YandexWebRTCTransport) handleMessages() {
	for {
		msg := y.signaling.recieveMessage()
		if msg == nil {
			y.abort(nil)
			return
		}

		// NOTE обработка параллельно чтобы не мешать signaling разгребать почту
		go y.handleMessage(msg)
	}
}

func (y *YandexWebRTCTransport) handleMessage(msg *signalingMsg) {
	slog.Debug(fmt.Sprintf("%s: handleMessage msg.Typ=%s y.callingState=%s", log.ObjName(y), msg.Typ, y.callingState))

	switch msg.Typ {
	case signalingMsgTypeOffer:
		switch y.callingState {
		case callingStateNone:
			if err := y.callingCreateAnswer(msg.Buf); err != nil {
				y.abort(err)

				return
			}
			return
		case callingStateOffer: // bad logic
		case callingStateAnswer:
			// retry
			if err := y.callingCreateAnswer(msg.Buf); err != nil {
				y.abort(err)

				return
			}
			return
		case callingStateWaiting: // bad logic
		case callingStateConnected: // bad logic
		}
	case signalingMsgTypeAnswer:
		switch y.callingState {
		case callingStateNone: // bad logic
		case callingStateOffer:
			if err := y.callingResumeOffer(msg.Buf); err != nil {
				y.abort(err)

				return
			}
			return
		case callingStateAnswer: // bad logic
		case callingStateWaiting: // bad logic
		case callingStateConnected: // bad logic
		}
	}

	slog.Error(fmt.Sprintf("%s: handleMessage bad logic msg.Typ=%s y.callingState=%s", log.ObjName(y), msg.Typ, y.callingState))
}

func (y *YandexWebRTCTransport) callingCreateOffer() error {
	iceServer, err := y.getIceServer()
	if err != nil {
		slog.Error(fmt.Sprintf("%s: failed to get iceServer %v", log.ObjName(y), err))

		return err
	}

	buf, err := y.webRTCTransport.CreateOffer(iceServer)
	if err != nil {
		slog.Error(fmt.Sprintf("%s: failed to create offer %v", log.ObjName(y), err))

		return err
	}

	err = y.signaling.sendMessage(uuid.New().String(), signalingMsgTypeOffer, buf)
	if err != nil {
		slog.Error(fmt.Sprintf("%s: failed to send signal message %v", log.ObjName(y), err))

		return err
	}

	y.callingState = callingStateOffer
	y.callingStartTime = time.Now()

	slog.Debug(fmt.Sprintf("%s: callingState %s %s", log.ObjName(y), y.callingState, y.callingStartTime.Format(time.RFC3339)))

	return nil
}

func (y *YandexWebRTCTransport) callingCreateAnswer(buf []byte) error {
	buf, err := y.webRTCTransport.CreateAnswer(buf)
	if err != nil {
		slog.Error(fmt.Sprintf("%s: failed to create answer %v", log.ObjName(y), err))

		return err
	}

	err = y.signaling.sendMessage(uuid.New().String(), signalingMsgTypeAnswer, buf)
	if err != nil {
		slog.Error(fmt.Sprintf("%s: failed to send signal message %v", log.ObjName(y), err))

		return err
	}

	y.callingState = callingStateAnswer
	y.callingStartTime = time.Now()

	slog.Debug(fmt.Sprintf("%s: callingState %s %s", log.ObjName(y), y.callingState, y.callingStartTime.Format(time.RFC3339)))

	return nil
}

func (y *YandexWebRTCTransport) callingResumeOffer(buf []byte) error {
	err := y.webRTCTransport.SetAnswer(buf)
	if err != nil {
		slog.Error(fmt.Sprintf("%s: failed to set answer %v", log.ObjName(y), err))

		return err
	}

	y.callingState = callingStateWaiting

	slog.Debug(fmt.Sprintf("%s: callingState %s %s", log.ObjName(y), y.callingState, y.callingStartTime.Format(time.RFC3339)))

	return nil
}

type IceServersCache struct {
	Time       time.Time
	IceServers []webrtc.ICEServer
}

const (
	iceServersCacheTimeout  = 24 * time.Hour
	iceServersCacheFileName = "iceServers.data"
)

func (y *YandexWebRTCTransport) getIceServer() (*webrtc.ICEServer, error) {
	iceServers := y.loadIceServersCache()
	if iceServers == nil ||
		len(*iceServers) == 0 {
		slog.Debug(fmt.Sprintf("%s: trying to get iceServers from yandex server", log.ObjName(y)))

		iceServer, err := GetIceServerFromJoinIdOrLink(y.cfg.JoinId)
		if err != nil {
			slog.Debug(fmt.Sprintf("%s: get iceServers from yandex server error %v", log.ObjName(y), err))

			return nil, err
		}

		slog.Debug(fmt.Sprintf("%s: success got iceServers from yandex server, caching", log.ObjName(y)))

		iceServers = &[]webrtc.ICEServer{*iceServer}

		y.saveIceServersCache(iceServers)
	}

	if len(*iceServers) == 0 {
		return nil, errors.New("empty iceServers")
	}

	return &(*iceServers)[0], nil
}

func (y *YandexWebRTCTransport) loadIceServersCache() *[]webrtc.ICEServer {
	slog.Debug(fmt.Sprintf("%s: trying to read iceServers from cache", log.ObjName(y)))

	iceServersCacheBuf, err := app.ReadCacheFile(iceServersCacheFileName)
	if err != nil {
		return nil
	}

	iceServersCacheBuf, err = y.signaling.cipher.Decrypt(iceServersCacheBuf)
	if err != nil {
		return nil
	}

	var iceServersCache IceServersCache
	if err := yaml.Unmarshal(iceServersCacheBuf, &iceServersCache); err != nil {
		return nil
	}

	if time.Since(iceServersCache.Time) < iceServersCacheTimeout &&
		len(iceServersCache.IceServers) > 0 {
		slog.Debug(fmt.Sprintf("%s: iceServers readed from cache", log.ObjName(y)))

		return &iceServersCache.IceServers
	}

	return nil
}

func (y *YandexWebRTCTransport) saveIceServersCache(iceServers *[]webrtc.ICEServer) {
	iceServersCache := &IceServersCache{Time: time.Now(), IceServers: *iceServers}
	iceServersCacheBuf, err := yaml.Marshal(&iceServersCache)
	if err != nil {
		slog.Debug(fmt.Sprintf("%s: iceServers save to cache error %v", log.ObjName(y), err))

		return
	}

	iceServersCacheBuf, err = y.signaling.cipher.Encrypt(iceServersCacheBuf)
	if err != nil {
		slog.Debug(fmt.Sprintf("%s: iceServers save to cache error %v", log.ObjName(y), err))

		return
	}

	if err := app.WriteCacheFile(iceServersCacheFileName, iceServersCacheBuf); err != nil {
		slog.Debug(fmt.Sprintf("%s: iceServers save to cache error %v", log.ObjName(y), err))

		return
	}
}
