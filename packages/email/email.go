package email

import (
	"context"
	"fmt"
)

type Message struct {
	From    string
	To      []string
	Subject string
	Body    string
	HTML    bool
}

type Provider interface {
	Send(ctx context.Context, msg Message) error
}

type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string

	// Usados apenas pelo provider "ses". As credenciais vem do IRSA do pod,
	// nunca de chave estatica -- ver decisao 21 no README do
	// oficina-mecanica-infrastructure.
	Region         string
	SESSenderEmail string
	SESReplyTo     string
	SESConfigSet   string
}

func New(name string, cfg Config) (Provider, error) {
	switch name {
	case "mailhog":
		return newMailhog(cfg), nil
	case "ses":
		return newSES(cfg)
	default:
		return nil, fmt.Errorf("email: unknown provider %q", name)
	}
}
