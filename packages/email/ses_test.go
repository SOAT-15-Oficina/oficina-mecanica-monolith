package email

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSESClient struct {
	in  *sesv2.SendEmailInput
	err error
}

func (f *fakeSESClient) SendEmail(_ context.Context, in *sesv2.SendEmailInput, _ ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
	f.in = in
	return &sesv2.SendEmailOutput{}, f.err
}

func TestSESSend_HTMLMessage(t *testing.T) {
	fake := &fakeSESClient{}
	p := &sesProvider{client: fake, from: "oficina@example.com"}

	err := p.Send(context.Background(), Message{
		To:      []string{"cliente@example.com"},
		Subject: "Orçamento",
		Body:    "<p>ok</p>",
		HTML:    true,
	})

	require.NoError(t, err)
	require.NotNil(t, fake.in)
	assert.Equal(t, "oficina@example.com", *fake.in.FromEmailAddress)
	assert.Equal(t, []string{"cliente@example.com"}, fake.in.Destination.ToAddresses)
	assert.Equal(t, "Orçamento", *fake.in.Content.Simple.Subject.Data)
	require.NotNil(t, fake.in.Content.Simple.Body.Html)
	assert.Equal(t, "<p>ok</p>", *fake.in.Content.Simple.Body.Html.Data)
	assert.Nil(t, fake.in.Content.Simple.Body.Text)
}

func TestSESSend_PlainTextMessage(t *testing.T) {
	fake := &fakeSESClient{}
	p := &sesProvider{client: fake, from: "oficina@example.com"}

	err := p.Send(context.Background(), Message{
		To:      []string{"cliente@example.com"},
		Subject: "Aviso",
		Body:    "texto",
	})

	require.NoError(t, err)
	require.NotNil(t, fake.in.Content.Simple.Body.Text)
	assert.Equal(t, "texto", *fake.in.Content.Simple.Body.Text.Data)
	assert.Nil(t, fake.in.Content.Simple.Body.Html)
}

func TestSESSend_UsesReplyToAndConfigSet(t *testing.T) {
	fake := &fakeSESClient{}
	p := &sesProvider{
		client:    fake,
		from:      "oficina@example.com",
		replyTo:   "contato@example.com",
		configSet: "oficina-prod",
	}

	require.NoError(t, p.Send(context.Background(), Message{
		To: []string{"cliente@example.com"}, Subject: "s", Body: "b",
	}))

	assert.Equal(t, []string{"contato@example.com"}, fake.in.ReplyToAddresses)
	assert.Equal(t, "oficina-prod", *fake.in.ConfigurationSetName)
}

func TestSESSend_MessageFromOverridesDefault(t *testing.T) {
	fake := &fakeSESClient{}
	p := &sesProvider{client: fake, from: "padrao@example.com"}

	require.NoError(t, p.Send(context.Background(), Message{
		From: "especifico@example.com",
		To:   []string{"cliente@example.com"}, Subject: "s", Body: "b",
	}))

	assert.Equal(t, "especifico@example.com", *fake.in.FromEmailAddress)
}

func TestSESSend_EmptyRecipients(t *testing.T) {
	p := &sesProvider{client: &fakeSESClient{}, from: "oficina@example.com"}

	err := p.Send(context.Background(), Message{Subject: "s", Body: "b"})

	assert.ErrorContains(t, err, "recipient list is empty")
}

// Em sandbox o SES recusa destinatario nao verificado. O erro tem que subir,
// nao virar silencio.
func TestSESSend_PropagatesClientError(t *testing.T) {
	sesErr := errors.New("MessageRejected: Email address is not verified")
	p := &sesProvider{client: &fakeSESClient{err: sesErr}, from: "oficina@example.com"}

	err := p.Send(context.Background(), Message{
		To: []string{"nao-verificado@example.com"}, Subject: "s", Body: "b",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, sesErr)
	assert.ErrorContains(t, err, "ses: send failed")
}

func TestNewSES_RequiresSender(t *testing.T) {
	_, err := newSES(Config{Region: "sa-east-1"})

	assert.ErrorContains(t, err, "sender email is not configured")
}
