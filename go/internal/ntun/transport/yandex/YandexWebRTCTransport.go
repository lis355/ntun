package yandex

import (
	"context"
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

	callingTimeout = 2 * time.Minute
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
		mailCh: make(chan *signalingMsg, 16),
	}

	s.YandexMail = *yandex.NewYandexMail(transportCfg.MailUser, transportCfg.MailPass, s.processInbox)

	return s, nil
}

func (s *YandexMailSignaling) Close() error {
	close(s.mailCh)

	return s.YandexMail.Close()
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
		slog.Debug(fmt.Sprintf("%s: mail is old", log.ObjName(s)))
	}

	var decodedBuf []byte
	if !delete {
		buf, decodingErr := GZipCipherBase64Decode(s.cipher, buf)
		if decodingErr != nil {
			delete = true
			slog.Debug(fmt.Sprintf("%s: recieved mail error decoding", log.ObjName(s)))
		}

		decodedBuf = buf
	}

	if !delete {
		var msg signalingMsg
		if err := json.Unmarshal(decodedBuf, &msg); err != nil {
			delete = true
			slog.Debug(fmt.Sprintf("%s: recieved mail error decoding", log.ObjName(s)))
		} else if msg.Version != app.Version {
			delete = true
			slog.Debug(fmt.Sprintf("%s: recieved mail bad version %s app version %s", log.ObjName(s), msg.Version, app.Version))
		} else {
			// Сообщения удаляет тот, для кого они предназначались
			if msg.Sender == s.node.Config.Id {
				// пропускаем, это не нам сообщение (оно нами сделанное)
				delete = false
				slog.Debug(fmt.Sprintf("%s: recieved mail is ours, skip", log.ObjName(s)))
			} else if !s.node.HasAllowedToConnectNodeId(msg.Sender) {
				delete = true
				slog.Debug(fmt.Sprintf("%s: recieved mail from unknown node", log.ObjName(s)))
			} else {
				delete = true
				slog.Debug(fmt.Sprintf("%s: recieved mail from %s node", log.ObjName(s), msg.Sender))

				// NOTE неблокирующая отправка в канал с буфером
				select {
				case s.mailCh <- &msg:
				default:
				}
			}
		}
	}

	if delete {
		slog.Debug(fmt.Sprintf("%s: recieved mail will deleted", log.ObjName(s)))
	}

	return delete
}

func (s *YandexMailSignaling) sendMessage(typ string, data []byte) error {
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

	return s.SendMail(mailSubjectPrefix+utils.RandShortString(), string(body))
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
	callingCtx       context.Context
	callingCtxCancel context.CancelFunc
	callingState     string
	callingStartTime time.Time
	callingTimer     *time.Timer
}

func NewYandexWebRTCTransport(cfg *cfg.YandexWebRTCTransport, node *node.Node) (*YandexWebRTCTransport, error) {
	signaling, err := NewYandexMailSignaling(cfg, node)
	if err != nil {
		return nil, err
	}

	return &YandexWebRTCTransport{
		cfg:       cfg,
		node:      node,
		signaling: signaling,
	}, nil
}

func (y *YandexWebRTCTransport) Transport() (net.Conn, error) {
	select {
	case <-y.callingCtx.Done():
		return nil, y.callingCtx.Err()
	default:
	}

	conn, err := y.webRTCTransport.Transport()
	if err != nil {
		return nil, err
	}

	// conn = dev.NewSnifferHexDumpDebugConn(conn, fmt.Sprintf("[%s]", log.ObjName(y)), false, true, false, false)
	conn = connections.NewBufferedConn(conn, connections.DefaultBufferedConnMaxSize, connections.DefaultBufferedConnMaxDelay)

	y.callingState = callingStateConnected

	y.callingTimer.Stop()

	slog.Debug(fmt.Sprintf("%s: callingState %s %s", log.ObjName(y), y.callingState, y.callingStartTime.Format(time.RFC3339)))

	return conn, err
}

func (c *YandexWebRTCTransport) RateLimit() *cfg.Rate {
	return &c.cfg.RateLimit
}

func (y *YandexWebRTCTransport) Listen() error {
	// TODO сделать фазу старта чтобы разбирать заявку на старте

	y.callingStart()

	go y.handleMessages()

	go func() {
		select {
		case <-y.callingCtx.Done():
			return
		case <-y.webRTCTransport.DisconnectCh:
			y.callingAbort(errors.New("transport disconnected"))

			y.callingStart()

			return
		}
	}()

	return nil
}

func (y *YandexWebRTCTransport) Close() error {
	y.abort(nil)

	return nil
}

