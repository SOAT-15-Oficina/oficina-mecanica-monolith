package email

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
)

const (
	defaultMailhogHost = "localhost"
	defaultMailhogPort = 1025
)

type mailhogProvider struct {
	addr string
	from string
}

func newMailhog(cfg Config) *mailhogProvider {
	host := cfg.Host
	if host == "" {
		host = defaultMailhogHost
	}
	port := cfg.Port
	if port == 0 {
		port = defaultMailhogPort
	}
	from := cfg.From
	if from == "" {
		from = "no-reply@localhost"
	}
	return &mailhogProvider{
		addr: fmt.Sprintf("%s:%d", host, port),
		from: from,
	}
}

func (p *mailhogProvider) Send(ctx context.Context, msg Message) error {
	if len(msg.To) == 0 {
		return fmt.Errorf("mailhog: recipient list is empty")
	}

	from := msg.From
	if from == "" {
		from = p.from
	}

	contentType := "text/plain; charset=\"utf-8\""
	if msg.HTML {
		contentType = "text/html; charset=\"utf-8\""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", from)
	fmt.Fprintf(&b, "To: %s\r\n", strings.Join(msg.To, ", "))
	fmt.Fprintf(&b, "Subject: %s\r\n", msg.Subject)
	fmt.Fprintf(&b, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&b, "Content-Type: %s\r\n", contentType)
	b.WriteString("\r\n")
	b.WriteString(msg.Body)

	errCh := make(chan error, 1)
	go func() {
		errCh <- smtp.SendMail(p.addr, nil, from, msg.To, []byte(b.String()))
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("mailhog: send failed: %w", err)
		}
		return nil
	}
}
