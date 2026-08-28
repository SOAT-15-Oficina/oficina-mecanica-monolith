package service

import (
	"context"
	"fmt"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
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

	return s.evaluateWorkOrderCompletion(ctx, wos.WorkOrderID)
}

func (s *workOrderItemService) ApproveAllByWorkOrder(ctx context.Context, workOrderID uuid.UUID) error {
	if err := s.wosRepo.UpdateApprovalStatusByWorkOrderID(ctx, workOrderID, domain.WorkOrderServiceApprovalApproved); err != nil {
		return fmt.Errorf("approve all: update status: %w", err)
	}

	return s.evaluateWorkOrderCompletion(ctx, workOrderID)
}

func (s *workOrderItemService) RejectAllByWorkOrder(ctx context.Context, workOrderID uuid.UUID) error {
	if err := s.wosRepo.UpdateApprovalStatusByWorkOrderID(ctx, workOrderID, domain.WorkOrderServiceApprovalRejected); err != nil {
		return fmt.Errorf("reject all: update status: %w", err)
	}

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

func (s *workOrderItemService) sendPurchaseAlertIfNeeded(ctx context.Context, workOrderID uuid.UUID) {
	shortages, err := s.wosRepo.FindSupplyShortagesByWorkOrderID(ctx, workOrderID)
	if err != nil || len(shortages) == 0 {
		return
	}

	alerts, err := s.wosRepo.FindApprovedServicesWithShortages(ctx)
	if err != nil || len(alerts) == 0 {
		return
	}

	wo, err := s.woRepo.FindByID(ctx, workOrderID)
	if err != nil {
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

	_ = s.alerts.SendPurchaseAlert(ctx, application.PurchaseAlertNotification{
		To:             s.alertTo,
		WorkOrderCode:  wo.Code,
		WorkOrderTitle: wo.Title,
		Items:          items,
	})
}
