package service

import (
	"context"
	"log/slog"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/observability"
)

type WorkOrderStatusNotifier interface {
	NotifyTransition(ctx context.Context, workOrder *domain.WorkOrder, previousStatus domain.WorkOrderStatus)
}

type workOrderStatusNotifier struct {
	custRepo     application.CustomerRepository
	statusSender application.StatusChangeSender
	budgetSvc    BudgetService
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

// A NOTIFICACAO NAO FALHA A OPERACAO -- mas passa a ser visivel quando falha.
//
// Nenhum caminho aqui devolve erro: a transicao de status ja aconteceu e ja foi
// gravada, e desfaze-la porque um e-mail nao saiu seria trocar um problema
// pequeno por um grande. Antes disto, porem, a falha ia para `log.Printf` e
// morria ali. Agora cada uma carrega `@work_order_id`, que e o segundo gatilho
// do alerta exigido pela fase (`status:error @work_order_id:*`).
func (n *workOrderStatusNotifier) NotifyTransition(
	ctx context.Context,
	workOrder *domain.WorkOrder,
	previousStatus domain.WorkOrderStatus,
) {
	if workOrder == nil {
		return
	}

	newStatus := workOrder.Status

	logger := observability.FromContext(ctx).With(
		slog.String(observability.KeyWorkOrderID, workOrder.ID.String()),
		slog.String(observability.KeyWorkOrderCode, workOrder.Code),
		slog.String(observability.KeyFrom, string(previousStatus)),
		slog.String(observability.KeyTo, string(newStatus)))

	// O orcamento tem evento proprio (`budget.sent`/`budget.send_failed`), que
	// sai de dentro do budgetSvc. Aqui so sobra o erro que ele devolve, que e
	// falha ANTES do envio -- montar o orcamento, achar o cliente, calcular o
	// total.
	if newStatus == domain.WorkOrderStatusWaitingApproval {
		previous := previousStatus
		if err := n.budgetSvc.GenerateAndSendBudget(ctx, workOrder.ID, &previous); err != nil {
			logger.LogAttrs(ctx, slog.LevelError, "budget generation failed",
				observability.Event(observability.EventBudgetSendFailed),
				observability.Err(err))
		}
		return
	}

	// Configuracao ausente, e nao falha: WARN, sem `@integration`. Ver o mesmo
	// raciocinio em budget_service.go.
	if n.statusSender == nil {
		logger.WarnContext(ctx, "status change sender is not configured")
		return
	}

	customer, err := n.custRepo.FindByID(ctx, workOrder.CustomerID)
	if err != nil {
		logger.LogAttrs(ctx, slog.LevelError, "find customer for status notification",
			observability.Integration(observability.IntegrationRDS),
			observability.Err(err))
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
		logger.LogAttrs(ctx, slog.LevelError, "status change email failed",
			observability.Integration(observability.IntegrationSES),
			observability.Err(err))
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
