package yandex

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/quotedprintable"
	"ntun/internal/log"
	"ntun/internal/utils"
	"ntun/ntun/cipher"
	"sync"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/google/uuid"
	"github.com/wneessen/go-mail"
)

const (
	imapServer = "imap.yandex.ru:993"
	smtpServer = "smtp.yandex.com"
	smtpPort   = 587

	mailSubject = "TUN_MSG"
)

type YandexMailService struct {
	lock            sync.Mutex
	client          *imapclient.Client
	idleCmd         *imapclient.IdleCommand
	email, password string
	cipher          *cipher.CipherAesGcm
	newMailCh       chan struct{}
	mailCh          chan []byte
}

func NewYandexMailService(email, password, cipherKey string) (*YandexMailService, error) {
	cipher, err := cipher.NewCipherAesGcm([]byte(cipherKey))
	if err != nil {
		return nil, err
	}

	return &YandexMailService{
		email:     email,
		password:  password,
		cipher:    cipher,
		newMailCh: make(chan struct{}, 1),
		mailCh:    make(chan []byte),
	}, nil
}

func (s *YandexMailService) Listen() error {
	opts := &imapclient.Options{
		UnilateralDataHandler: &imapclient.UnilateralDataHandler{
			Mailbox: func(data *imapclient.UnilateralDataMailbox) {
				if data.NumMessages != nil {
					select {
					case s.newMailCh <- struct{}{}:
					default:
					}
				}
			},
		},
	}

	client, err := imapclient.DialTLS(imapServer, opts)
	if err != nil {
		return err
	}

	go s.handleClient(client)

	return nil
}

func (s *YandexMailService) handleClient(client *imapclient.Client) error {
	s.client = client

	defer s.Close()

	if err := s.client.Login(s.email, s.password).Wait(); err != nil {
		return err
	}

	_, err := s.client.Select("INBOX", nil).Wait()
	if err != nil {
		return err
	}

	if err := s.process(); err != nil {
		return err
	}

	for {
		s.idleCmd, err = s.client.Idle()
		if err != nil {
			return err
		}
		defer s.Close()

		idleDone := make(chan error, 1)

		go func() {
			idleDone <- s.idleCmd.Wait()
		}()

		select {
		case <-s.newMailCh:
			if err := s.idleCmd.Close(); err != nil {
				return err
			}

			if err := <-idleDone; err != nil {
				return err
			}

			if err := s.process(); err != nil {
				return err
			}

		case err := <-idleDone:
			if err != nil {
				return err
			}

		case <-time.After(25 * time.Minute):
			if err := s.idleCmd.Close(); err != nil {
				return err
			}

			if err := <-idleDone; err != nil {
				return err
			}
		}
	}
}

func (s *YandexMailService) Close() error {
	s.lock.Lock()
	if s.client == nil {
		s.lock.Unlock()

		return errors.New("already closed")
	}

	if s.idleCmd != nil {
		s.idleCmd.Close()
	}

	err := s.client.Close()
	s.client = nil
	s.lock.Unlock()

	close(s.newMailCh)
	close(s.mailCh)

	return err
}

func (s *YandexMailService) process() error {
	searchCriteria := &imap.SearchCriteria{
		Header: []imap.SearchCriteriaHeaderField{
			{Key: "Subject", Value: mailSubject},
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
		Envelope: true,
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

		var subj string
		var body []byte

		for {
			item := msg.Next()
			if item == nil {
				break
			}

			switch item := item.(type) {
			case imapclient.FetchItemDataEnvelope:
				subj = item.Envelope.Subject
				// slog.Debug(fmt.Sprintf("%s: subject %s", log.ObjName(s), item.Envelope.Subject))

			case imapclient.FetchItemDataBodySection:
				body, err = io.ReadAll(quotedprintable.NewReader(item.Literal))
				if err != nil {
					slog.Debug(fmt.Sprintf("%s: error reading msg body %v", log.ObjName(s), err))

					continue
				}
			}
		}

		slog.Debug(fmt.Sprintf("%s: recieve mail %s %d bytes", log.ObjName(s), subj, len(body)))

		s.mailCh <- body

		storeCmd := s.client.Store(seqSet, &imap.StoreFlags{
			Op:    imap.StoreFlagsAdd,
			Flags: []imap.Flag{imap.FlagDeleted},
		}, nil)

		if err := storeCmd.Close(); err != nil {
			return err
		}
	}

	if err := fetchCmd.Close(); err != nil {
		return err
	}

	if err := s.client.Expunge().Close(); err != nil {
		return err
	}

	return nil
}

func (s *YandexMailService) sendSmtpMail(content string) error {
	msg := mail.NewMsg()

	if err := msg.From(s.email); err != nil {
		return err
	}

	if err := msg.To(s.email); err != nil {
		return err
	}

	subj := fmt.Sprintf("%s_%s", mailSubject, uuid.New())
	msg.Subject(subj)

	msg.SetBodyString(mail.TypeTextPlain, content)

	client, err := mail.NewClient(smtpServer,
		mail.WithPort(smtpPort),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(s.email),
		mail.WithPassword(s.password),
	)
	if err != nil {
		return err
	}

	err = client.DialAndSend(msg)
	if err != nil {
		return err
	}

	slog.Debug(fmt.Sprintf("%s: sent mail %s %d bytes", log.ObjName(s), subj, len(content)))

	return nil
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

func (s *YandexMailService) SendMessage(buf []byte) error {
	body, err := GZipCipherBase64Encode(s.cipher, buf)
	if err != nil {
		return err
	}

	return s.sendSmtpMail(string(body))
}

func (s *YandexMailService) RecieveMessage() ([]byte, error) {
	body, ok := <-s.mailCh
	if !ok {
		return nil, errors.New("closed")
	}

	buf, err := GZipCipherBase64Decode(s.cipher, body)
	if err != nil {
		return nil, err
	}

	return buf, nil
}
