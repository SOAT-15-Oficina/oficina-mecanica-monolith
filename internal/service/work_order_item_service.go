package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/observability"
	"github.com/google/uuid"
)

type WorkOrderItemService interface {
	ApproveService(ctx context.Context, workOrderServiceID uuid.UUID) error
	RejectService(ctx context.Context, workOrderServiceID uuid.UUID) error
	ApproveAllByWorkOrder(ctx context.Context, workOrderID uuid.UUID) error
	RejectAllByWorkOrder(ctx context.Context, workOrderID uuid.UUID) error
}

type workOrderItemService struct {
	wosRepo   application.WorkOrderServiceRepository
	woRepo    application.WorkOrderRepository
	statusSvc WorkOrderStatusService
	alerts    application.PurchaseAlertSender
	alertTo   string
}

func NewWorkOrderItemService(
	wosRepo application.WorkOrderServiceRepository,
	woRepo application.WorkOrderRepository,
	statusSvc WorkOrderStatusService,
	opts ...WorkOrderItemServiceOption,
) WorkOrderItemService {
	svc := &workOrderItemService{
		wosRepo:   wosRepo,
		woRepo:    woRepo,
		statusSvc: statusSvc,
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

type WorkOrderItemServiceOption func(*workOrderItemService)

func WithPurchaseAlert(sender application.PurchaseAlertSender, to string) WorkOrderItemServiceOption {
	return func(s *workOrderItemService) {
		s.alerts = sender
		s.alertTo = to
	}
}

func (s *workOrderItemService) ApproveService(ctx context.Context, workOrderServiceID uuid.UUID) error {
	wos, err := s.wosRepo.FindByID(ctx, workOrderServiceID)
	if err != nil {
		return fmt.Errorf("approve: find service: %w", err)
	}

	if wos.ApprovalStatus != domain.WorkOrderServiceApprovalPending {
		return nil // idempotent
	}

	if err := s.wosRepo.UpdateApprovalStatus(ctx, workOrderServiceID, domain.WorkOrderServiceApprovalApproved); err != nil {
		return fmt.Errorf("approve: update status: %w", err)
	}

	logApprovalDecision(ctx, wos.WorkOrderID, observability.DecisionApproved)

	return s.evaluateWorkOrderCompletion(ctx, wos.WorkOrderID)
}

func (s *workOrderItemService) RejectService(ctx context.Context, workOrderServiceID uuid.UUID) error {
	wos, err := s.wosRepo.FindByID(ctx, workOrderServiceID)
	if err != nil {
		return fmt.Errorf("reject: find service: %w", err)
	}

	if wos.ApprovalStatus != domain.WorkOrderServiceApprovalPending {
		return nil // idempotent
	}

	if err := s.wosRepo.UpdateApprovalStatus(ctx, workOrderServiceID, domain.WorkOrderServiceApprovalRejected); err != nil {
		return fmt.Errorf("reject: update status: %w", err)
	}

	logApprovalDecision(ctx, wos.WorkOrderID, observability.DecisionRejected)

	return s.evaluateWorkOrderCompletion(ctx, wos.WorkOrderID)
}

func (s *workOrderItemService) ApproveAllByWorkOrder(ctx context.Context, workOrderID uuid.UUID) error {
	if err := s.wosRepo.UpdateApprovalStatusByWorkOrderID(ctx, workOrderID, domain.WorkOrderServiceApprovalApproved); err != nil {
		return fmt.Errorf("approve all: update status: %w", err)
	}

	logApprovalDecision(ctx, workOrderID, observability.DecisionApproved)

	return s.evaluateWorkOrderCompletion(ctx, workOrderID)
}

func (s *workOrderItemService) RejectAllByWorkOrder(ctx context.Context, workOrderID uuid.UUID) error {
	if err := s.wosRepo.UpdateApprovalStatusByWorkOrderID(ctx, workOrderID, domain.WorkOrderServiceApprovalRejected); err != nil {
		return fmt.Errorf("reject all: update status: %w", err)
	}

	logApprovalDecision(ctx, workOrderID, observability.DecisionRejected)

	return s.evaluateWorkOrderCompletion(ctx, workOrderID)
}

func (s *workOrderItemService) evaluateWorkOrderCompletion(ctx context.Context, workOrderID uuid.UUID) error {
	services, err := s.wosRepo.FindByWorkOrderID(ctx, workOrderID)
	if err != nil {
		return fmt.Errorf("evaluate: find services: %w", err)
	}

	hasApproved := false
	for _, svc := range services {
		if svc.ApprovalStatus == domain.WorkOrderServiceApprovalPending {
			return nil // still pending decisions
		}
		if svc.ApprovalStatus == domain.WorkOrderServiceApprovalApproved {
			hasApproved = true
		}
	}

	var newStatus domain.WorkOrderStatus
	if hasApproved {
		newStatus = domain.WorkOrderStatusApproved
	} else {
		newStatus = domain.WorkOrderStatusCanceled
	}

	wo, err := s.statusSvc.TransitionTo(ctx, workOrderID, newStatus)
	if err != nil {
		return fmt.Errorf("evaluate: transition status: %w", err)
	}

	approvedTotal, err := s.wosRepo.CalculateApprovedTotalForWorkOrder(ctx, workOrderID)
	if err != nil {
		return fmt.Errorf("evaluate: calculate approved total: %w", err)
	}
	wo.TotalEstimatedPriceCents = approvedTotal

	if _, err := s.woRepo.Update(ctx, wo); err != nil {
		return fmt.Errorf("evaluate: update work order: %w", err)
	}

	if hasApproved && s.alerts != nil {
		s.sendPurchaseAlertIfNeeded(ctx, workOrderID)
	}

	return nil
}

// O alerta de compra e o unico caminho aqui que nao tem quem o observe: ele
// nasce de uma transicao ja concluida e nao devolve erro a ninguem. Sem log, um
// alerta que parou de sair e indistinguivel de um alerta que nao precisava
// sair -- e a diferenca aparece como insumo que ninguem comprou.
func (s *workOrderItemService) sendPurchaseAlertIfNeeded(ctx context.Context, workOrderID uuid.UUID) {
	logger := observability.FromContext(ctx).With(
		slog.String(observability.KeyWorkOrderID, workOrderID.String()))

	shortages, err := s.wosRepo.FindSupplyShortagesByWorkOrderID(ctx, workOrderID)
	if err != nil {
		logger.LogAttrs(ctx, slog.LevelError, "find supply shortages",
			observability.Integration(observability.IntegrationRDS),
			observability.Err(err))
		return
	}
	if len(shortages) == 0 {
		return
	}

	alerts, err := s.wosRepo.FindApprovedServicesWithShortages(ctx)
	if err != nil {
		logger.LogAttrs(ctx, slog.LevelError, "find approved services with shortages",
			observability.Integration(observability.IntegrationRDS),
			observability.Err(err))
		return
	}
	if len(alerts) == 0 {
		return
	}

	wo, err := s.woRepo.FindByID(ctx, workOrderID)
	if err != nil {
		logger.LogAttrs(ctx, slog.LevelError, "find work order for purchase alert",
			observability.Integration(observability.IntegrationRDS),
			observability.Err(err))
		return
	}

	var items []application.PurchaseAlertNotificationItem
	for _, a := range alerts {
		items = append(items, application.PurchaseAlertNotificationItem{
			ServiceTitle: a.ServiceTitle,
			SupplyTitle:  a.SupplyTitle,
			Required:     a.Required,
			InStock:      a.InStock,
			ToBuy:        a.Required - a.InStock,
		})
	}

	if err := s.alerts.SendPurchaseAlert(ctx, application.PurchaseAlertNotification{
		To:             s.alertTo,
		WorkOrderCode:  wo.Code,
		WorkOrderTitle: wo.Title,
		Items:          items,
	}); err != nil {
		logger.LogAttrs(ctx, slog.LevelError, "purchase alert email failed",
			slog.String(observability.KeyWorkOrderCode, wo.Code),
			observability.Integration(observability.IntegrationSES),
			observability.Err(err))
		return
	}

	logger.LogAttrs(ctx, slog.LevelInfo, "purchase alert sent",
		observability.Event(observability.EventPurchaseAlertSent),
		slog.String(observability.KeyWorkOrderCode, wo.Code),
		observability.Integration(observability.IntegrationSES))
}

// logApprovalDecision emite `approval.decided` -- o evento que conta as
// decisoes do cliente sobre o orcamento (`oficina.approval_decided`).
//
// A decisao vai num campo proprio, e nao no `msg`: um evento que diz que houve
// uma decisao sem dizer qual nao responde a unica pergunta que se faz dele.
func logApprovalDecision(ctx context.Context, workOrderID uuid.UUID, decision string) {
	observability.FromContext(ctx).LogAttrs(ctx, slog.LevelInfo, "customer decided on budget",
		observability.Event(observability.EventApprovalDecided),
		slog.String(observability.KeyWorkOrderID, workOrderID.String()),
		slog.String(observability.KeyDecision, decision))
}