func (y *YandexWebRTCTransport) callingStart() {
	callingCtx, callingCtxCancel := context.WithCancel(context.Background())

	y.webRTCTransport = transport.NewWebRTCTransport()
	y.callingCtx = callingCtx
	y.callingCtxCancel = callingCtxCancel
	y.callingState = callingStateNone

	if err := y.signaling.Listen(); err != nil {
		y.abort(err)

		return
	}

	// TODO пока что клиент будет пробовать сделать подключение сразу как включился
	// клиента (тот, что начинает webrtc) определяем как того, у кого есть Input
	if y.node.Input != nil {
		if err := y.callingCreateOffer(); err != nil {
			y.abort(err)

			return
		}
	}
}

func (y *YandexWebRTCTransport) callingAbort(err error) {
	select {
	case <-y.callingCtx.Done():
		return
	default:
	}

	slog.Debug(fmt.Sprintf("%s: abort calling %v", log.ObjName(y), err))

	if y.callingTimer != nil {
		y.callingTimer.Stop()
	}

	if err := y.webRTCTransport.Close(); err != nil {
		return
	}

	y.callingState = callingStateNone

	y.callingCtxCancel()

	// TODO
}

func (y *YandexWebRTCTransport) abort(err error) {
	slog.Debug(fmt.Sprintf("%s: aborting %v", log.ObjName(y), err))

	y.callingAbort(nil)

	if err := y.signaling.Close(); err != nil {
		return
	}
}

func (y *YandexWebRTCTransport) handleMessages() {
	for {
		msg := y.signaling.recieveMessage()
		if msg == nil {
			y.abort(nil)

			return
		}

		select {
		case <-y.callingCtx.Done():
			return
		default:
		}

		y.handleMessage(msg)
	}
}

func (y *YandexWebRTCTransport) handleMessage(msg *signalingMsg) {
	slog.Debug(fmt.Sprintf("%s: handleMessage msg.Typ=%s y.callingState=%s", log.ObjName(y), msg.Typ, y.callingState))

	switch msg.Typ {
	case signalingMsgTypeOffer:
		switch y.callingState {
		case callingStateNone:
			if err := y.callingCreateAnswer(msg.Buf); err != nil {
				y.callingAbort(err)

				return
			}
			return
		case callingStateOffer: // bad logic
		case callingStateAnswer:
			// retry
			if err := y.callingCreateAnswer(msg.Buf); err != nil {
				y.callingAbort(err)

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
				y.callingAbort(err)

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
	slog.Debug(fmt.Sprintf("%s: callingCreateOffer", log.ObjName(y)))

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

	err = y.signaling.sendMessage(signalingMsgTypeOffer, buf)
	if err != nil {
		slog.Error(fmt.Sprintf("%s: failed to send signal message %v", log.ObjName(y), err))

		return err
	}

	y.callingState = callingStateOffer
	y.callingStartTime = time.Now()

	y.callingTimer = time.AfterFunc(callingTimeout, y.callingTimerTimeout)

	slog.Debug(fmt.Sprintf("%s: callingState %s %s", log.ObjName(y), y.callingState, y.callingStartTime.Format(time.RFC3339)))

	return nil
}

func (y *YandexWebRTCTransport) callingCreateAnswer(buf []byte) error {
	slog.Debug(fmt.Sprintf("%s: callingCreateAnswer", log.ObjName(y)))

	buf, err := y.webRTCTransport.CreateAnswer(buf)
	if err != nil {
		slog.Error(fmt.Sprintf("%s: failed to create answer %v", log.ObjName(y), err))

		return err
	}

	err = y.signaling.sendMessage(signalingMsgTypeAnswer, buf)
	if err != nil {
		slog.Error(fmt.Sprintf("%s: failed to send signal message %v", log.ObjName(y), err))

		return err
	}

	y.callingState = callingStateAnswer
	y.callingStartTime = time.Now()

	y.callingTimer = time.AfterFunc(callingTimeout, y.callingTimerTimeout)

	slog.Debug(fmt.Sprintf("%s: callingState %s %s", log.ObjName(y), y.callingState, y.callingStartTime.Format(time.RFC3339)))

	return nil
}

func (y *YandexWebRTCTransport) callingResumeOffer(buf []byte) error {
	slog.Debug(fmt.Sprintf("%s: callingResumeOffer", log.ObjName(y)))

	err := y.webRTCTransport.SetAnswer(buf)
	if err != nil {
		slog.Error(fmt.Sprintf("%s: failed to set answer %v", log.ObjName(y), err))

		return err
	}

	y.callingState = callingStateWaiting

	slog.Debug(fmt.Sprintf("%s: callingState %s %s", log.ObjName(y), y.callingState, y.callingStartTime.Format(time.RFC3339)))

	return nil
}

func (y *YandexWebRTCTransport) callingTimerTimeout() {
	y.callingAbort(errors.New("calling timeout"))

	y.callingStart()
}
