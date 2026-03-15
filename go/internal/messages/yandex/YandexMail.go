package yandex

import (
	"errors"
	"fmt"
	"log/slog"
	"ntun/internal/log"
	"sync"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/wneessen/go-mail"
)

const (
	imapServer = "imap.yandex.ru:993"
	smtpServer = "smtp.yandex.com"
	smtpPort   = 587
)

type inboxProcessor func(start bool) error

type YandexMail struct {
	lock            sync.Mutex
	client          *imapclient.Client
	idleCmd         *imapclient.IdleCommand
	email, password string
	inboxProcessor  inboxProcessor
	newMailCh       chan struct{}
}

func NewYandexMail(email, password string, inboxProcessor inboxProcessor) *YandexMail {
	return &YandexMail{
		email:          email,
		password:       password,
		inboxProcessor: inboxProcessor,
		newMailCh:      make(chan struct{}, 1),
	}
}

func (s *YandexMail) Listen() error {
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

func (s *YandexMail) handleClient(client *imapclient.Client) error {
	s.client = client

	defer s.Close()

	if err := s.client.Login(s.email, s.password).Wait(); err != nil {
		return err
	}

	_, err := s.client.Select("INBOX", nil).Wait()
	if err != nil {
		return err
	}

	if err := s.inboxProcessor(true); err != nil {
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

			if err := s.inboxProcessor(false); err != nil {
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

func (s *YandexMail) Close() error {
	s.lock.Lock()
	if s.client == nil {
		s.lock.Unlock()

		return errors.New("already closed")
	}

	err := s.shutdown()

	s.lock.Unlock()

	return err
}

func (s *YandexMail) shutdown() error {
	if s.idleCmd != nil {
		s.idleCmd.Close()
		s.idleCmd = nil
	}

	err := s.client.Close()
	if err != nil {
		return err
	}

	s.client = nil

	close(s.newMailCh)

	return nil
}

func (s *YandexMail) DeleteMail(uid imap.UID) error {
	storeCmd := s.client.Store(imap.UIDSetNum(uid), &imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagDeleted},
	}, nil)

	if err := storeCmd.Close(); err != nil {
		return err
	}

	if err := s.client.Expunge().Close(); err != nil {
		return err
	}

	return nil
}

func (s *YandexMail) SendMail(subject, content string) error {
	msg := mail.NewMsg()

	if err := msg.From(s.email); err != nil {
		return err
	}

	if err := msg.To(s.email); err != nil {
		return err
	}

	msg.Subject(subject)
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

	slog.Debug(fmt.Sprintf("%s: sent mail %s %d bytes", log.ObjName(s), subject, len(content)))

	return nil
}
