package memail

import (
	"crypto/tls"
	"fmt"
	"gopkg.in/gomail.v2"
	"io"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type EmailMessage struct {
	Msg *gomail.Message
	FC  func(msg *gomail.Message, err error)
}

type EmailClient struct {
	Host     string
	Port     int
	Username string
	Password string
	SSL      bool
	Auth     smtp.Auth
	MsgCh    chan EmailMessage
	c        *smtp.Client
}

func NewEmailClient(host string, port int, username string, password string, cacheNumber int) *EmailClient {
	cli := &EmailClient{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		SSL:      port == 465,
		MsgCh:    make(chan EmailMessage, cacheNumber),
	}
	return cli
}

func (ecli *EmailClient) Dial() (gomail.SendCloser, error) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ecli.Host, ecli.Port), 30*time.Second)
	if err != nil {
		fmt.Println("dial", err)
		return nil, err
	}
	tlsConfig := &tls.Config{ServerName: ecli.Host}
	if ecli.SSL {
		conn = tls.Client(conn, tlsConfig)
	}
	c, err := smtp.NewClient(conn, ecli.Host)
	if err != nil {
		fmt.Println("newClient", err)
		return nil, err
	}
	if !ecli.SSL {
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(tlsConfig); err != nil {
				c.Close()
				fmt.Println("tls", err)
				return nil, err
			}
		}
	}
	if ecli.Auth == nil && ecli.Username != "" {
		if ok, auths := c.Extension("AUTH"); ok {
			if strings.Contains(auths, "CRAM-MD5") {
				ecli.Auth = smtp.CRAMMD5Auth(ecli.Username, ecli.Password)
			} else if strings.Contains(auths, "LOGIN") &&
				!strings.Contains(auths, "PLAIN") {
				ecli.Auth = &loginAuth{
					username: ecli.Username,
					password: ecli.Password,
					host:     ecli.Host,
				}
			} else {
				ecli.Auth = smtp.PlainAuth("", ecli.Username, ecli.Password, ecli.Host)
			}
		}
	}

	if ecli.Auth != nil {
		if err = c.Auth(ecli.Auth); err != nil {
			c.Close()
			fmt.Println("auth:", err)
			return nil, err
		}
	}
	ecli.c = c
	return ecli, nil
}

func (ecli *EmailClient) Send(from string, to []string, msg io.WriterTo) error {
	if err := ecli.c.Mail(from); err != nil {
		if err == io.EOF {
			// This is probably due to a timeout, so reconnect and try again.
			sc, derr := ecli.Dial()
			if derr == nil {
				if s, ok := sc.(*EmailClient); ok {
					*ecli = *s
					return ecli.Send(from, to, msg)
				}
			}
		}
		return err
	}

	for _, addr := range to {
		if err := ecli.c.Rcpt(addr); err != nil {
			return err
		}
	}

	w, err := ecli.c.Data()
	if err != nil {
		return err
	}

	if _, err = msg.WriteTo(w); err != nil {
		w.Close()
		return err
	}

	return w.Close()
}

func (ecli *EmailClient) Close() error {
	if err := ecli.c.Quit(); err != nil && err != io.EOF {
		return err
	}
	return nil
}

func (ecli *EmailClient) GoRun() {
	var s gomail.SendCloser
	var err error
	open := false
	for {
		select {
		case m, ok := <-ecli.MsgCh:
			if !ok {
				return
			}
			if !open {
				if s, err = ecli.Dial(); err != nil {
					goto CALLBACK
				}
				open = true
			}
			err = gomail.Send(s, m.Msg)
			if err != nil {
				goto CALLBACK
			}
		CALLBACK:
			if m.FC != nil {
				m.FC(m.Msg, err)
			}
		case <-time.After(20 * time.Second):
			if open {
				if err := s.Close(); err != nil {
					panic(err)
				}
				open = false
			}

		}
	}
}
