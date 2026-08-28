package email

import (
	"context"
	"testing"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/stretchr/testify/assert"
)

func TestNew_Mailhog(t *testing.T) {
	cfg := Config{Host: "localhost", Port: 1025, From: "test@test.com"}
	provider, err := New("mailhog", cfg)
	assert.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestNew_UnknownProvider(t *testing.T) {
	cfg := Config{}
	provider, err := New("unknown", cfg)
	assert.Error(t, err)
	assert.Nil(t, provider)
}

func TestWorkOrderNotificationSender_NilReceiverSkipsSend(t *testing.T) {
	var sender *WorkOrderNotificationSender

	err := sender.SendBudget(context.Background(), application.BudgetNotification{})
	assert.NoError(t, err)

	err = sender.SendPurchaseAlert(context.Background(), application.PurchaseAlertNotification{})
	assert.NoError(t, err)
}

func TestWorkOrderNotificationSender_NilProviderSkipsSend(t *testing.T) {
	sender := NewWorkOrderNotificationSender(nil)

	err := sender.SendBudget(context.Background(), application.BudgetNotification{})
	assert.NoError(t, err)

	err = sender.SendPurchaseAlert(context.Background(), application.PurchaseAlertNotification{})
	assert.NoError(t, err)
}
