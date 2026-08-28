package email

import (
	"context"
	"fmt"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
)

type WorkOrderNotificationSender struct {
	provider Provider
}

func NewWorkOrderNotificationSender(provider Provider) *WorkOrderNotificationSender {
	return &WorkOrderNotificationSender{provider: provider}
}

func (s *WorkOrderNotificationSender) SendBudget(ctx context.Context, notification application.BudgetNotification) error {
	if s == nil || s.provider == nil {
		return nil
	}

	body, err := RenderBudgetEmail(BudgetEmailData{
		CustomerName:        notification.CustomerName,
		WorkOrderCode:       notification.WorkOrderCode,
		PreviousStatusLabel: notification.PreviousStatusLabel,
		NewStatusLabel:      notification.NewStatusLabel,
		Amount:              notification.Amount,
		BudgetLink:          notification.BudgetLink,
		Services:            toBudgetServiceItems(notification.Services),
		ApproveAllLink:      notification.ApproveAllLink,
		RejectAllLink:       notification.RejectAllLink,
	})
	if err != nil {
		return err
	}

	return s.provider.Send(ctx, Message{
		To:      []string{notification.CustomerEmail},
		Subject: fmt.Sprintf("Orçamento - OS %s", notification.WorkOrderCode),
		Body:    body,
		HTML:    true,
	})
}

func (s *WorkOrderNotificationSender) SendPurchaseAlert(ctx context.Context, notification application.PurchaseAlertNotification) error {
	if s == nil || s.provider == nil {
		return nil
	}

	body, err := RenderPurchaseAlertEmail(PurchaseAlertEmailData{
		WorkOrderCode:  notification.WorkOrderCode,
		WorkOrderTitle: notification.WorkOrderTitle,
		Items:          toPurchaseAlertItems(notification.Items),
	})
	if err != nil {
		return err
	}

	return s.provider.Send(ctx, Message{
		To:      []string{notification.To},
		Subject: fmt.Sprintf("Alerta de Compra - OS %s", notification.WorkOrderCode),
		Body:    body,
		HTML:    true,
	})
}

func (s *WorkOrderNotificationSender) SendStatusChange(ctx context.Context, notification application.StatusChangeNotification) error {
	if s == nil || s.provider == nil {
		return nil
	}

	body, err := RenderStatusChangeEmail(StatusChangeEmailData{
		CustomerName:        notification.CustomerName,
		WorkOrderCode:       notification.WorkOrderCode,
		PreviousStatusLabel: notification.PreviousStatusLabel,
		NewStatusLabel:      notification.NewStatusLabel,
		Message:             notification.Message,
	})
	if err != nil {
		return err
	}

	return s.provider.Send(ctx, Message{
		To:      []string{notification.CustomerEmail},
		Subject: fmt.Sprintf("Atualização da OS %s - %s", notification.WorkOrderCode, notification.NewStatusLabel),
		Body:    body,
		HTML:    true,
	})
}

func toBudgetServiceItems(items []application.BudgetNotificationService) []BudgetServiceItem {
	out := make([]BudgetServiceItem, len(items))
	for i, item := range items {
		out[i] = BudgetServiceItem{
			Title:       item.Title,
			Amount:      item.Amount,
			Estimated:   item.Estimated,
			ApproveLink: item.ApproveLink,
			RejectLink:  item.RejectLink,
		}
	}
	return out
}

func toPurchaseAlertItems(items []application.PurchaseAlertNotificationItem) []PurchaseAlertItem {
	out := make([]PurchaseAlertItem, len(items))
	for i, item := range items {
		out[i] = PurchaseAlertItem{
			ServiceTitle: item.ServiceTitle,
			SupplyTitle:  item.SupplyTitle,
			Required:     item.Required,
			InStock:      item.InStock,
			ToBuy:        item.ToBuy,
		}
	}
	return out
}
