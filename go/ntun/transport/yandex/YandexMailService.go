package yandex

import (
	"encoding/base64"
	"io"
	"log"
	"ntun/internal/utils"
	"ntun/ntun/cipher"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/wneessen/go-mail"
)

const (
	imapServer = "imap.yandex.ru:993"
	smtpServer = "smtp.yandex.com"
	smtpPort   = 587

	mailSubject = "TUN_MSG"
)

type YandexMailService struct {
	client          *imapclient.Client
	email, password string
	cipher          *cipher.CipherAesGcm

	Mails chan []byte
}

func NewYandexMailService(email, password, cipherKey string) (*YandexMailService, error) {
	cipher, err := cipher.NewCipherAesGcm([]byte(cipherKey))
	if err != nil {
		return nil, err
	}

	return &YandexMailService{
		email:    email,
		password: password,
		cipher:   cipher,
		Mails:    make(chan []byte),
	}, nil
}

func (s *YandexMailService) Listen() error {
	newMail := make(chan struct{}, 1)

	opts := &imapclient.Options{
		UnilateralDataHandler: &imapclient.UnilateralDataHandler{
			Mailbox: func(data *imapclient.UnilateralDataMailbox) {
				if data.NumMessages != nil {
					select {
					case newMail <- struct{}{}:
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

	s.client = client

	defer client.Close()

	if err := client.Login(s.email, s.password).Wait(); err != nil {
		return err
	}

	_, err = client.Select("INBOX", nil).Wait()
	if err != nil {
		return err
	}

	if err := s.process(); err != nil {
		return err
	}

	for {
		idleCmd, err := client.Idle()
		if err != nil {
			return err
		}
		defer idleCmd.Close()

		idleDone := make(chan error, 1)

		go func() {
			idleDone <- idleCmd.Wait()
		}()

		select {
		case <-newMail:
			if err := idleCmd.Close(); err != nil {
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
			if err := idleCmd.Close(); err != nil {
				return err
			}

			if err := <-idleDone; err != nil {
				return err
			}
		}
	}
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

		for {
			item := msg.Next()
			if item == nil {
				break
			}

			switch item := item.(type) {
			case imapclient.FetchItemDataEnvelope:
				log.Printf("Тема письма: %s", item.Envelope.Subject)

			case imapclient.FetchItemDataBodySection:
				bodyBytes, err := io.ReadAll(item.Literal)
				if err != nil {
					log.Printf("Ошибка чтения тела: %v", err)
					continue
				}

				bodyBytes, err = s.decodeMessage(strings.TrimSpace(string(bodyBytes)))
				if err != nil {
					log.Printf("Ошибка чтения тела: %v", err)
					continue
				}

				log.Printf("Тело: %s", string(bodyBytes))
				s.Mails <- bodyBytes
			}
		}

		log.Println("delete msg")
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

func (s *YandexMailService) SendMail(buf []byte) error {
	msg := mail.NewMsg()

	if err := msg.From(s.email); err != nil {
		return err
	}

	if err := msg.To(s.email); err != nil {
		return err
	}

	msg.Subject(mailSubject)

	body, err := s.encodeMessage(buf)
	if err != nil {
		return err
	}

	msg.SetBodyString(mail.TypeTextPlain, body)
	msg.SetDate()

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

	return nil
}

func GZipCipherBase64Encode(cipher *cipher.CipherAesGcm, buf []byte) (string, error) {
	compressedBuf, err := utils.GZipEncode(buf)
	if err != nil {
		return "", err
	}

	encryptedBuf, err := cipher.Encrypt(compressedBuf)
	if err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(encryptedBuf)

	return encoded, nil
}

func GZipCipherBase64Decode(cipher *cipher.CipherAesGcm, encoded string) ([]byte, error) {
	encryptedBuf, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

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

func (s *YandexMailService) encodeMessage(msg []byte) (string, error) {
	return GZipCipherBase64Encode(s.cipher, msg)
}

func (s *YandexMailService) decodeMessage(msg string) ([]byte, error) {
	return GZipCipherBase64Decode(s.cipher, msg)
}

func main() {

}
