package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/google/uuid"
)

var ErrVehicleNotBelongingToCustomer = errors.New("vehicle does not belong to the given customer")

type WorkOrderService interface {
	Create(ctx context.Context, workOrder *domain.WorkOrder) (*domain.WorkOrder, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.WorkOrder, error)
	GetAll(ctx context.Context) ([]domain.WorkOrder, error)
	GetAllWithFilters(ctx context.Context, filters application.WorkOrderListFilters) (*application.WorkOrderListResponse, error)
	Update(ctx context.Context, workOrder *domain.WorkOrder) (*domain.WorkOrder, error)
}

type workOrderService struct {
	repo        application.WorkOrderRepository
	vehicleRepo application.VehicleRepository
}

func NewWorkOrderService(repo application.WorkOrderRepository, vehicleRepo application.VehicleRepository) WorkOrderService {
	return &workOrderService{repo: repo, vehicleRepo: vehicleRepo}
}

func generateWorkOrderCode() string {
	date := time.Now().Format("20060102")
	suffix := fmt.Sprintf("%04X", rand.Intn(0x10000))
	return fmt.Sprintf("OS-%s-%s", date, suffix)
}

func (s *workOrderService) Create(ctx context.Context, wo *domain.WorkOrder) (*domain.WorkOrder, error) {
	if wo.Title == "" {
		return nil, application.NewValidationError("title is required")
	}
	if wo.CustomerID == uuid.Nil {
		return nil, application.NewValidationError("customer_id is required")
	}
	if wo.VehicleID == uuid.Nil {
		return nil, application.NewValidationError("vehicle_id is required")
	}
	if wo.OpenedByUserID == uuid.Nil {
		return nil, application.NewValidationError("opened_by_user_id is required")
	}

	vehicle, err := s.vehicleRepo.FindByID(ctx, wo.VehicleID)
	if err != nil {
		return nil, fmt.Errorf("validate vehicle: %w", err)
	}
	if vehicle.CustomerID != wo.CustomerID {
		return nil, ErrVehicleNotBelongingToCustomer
	}

	if wo.ID == uuid.Nil {
		wo.ID = uuid.New()
	}
	if wo.Code == "" {
		wo.Code = generateWorkOrderCode()
	}
	if wo.Status == "" {
		wo.Status = domain.WorkOrderStatusReceived
	}

	now := time.Now()
	wo.CreatedAt = now
	wo.UpdatedAt = now
	if wo.ReceivedAt.IsZero() {
		wo.ReceivedAt = now
	}

	created, err := s.repo.Create(ctx, wo)
	if err != nil {
		return nil, err
	}
	return s.repo.FindByID(ctx, created.ID)
}

func (s *workOrderService) GetByID(ctx context.Context, id uuid.UUID) (*domain.WorkOrder, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *workOrderService) GetAll(ctx context.Context) ([]domain.WorkOrder, error) {
	return s.repo.FindAll(ctx)
}

func (s *workOrderService) Update(ctx context.Context, wo *domain.WorkOrder) (*domain.WorkOrder, error) {
	existing, err := s.repo.FindByID(ctx, wo.ID)
	if err != nil {
		return nil, err
	}
	wo.CreatedAt = existing.CreatedAt
	wo.UpdatedAt = time.Now()

	if wo.Code != "" {
		existing.Code = wo.Code
	}
	if wo.Title != "" {
		existing.Title = wo.Title
	}
	if wo.Description != nil {
		existing.Description = wo.Description
	}
	if wo.CustomerID != uuid.Nil {
		existing.CustomerID = wo.CustomerID
	}
	if wo.VehicleID != uuid.Nil {
		existing.VehicleID = wo.VehicleID
	}
	if wo.AssignedTechnicianID != nil {
		existing.AssignedTechnicianID = wo.AssignedTechnicianID
	}
	if wo.TotalEstimatedPriceCents != 0 {
		existing.TotalEstimatedPriceCents = wo.TotalEstimatedPriceCents
	}
	if !wo.ReceivedAt.IsZero() {
		existing.ReceivedAt = wo.ReceivedAt
	}
	if wo.QuoteSentAt != nil {
		existing.QuoteSentAt = wo.QuoteSentAt
	}
	if wo.ApprovedAt != nil {
		existing.ApprovedAt = wo.ApprovedAt
	}
	if wo.StartedAt != nil {
		existing.StartedAt = wo.StartedAt
	}
	if wo.FinishedAt != nil {
		existing.FinishedAt = wo.FinishedAt
	}
	if wo.DeliveredAt != nil {
		existing.DeliveredAt = wo.DeliveredAt
	}

	updated, err := s.repo.Update(ctx, existing)
	if err != nil {
		return nil, err
	}
	return s.repo.FindByID(ctx, updated.ID)
}

func (s *workOrderService) GetAllWithFilters(ctx context.Context, filters application.WorkOrderListFilters) (*application.WorkOrderListResponse, error) {
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.Limit < 1 || filters.Limit > 100 {
		filters.Limit = 10
	}

	return s.repo.FindAllWithFilters(ctx, filters)
}
