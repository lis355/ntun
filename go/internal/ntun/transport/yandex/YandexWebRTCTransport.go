package yandex

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/quotedprintable"
	"net"
	"ntun/internal/app"
	"ntun/internal/cfg"
	"ntun/internal/cipher"
	"ntun/internal/log"
	"ntun/internal/messages/yandex"
	"ntun/internal/ntun/node"
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
// Тайм-аут ожидания offer/answer - 3 минуты.

const (
	mailSubjectPrefix = "TUN_MSG_"
)

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
}

func NewYandexMailSignaling(email, password, cipherKey string) (*YandexMailSignaling, error) {
	cipher, err := cipher.NewCipherAesGcm([]byte(cipherKey))
	if err != nil {
		return nil, err
	}

	s := &YandexMailSignaling{}
	s.YandexMail = *yandex.NewYandexMail(email, password, s.processInbox)
	s.cipher = cipher

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

		s.processMail(uid, start, subj, date, []byte(body))
	}

	if err := fetchCmd.Close(); err != nil {
		return err
	}

	return nil
}

func (s *YandexMailSignaling) processMail(uid imap.UID, start bool, subject string, date time.Time, buf []byte) {
	// s.mailCh <- buf

	delete := false

	decodedBuf, err := GZipCipherBase64Decode(s.cipher, buf)
	if err != nil {
		delete = true
	} else {
		slog.Debug(fmt.Sprintf("%s: recieve mail %s %s %d bytes", log.ObjName(s), subject, date.Format(time.RFC3339), len(decodedBuf)))
		slog.Debug(fmt.Sprintf("%s", decodedBuf))
	}

	if delete {
		slog.Debug(fmt.Sprintf("%s: recieve error mail %s %s %d bytes, deleting", log.ObjName(s), subject, date.Format(time.RFC3339), len(buf)))

		s.DeleteMail(uid)
	}
}

func (s *YandexMailSignaling) SendMessage(subject string, buf []byte) error {
	body, err := GZipCipherBase64Encode(s.cipher, buf)
	if err != nil {
		return err
	}

	return s.SendMail(mailSubjectPrefix+subject, string(body))
}

// func (s *YandexMailSignaling) RecieveMessage() (string, []byte, error) {
// 	// body, ok := <-s.mailCh
// 	// if !ok {
// 	// 	return nil, errors.New("closed")
// 	// }

// 	// buf, err := GZipCipherBase64Decode(s.cipher, body)
// 	// if err != nil {
// 	// 	return nil, err
// 	// }

// 	// return buf, nil

// 	return "", nil, nil
// }

const (
	signalingMsgTypeOffer  = "offer"
	signalingMsgTypeAnswer = "answer"
)

type signalingMsg struct {
	Version string
	Sender  uuid.UUID // TODO Id type
	Typ     string
	Data    []byte
}

type YandexWebRTCTransport struct {
	cfg       *cfg.YandexWebRTCTransport
	node      *node.Node
	signaling *YandexMailSignaling
}

func NewYandexWebRTCTransport(cfg *cfg.YandexWebRTCTransport, node *node.Node) (*YandexWebRTCTransport, error) {
	signaling, err := NewYandexMailSignaling(cfg.MailUser, cfg.MailPass, node.Config.CipherKey)
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
	return nil, nil
}

func (y *YandexWebRTCTransport) Listen() error {
	return nil
}

func (y *YandexWebRTCTransport) Close() error {
	return nil
}

func (y *YandexWebRTCTransport) createMsg(typ string, data []byte) ([]byte, error) {
	msg := &signalingMsg{
		Version: app.Version,
		Sender:  y.node.Config.Id,
		Typ:     typ,
		Data:    data,
	}

	msgBuf, err := json.Marshal(&msg)
	if err != nil {
		return nil, err
	}

	return msgBuf, nil
}
