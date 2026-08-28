package service

import (
	"context"
	"log"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
)

type WorkOrderStatusNotifier interface {
	NotifyTransition(ctx context.Context, workOrder *domain.WorkOrder, previousStatus domain.WorkOrderStatus)
}

type workOrderStatusNotifier struct {
	custRepo      application.CustomerRepository
	statusSender  application.StatusChangeSender
	budgetSvc     BudgetService
}

func NewWorkOrderStatusNotifier(
	custRepo application.CustomerRepository,
	statusSender application.StatusChangeSender,
	budgetSvc BudgetService,
) WorkOrderStatusNotifier {
	return &workOrderStatusNotifier{
		custRepo:     custRepo,
		statusSender: statusSender,
		budgetSvc:    budgetSvc,
	}
}

func (n *workOrderStatusNotifier) NotifyTransition(
	ctx context.Context,
	workOrder *domain.WorkOrder,
	previousStatus domain.WorkOrderStatus,
) {
	if workOrder == nil {
		return
	}

	newStatus := workOrder.Status

	if newStatus == domain.WorkOrderStatusWaitingApproval {
		previous := previousStatus
		if err := n.budgetSvc.GenerateAndSendBudget(ctx, workOrder.ID, &previous); err != nil {
			log.Printf("work order status notification: budget email failed for work order %s: %v", workOrder.ID, err)
		}
		return
	}

	if n.statusSender == nil {
		log.Printf("work order status notification: sender not configured for work order %s", workOrder.ID)
		return
	}

	customer, err := n.custRepo.FindByID(ctx, workOrder.CustomerID)
	if err != nil {
		log.Printf("work order status notification: find customer for work order %s: %v", workOrder.ID, err)
		return
	}

	if err := n.statusSender.SendStatusChange(ctx, application.StatusChangeNotification{
		CustomerEmail:       customer.Email,
		CustomerName:        customer.Name,
		WorkOrderCode:       workOrder.Code,
		PreviousStatusLabel: domain.WorkOrderStatusLabel(previousStatus),
		NewStatusLabel:      domain.WorkOrderStatusLabel(newStatus),
		Message:             statusChangeMessage(previousStatus, newStatus),
	}); err != nil {
		log.Printf("work order status notification: send email for work order %s: %v", workOrder.ID, err)
	}
}

func statusChangeMessage(previousStatus, newStatus domain.WorkOrderStatus) string {
	switch newStatus {
	case domain.WorkOrderStatusInDiagnosis:
		return "Sua ordem de serviço entrou em diagnóstico. Em breve nossa equipe concluirá a avaliação do veículo."
	case domain.WorkOrderStatusApproved:
		return "Seu orçamento foi aprovado. Em seguida iniciaremos a execução dos serviços autorizados."
	case domain.WorkOrderStatusCanceled:
		if previousStatus == domain.WorkOrderStatusWaitingApproval {
			return "Seu orçamento foi recusado e a ordem de serviço foi cancelada."
		}
		return "Sua ordem de serviço foi cancelada."
	case domain.WorkOrderStatusInProgress:
		return "A execução dos serviços da sua ordem de serviço foi iniciada."
	case domain.WorkOrderStatusFinished:
		return "Os serviços da sua ordem de serviço foram finalizados."
	case domain.WorkOrderStatusDelivered:
		return "Sua ordem de serviço foi entregue. Obrigado por confiar em nossa oficina."
	default:
		return "O status da sua ordem de serviço foi atualizado."
	}
}
