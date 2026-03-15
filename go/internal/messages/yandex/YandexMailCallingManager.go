package yandex

import (
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"mime/quotedprintable"
	"ntun/internal/cipher"
	"ntun/internal/log"
	"ntun/internal/utils"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

const (
	mailSubjectPrefix = "TUN_MSG_"
)

type YandexMailCallingManager struct {
	YandexMail
	cipher *cipher.CipherAesGcm
}

func NewYandexMailCallingManager(email, password, cipherKey string) (*YandexMailCallingManager, error) {
	cipher, err := cipher.NewCipherAesGcm([]byte(cipherKey))
	if err != nil {
		return nil, err
	}

	s := &YandexMailCallingManager{}
	s.YandexMail = *NewYandexMail(email, password, s.processInbox)
	s.cipher = cipher

	return s, nil
}

func (s *YandexMailCallingManager) processInbox(start bool) error {
	searchCriteria := &imap.SearchCriteria{
		Header: []imap.SearchCriteriaHeaderField{
			{Key: "Subject", Value: mailSubjectPrefix},
		},
	}

	searchData, err := s.client.Search(searchCriteria, nil).Wait()
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

	fetchCmd := s.client.Fetch(seqSet, fetchOptions)

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

func (s *YandexMailCallingManager) processMail(uid imap.UID, start bool, subject string, date time.Time, buf []byte) {
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

func (s *YandexMailCallingManager) SendMessage(subject string, buf []byte) error {
	body, err := GZipCipherBase64Encode(s.cipher, buf)
	if err != nil {
		return err
	}

	return s.SendMail(mailSubjectPrefix+subject, string(body))
}

func (s *YandexMailCallingManager) RecieveMessage() (string, []byte, error) {
	// body, ok := <-s.mailCh
	// if !ok {
	// 	return nil, errors.New("closed")
	// }

	// buf, err := GZipCipherBase64Decode(s.cipher, body)
	// if err != nil {
	// 	return nil, err
	// }

	// return buf, nil

	return "", nil, nil
}
