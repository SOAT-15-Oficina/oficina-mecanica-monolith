package email

import (
	"context"
	"fmt"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

// sesSender e a fatia da API v2 do SES que este provider usa. Existe para o
// teste poder substituir o cliente sem falar com a AWS.
type sesSender interface {
	SendEmail(ctx context.Context, in *sesv2.SendEmailInput, optFns ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error)
}

type sesProvider struct {
	client    sesSender
	from      string
	replyTo   string
	configSet string
}

// newSES resolve as credenciais pela cadeia padrao do SDK. No cluster isso cai
// no IRSA (ServiceAccount anotado com a role que tem ses:SendEmail), entao nao
// existe chave estatica em lugar nenhum.
func newSES(cfg Config) (*sesProvider, error) {
	from := cfg.SESSenderEmail
	if from == "" {
		from = cfg.From
	}
	if from == "" {
		return nil, fmt.Errorf("ses: sender email is not configured")
	}

	opts := []func(*awsconfig.LoadOptions) error{}
	if cfg.Region != "" {
		opts = append(opts, awsconfig.WithRegion(cfg.Region))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("ses: load aws config: %w", err)
	}

	return &sesProvider{
		client:    sesv2.NewFromConfig(awsCfg),
		from:      from,
		replyTo:   cfg.SESReplyTo,
		configSet: cfg.SESConfigSet,
	}, nil
}

func (p *sesProvider) Send(ctx context.Context, msg Message) error {
	if len(msg.To) == 0 {
		return fmt.Errorf("ses: recipient list is empty")
	}

	from := msg.From
	if from == "" {
		from = p.from
	}

	body := &types.Body{}
	if msg.HTML {
		body.Html = &types.Content{Data: &msg.Body, Charset: strPtr("UTF-8")}
	} else {
		body.Text = &types.Content{Data: &msg.Body, Charset: strPtr("UTF-8")}
	}

	in := &sesv2.SendEmailInput{
		FromEmailAddress: &from,
		Destination:      &types.Destination{ToAddresses: msg.To},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: &msg.Subject, Charset: strPtr("UTF-8")},
				Body:    body,
			},
		},
	}
	if p.replyTo != "" {
		in.ReplyToAddresses = []string{p.replyTo}
	}
	if p.configSet != "" {
		in.ConfigurationSetName = &p.configSet
	}

	if _, err := p.client.SendEmail(ctx, in); err != nil {
		// Em sandbox o SES recusa destinatario nao verificado. O erro precisa
		// chegar inteiro ao chamador, senao a falha vira silencio.
		return fmt.Errorf("ses: send failed: %w", err)
	}

	return nil
}

func strPtr(s string) *string { return &s }
