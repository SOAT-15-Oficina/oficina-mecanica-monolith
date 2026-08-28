package service

import (
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/packages/email"
)

func NewWorkOrderStatusServiceWithNotifications(
	woRepo application.WorkOrderRepository,
	wosRepo application.WorkOrderServiceRepository,
	customerRepo application.CustomerRepository,
	emailProv email.Provider,
	baseURL string,
) WorkOrderStatusService {
	sender := email.NewWorkOrderNotificationSender(emailProv)
	budgetSvc := NewBudgetService(woRepo, wosRepo, customerRepo, sender, baseURL)
	notifier := NewWorkOrderStatusNotifier(customerRepo, sender, budgetSvc)
	return NewWorkOrderStatusService(woRepo, notifier)
}
