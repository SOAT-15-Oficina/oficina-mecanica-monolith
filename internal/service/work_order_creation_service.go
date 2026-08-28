package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/google/uuid"
)

var (
	ErrWorkOrderInvalidStatusForItems = errors.New("work order status does not allow changing services or supplies")
	ErrWorkshopServiceInactive        = errors.New("workshop service is inactive")
	ErrWorkOrderServiceOwnership      = errors.New("work order service does not belong to this work order")
	ErrWorkOrderNotInProgress         = errors.New("work order must be in EM_EXECUCAO status")
	ErrServiceNotPending              = errors.New("service must be PENDENTE to start")
	ErrServiceNotApproved             = errors.New("service must be approved to start")
	ErrServiceNotInProgress           = errors.New("service must be in EM_EXECUCAO status to finalize")
	ErrInsufficientStock              = errors.New("insufficient stock for service supplies")
)

type AddWorkOrderServiceInput struct {
	ServiceID            uuid.UUID
	EstimatedTimeMinutes *int
}

type AddWorkOrderSupplyInput struct {
	SupplyID uuid.UUID
	Quantity int
}

type WorkOrderCreationService interface {
	AddServices(ctx context.Context, workOrderID uuid.UUID, items []AddWorkOrderServiceInput) ([]domain.WorkOrderService, error)
	AddSupplies(ctx context.Context, workOrderID, wosID uuid.UUID, items []AddWorkOrderSupplyInput) ([]domain.WorkOrderServiceSupply, error)
	RemoveSupplyFromService(ctx context.Context, workOrderID, wosID, supplyID uuid.UUID) error
	RemoveService(ctx context.Context, workOrderID, wosID uuid.UUID) error
	StartService(ctx context.Context, workOrderID, wosID uuid.UUID) error
	FinalizeService(ctx context.Context, workOrderID, wosID uuid.UUID) error
}

type workOrderCreationService struct {
	woRepo     application.WorkOrderRepository
	wosRepo    application.WorkOrderServiceRepository
	wsRepo     application.WorkshopServiceRepository
	supplyRepo application.SupplyRepository
	statusSvc  WorkOrderStatusService
	budget     BudgetService
}

