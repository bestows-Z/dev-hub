package emailcode

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/config"
)

type Sender interface {
	Send(context.Context, string, string, string) error
}

type SMTPSender struct {
	cfg  config.SMTPConfig
	from *mail.Address
}

func NewSMTPSender(cfg config.SMTPConfig) (*SMTPSender, error) {
	from, err := mail.ParseAddress(cfg.From)
	if err != nil || from.Address == "" {
		return nil, fmt.Errorf("invalid SMTP_FROM")
	}
	if cfg.Host == "" || cfg.Port < 1 || cfg.Port > 65535 {
		return nil, fmt.Errorf("invalid SMTP_HOST or SMTP_PORT")
	}
	if cfg.TLSMode != "none" && cfg.TLSMode != "tls" && cfg.TLSMode != "starttls" {
		return nil, fmt.Errorf("invalid SMTP_TLS_MODE")
	}
	if (cfg.Username == "") != (cfg.Password == "") {
		return nil, fmt.Errorf("SMTP_USERNAME and SMTP_PASSWORD must both be set")
	}
	return &SMTPSender{cfg: cfg, from: from}, nil
}

func (s *SMTPSender) Send(ctx context.Context, recipient, code, purpose string) error {
	to, err := mail.ParseAddress(recipient)
	if err != nil || to.Address != recipient || strings.ContainsAny(recipient, "\r\n") {
		return fmt.Errorf("invalid recipient")
	}
	address := net.JoinHostPort(s.cfg.Host, strconv.Itoa(s.cfg.Port))
	connection, err := (&net.Dialer{Timeout: 6 * time.Second}).DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("connect SMTP: %w", err)
	}
	defer connection.Close()
	_ = connection.SetDeadline(time.Now().Add(10 * time.Second))
	if s.cfg.TLSMode == "tls" {
		secure := tls.Client(connection, &tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12})
		if err := secure.HandshakeContext(ctx); err != nil {
			return fmt.Errorf("SMTP TLS handshake: %w", err)
		}
		connection = secure
	}
	client, err := smtp.NewClient(connection, s.cfg.Host)
	if err != nil {
		return fmt.Errorf("start SMTP client: %w", err)
	}
	defer client.Close()
	if s.cfg.TLSMode == "starttls" {
		if err := client.StartTLS(&tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("SMTP STARTTLS: %w", err)
		}
	}
	if s.cfg.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)); err != nil {
			return fmt.Errorf("SMTP auth: %w", err)
		}
	}
	if err := client.Mail(s.from.Address); err != nil {
		return fmt.Errorf("SMTP from: %w", err)
	}
	if err := client.Rcpt(recipient); err != nil {
		return fmt.Errorf("SMTP recipient: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP data: %w", err)
	}
	label := "注册"
	if purpose == "login" {
		label = "登录"
	} else if purpose == "change_email" {
		label = "更换邮箱"
	}
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: DevHub verification code\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n您的 DevHub %s验证码是 %s。10 分钟内有效，请勿转发给他人。\r\n", s.from.String(), recipient, label, code)
	if _, err := io.WriteString(writer, message); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write SMTP message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finish SMTP message: %w", err)
	}
	// DATA was accepted; a disconnect during QUIT must not invalidate its code.
	_ = client.Quit()
	return nil
}