func NewWorkOrderCreationService(
	woRepo application.WorkOrderRepository,
	wosRepo application.WorkOrderServiceRepository,
	wsRepo application.WorkshopServiceRepository,
	supplyRepo application.SupplyRepository,
	statusSvc WorkOrderStatusService,
	opts ...WorkOrderCreationServiceOption,
) WorkOrderCreationService {
	svc := &workOrderCreationService{
		woRepo:     woRepo,
		wosRepo:    wosRepo,
		wsRepo:     wsRepo,
		supplyRepo: supplyRepo,
		statusSvc:  statusSvc,
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

type WorkOrderCreationServiceOption func(*workOrderCreationService)

func WithBudgetRefresh(budget BudgetService) WorkOrderCreationServiceOption {
	return func(s *workOrderCreationService) {
		s.budget = budget
	}
}

func canChangeWorkOrderItems(status domain.WorkOrderStatus) bool {
	return status == domain.WorkOrderStatusReceived ||
		status == domain.WorkOrderStatusInDiagnosis ||
		status == domain.WorkOrderStatusWaitingApproval
}

func (s *workOrderCreationService) ensureWorkOrderItemsCanChange(ctx context.Context, workOrderID uuid.UUID, operation string) (*domain.WorkOrder, error) {
	wo, err := s.woRepo.FindByID(ctx, workOrderID)
	if err != nil {
		return nil, fmt.Errorf("%s: find work order: %w", operation, err)
	}
	if !canChangeWorkOrderItems(wo.Status) {
		return nil, ErrWorkOrderInvalidStatusForItems
	}
	return wo, nil
}

func (s *workOrderCreationService) refreshBudgetIfWaitingApproval(ctx context.Context, workOrderID uuid.UUID, operation string) error {
	if s.budget == nil {
		return nil
	}

	wo, err := s.woRepo.FindByID(ctx, workOrderID)
	if err != nil {
		return fmt.Errorf("%s: refresh budget: find work order: %w", operation, err)
	}
	if wo.Status != domain.WorkOrderStatusWaitingApproval {
		return nil
	}
	if err := s.budget.GenerateAndSendBudget(ctx, workOrderID, nil); err != nil {
		return fmt.Errorf("%s: refresh budget: %w", operation, err)
	}
	return nil
}

func (s *workOrderCreationService) AddServices(ctx context.Context, workOrderID uuid.UUID, items []AddWorkOrderServiceInput) ([]domain.WorkOrderService, error) {
	wo, err := s.ensureWorkOrderItemsCanChange(ctx, workOrderID, "add services")
	if err != nil {
		return nil, err
	}

	batch := make([]*domain.WorkOrderService, 0, len(items))
	for _, input := range items {
		ws, err := s.wsRepo.FindByID(ctx, input.ServiceID)
		if err != nil {
			return nil, fmt.Errorf("add services: find service %s: %w", input.ServiceID, err)
		}
		if !ws.Active {
			return nil, ErrWorkshopServiceInactive
		}

		estimatedTime := ws.EstimatedTimeMinutes
		if input.EstimatedTimeMinutes != nil {
			estimatedTime = *input.EstimatedTimeMinutes
		}

		batch = append(batch, &domain.WorkOrderService{
			WorkOrderID:                         workOrderID,
			ServiceID:                           ws.ID,
			ServiceTitleSnapshot:                ws.Title,
			ServiceDescriptionSnapshot:          &ws.Description,
			ServicePriceCentsSnapshot:           ws.PriceCents,
			ServiceEstimatedTimeMinutesSnapshot: estimatedTime,
			ApprovalStatus:                      domain.WorkOrderServiceApprovalPending,
			Status:                              domain.WorkOrderServiceStatusPending,
		})
	}

	created, err := s.wosRepo.CreateBatch(ctx, batch)
	if err != nil {
		return nil, fmt.Errorf("add services: create batch: %w", err)
	}

	if wo.Status == domain.WorkOrderStatusReceived {
		if _, err := s.statusSvc.TransitionTo(ctx, workOrderID, domain.WorkOrderStatusInDiagnosis); err != nil {
			return nil, fmt.Errorf("add services: transition status: %w", err)
		}
	}

	if err := s.refreshBudgetIfWaitingApproval(ctx, workOrderID, "add services"); err != nil {
		return nil, err
	}

	result := make([]domain.WorkOrderService, len(created))
	for i, item := range created {
		result[i] = *item
	}
	return result, nil
}

func (s *workOrderCreationService) RemoveService(ctx context.Context, workOrderID, wosID uuid.UUID) error {
	wos, err := s.wosRepo.FindByID(ctx, wosID)
	if err != nil {
		return fmt.Errorf("remove service: find: %w", err)
	}

	if wos.WorkOrderID != workOrderID {
		return ErrWorkOrderServiceOwnership
	}

	if _, err := s.ensureWorkOrderItemsCanChange(ctx, workOrderID, "remove service"); err != nil {
		return err
	}

	if err := s.wosRepo.DeleteSuppliesByWorkOrderServiceID(ctx, wosID); err != nil {
		return fmt.Errorf("remove service: delete supplies: %w", err)
	}

	if err := s.wosRepo.DeleteByID(ctx, wosID); err != nil {
		return fmt.Errorf("remove service: delete: %w", err)
	}

	if err := s.refreshBudgetIfWaitingApproval(ctx, workOrderID, "remove service"); err != nil {
		return err
	}

	return nil
}

func (s *workOrderCreationService) RemoveSupplyFromService(ctx context.Context, workOrderID, wosID, supplyID uuid.UUID) error {
	wos, err := s.wosRepo.FindByID(ctx, wosID)
	if err != nil {
		return fmt.Errorf("remove supply: find work order service: %w", err)
	}
	if wos.WorkOrderID != workOrderID {
		return ErrWorkOrderServiceOwnership
	}

	if _, err := s.ensureWorkOrderItemsCanChange(ctx, workOrderID, "remove supply"); err != nil {
		return err
	}

	if err := s.wosRepo.DeleteSupplyForWorkOrderService(ctx, wosID, supplyID); err != nil {
		return fmt.Errorf("remove supply: delete: %w", err)
	}

	if err := s.refreshBudgetIfWaitingApproval(ctx, workOrderID, "remove supply"); err != nil {
		return err
	}

	return nil
}

func (s *workOrderCreationService) AddSupplies(ctx context.Context, workOrderID, wosID uuid.UUID, items []AddWorkOrderSupplyInput) ([]domain.WorkOrderServiceSupply, error) {
	wos, err := s.wosRepo.FindByID(ctx, wosID)
	if err != nil {
		return nil, fmt.Errorf("add supplies: find work order service: %w", err)
	}
	if wos.WorkOrderID != workOrderID {
		return nil, ErrWorkOrderServiceOwnership
	}

	if _, err := s.ensureWorkOrderItemsCanChange(ctx, workOrderID, "add supplies"); err != nil {
		return nil, err
	}

	batch := make([]*domain.WorkOrderServiceSupply, 0, len(items))
	for _, input := range items {
		supply, err := s.supplyRepo.FindByID(ctx, input.SupplyID)
		if err != nil {
			return nil, fmt.Errorf("add supplies: find supply %s: %w", input.SupplyID, err)
		}

		batch = append(batch, &domain.WorkOrderServiceSupply{
			WorkOrderServiceID:       wosID,
			SupplyID:                 supply.ID,
			SupplyTitleSnapshot:      supply.Title,
			SupplyPriceCentsSnapshot: supply.PriceCents,
			SupplyQuantity:           input.Quantity,
		})
	}

	created, err := s.wosRepo.CreateSupplyBatch(ctx, batch)
	if err != nil {
		return nil, fmt.Errorf("add supplies: create batch: %w", err)
	}

	result := make([]domain.WorkOrderServiceSupply, len(created))
	for i, item := range created {
		result[i] = *item
	}
	if err := s.refreshBudgetIfWaitingApproval(ctx, workOrderID, "add supplies"); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *workOrderCreationService) StartService(ctx context.Context, workOrderID, wosID uuid.UUID) error {
	wos, err := s.wosRepo.FindByID(ctx, wosID)
	if err != nil {
		return fmt.Errorf("start service: find: %w", err)
	}
	if wos.WorkOrderID != workOrderID {
		return ErrWorkOrderServiceOwnership
	}

	wo, err := s.woRepo.FindByID(ctx, workOrderID)
	if err != nil {
		return fmt.Errorf("start service: find work order: %w", err)
	}
	if wo.Status != domain.WorkOrderStatusInProgress {
		return ErrWorkOrderNotInProgress
	}
	if wos.ApprovalStatus != domain.WorkOrderServiceApprovalApproved {
		return ErrServiceNotApproved
	}
	if wos.Status != domain.WorkOrderServiceStatusPending {
		return ErrServiceNotPending
	}

	hasShortage, err := s.wosRepo.HasSupplyShortagesForService(ctx, wosID)
	if err != nil {
		return fmt.Errorf("start service: check stock: %w", err)
	}
	if hasShortage {
		return ErrInsufficientStock
	}

	if err := s.wosRepo.MarkServiceAsStarted(ctx, wosID, time.Now()); err != nil {
		return fmt.Errorf("start service: update: %w", err)
	}
	return nil
}

func (s *workOrderCreationService) FinalizeService(ctx context.Context, workOrderID, wosID uuid.UUID) error {
	wos, err := s.wosRepo.FindByID(ctx, wosID)
	if err != nil {
		return fmt.Errorf("finalize service: find: %w", err)
	}
	if wos.WorkOrderID != workOrderID {
		return ErrWorkOrderServiceOwnership
	}

	wo, err := s.woRepo.FindByID(ctx, workOrderID)
	if err != nil {
		return fmt.Errorf("finalize service: find work order: %w", err)
	}
	if wo.Status != domain.WorkOrderStatusInProgress {
		return ErrWorkOrderNotInProgress
	}
	if wos.Status != domain.WorkOrderServiceStatusInProgress {
		return ErrServiceNotInProgress
	}

	if err := s.wosRepo.MarkServiceAsFinished(ctx, wosID, time.Now()); err != nil {
		return fmt.Errorf("finalize service: update: %w", err)
	}

	// Decrement stock for supplies used in this service
	if err := s.supplyRepo.DecrementStockForService(ctx, wosID); err != nil {
		return fmt.Errorf("finalize service: decrement stock: %w", err)
	}

	// Check if all approved services are now finalized → auto-finalize WO
	services, err := s.wosRepo.FindByWorkOrderID(ctx, workOrderID)
	if err != nil {
		return fmt.Errorf("finalize service: check completion: %w", err)
	}

	allFinished := true
	for _, svc := range services {
		if svc.ApprovalStatus == domain.WorkOrderServiceApprovalApproved &&
			svc.Status != domain.WorkOrderServiceStatusFinished {
			allFinished = false
			break
		}
	}

	if allFinished {
		if _, err := s.statusSvc.TransitionTo(ctx, workOrderID, domain.WorkOrderStatusFinished); err != nil {
			return fmt.Errorf("finalize service: auto-transition: %w", err)
		}
	}

	return nil
}
