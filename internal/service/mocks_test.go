package service

import (
	"context"
	"time"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// mockWorkOrderRepo mocks repository.WorkOrderRepository
type mockWorkOrderRepo struct {
	mock.Mock
}

func (m *mockWorkOrderRepo) Create(ctx context.Context, wo *domain.WorkOrder) (*domain.WorkOrder, error) {
	args := m.Called(ctx, wo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkOrder), args.Error(1)
}

func (m *mockWorkOrderRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.WorkOrder, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkOrder), args.Error(1)
}

func (m *mockWorkOrderRepo) FindByCode(ctx context.Context, code string) (*domain.WorkOrder, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkOrder), args.Error(1)
}

func (m *mockWorkOrderRepo) FindAll(ctx context.Context) ([]domain.WorkOrder, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.WorkOrder), args.Error(1)
}

func (m *mockWorkOrderRepo) FindAllWithFilters(ctx context.Context, filters application.WorkOrderListFilters) (*application.WorkOrderListResponse, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*application.WorkOrderListResponse), args.Error(1)
}

func (m *mockWorkOrderRepo) Update(ctx context.Context, wo *domain.WorkOrder) (*domain.WorkOrder, error) {
	args := m.Called(ctx, wo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkOrder), args.Error(1)
}

func (m *mockWorkOrderRepo) TransitionStatus(ctx context.Context, input application.WorkOrderStatusTransitionInput) (*domain.WorkOrder, bool, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).(*domain.WorkOrder), args.Bool(1), args.Error(2)
}

// mockWorkOrderServiceRepo mocks repository.WorkOrderServiceRepository
type mockWorkOrderServiceRepo struct {
	mock.Mock
}

func (m *mockWorkOrderServiceRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.WorkOrderService, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkOrderService), args.Error(1)
}

func (m *mockWorkOrderServiceRepo) FindByWorkOrderID(ctx context.Context, workOrderID uuid.UUID) ([]domain.WorkOrderService, error) {
	args := m.Called(ctx, workOrderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.WorkOrderService), args.Error(1)
}

func (m *mockWorkOrderServiceRepo) UpdateApprovalStatus(ctx context.Context, id uuid.UUID, status domain.WorkOrderServiceApprovalStatus) error {
	return m.Called(ctx, id, status).Error(0)
}

func (m *mockWorkOrderServiceRepo) UpdateApprovalStatusByWorkOrderID(ctx context.Context, workOrderID uuid.UUID, status domain.WorkOrderServiceApprovalStatus) error {
	return m.Called(ctx, workOrderID, status).Error(0)
}

func (m *mockWorkOrderServiceRepo) CalculateTotalForWorkOrder(ctx context.Context, workOrderID uuid.UUID) (int, error) {
	args := m.Called(ctx, workOrderID)
	return args.Int(0), args.Error(1)
}

func (m *mockWorkOrderServiceRepo) CalculateApprovedTotalForWorkOrder(ctx context.Context, workOrderID uuid.UUID) (int, error) {
	args := m.Called(ctx, workOrderID)
	return args.Int(0), args.Error(1)
}

func (m *mockWorkOrderServiceRepo) Create(ctx context.Context, wos *domain.WorkOrderService) (*domain.WorkOrderService, error) {
	args := m.Called(ctx, wos)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkOrderService), args.Error(1)
}

func (m *mockWorkOrderServiceRepo) CreateBatch(ctx context.Context, items []*domain.WorkOrderService) ([]*domain.WorkOrderService, error) {
	args := m.Called(ctx, items)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.WorkOrderService), args.Error(1)
}

func (m *mockWorkOrderServiceRepo) CreateSupply(ctx context.Context, supply *domain.WorkOrderServiceSupply) (*domain.WorkOrderServiceSupply, error) {
	args := m.Called(ctx, supply)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkOrderServiceSupply), args.Error(1)
}

func (m *mockWorkOrderServiceRepo) CreateSupplyBatch(ctx context.Context, items []*domain.WorkOrderServiceSupply) ([]*domain.WorkOrderServiceSupply, error) {
	args := m.Called(ctx, items)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.WorkOrderServiceSupply), args.Error(1)
}

func (m *mockWorkOrderServiceRepo) DeleteSuppliesByWorkOrderServiceID(ctx context.Context, workOrderServiceID uuid.UUID) error {
	return m.Called(ctx, workOrderServiceID).Error(0)
}

func (m *mockWorkOrderServiceRepo) DeleteSupplyForWorkOrderService(ctx context.Context, workOrderServiceID, supplyID uuid.UUID) error {
	return m.Called(ctx, workOrderServiceID, supplyID).Error(0)
}

func (m *mockWorkOrderServiceRepo) DeleteByID(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockWorkOrderServiceRepo) MarkAsStartedByWorkOrderID(ctx context.Context, workOrderID uuid.UUID, startedAt time.Time) error {
	return m.Called(ctx, workOrderID, startedAt).Error(0)
}

func (m *mockWorkOrderServiceRepo) MarkAsFinishedByWorkOrderID(ctx context.Context, workOrderID uuid.UUID, finishedAt time.Time) error {
	return m.Called(ctx, workOrderID, finishedAt).Error(0)
}

func (m *mockWorkOrderServiceRepo) MarkServiceAsFinished(ctx context.Context, id uuid.UUID, finishedAt time.Time) error {
	return m.Called(ctx, id, finishedAt).Error(0)
}

func (m *mockWorkOrderServiceRepo) MarkServiceAsStarted(ctx context.Context, id uuid.UUID, startedAt time.Time) error {
	return m.Called(ctx, id, startedAt).Error(0)
}

func (m *mockWorkOrderServiceRepo) HasSupplyShortagesForService(ctx context.Context, workOrderServiceID uuid.UUID) (bool, error) {
	args := m.Called(ctx, workOrderServiceID)
	return args.Bool(0), args.Error(1)
}

func (m *mockWorkOrderServiceRepo) FindApprovedServicesWithShortages(ctx context.Context) ([]application.SupplyShortageAlert, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]application.SupplyShortageAlert), args.Error(1)
}

func (m *mockWorkOrderServiceRepo) FindSupplyShortagesByWorkOrderID(ctx context.Context, workOrderID uuid.UUID) (map[uuid.UUID]bool, error) {
	args := m.Called(ctx, workOrderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[uuid.UUID]bool), args.Error(1)
}

// mockVehicleRepo mocks repository.VehicleRepository
type mockVehicleRepo struct {
	mock.Mock
}

func (m *mockVehicleRepo) Create(ctx context.Context, vehicle *domain.Vehicle) (*domain.Vehicle, error) {
	args := m.Called(ctx, vehicle)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Vehicle), args.Error(1)
}

func (m *mockVehicleRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Vehicle, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Vehicle), args.Error(1)
}

func (m *mockVehicleRepo) FindAll(ctx context.Context) ([]domain.Vehicle, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Vehicle), args.Error(1)
}

func (m *mockVehicleRepo) FindAllWithFilters(ctx context.Context, filters domain.VehicleListFilters) ([]domain.Vehicle, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Vehicle), args.Error(1)
}

func (m *mockVehicleRepo) Update(ctx context.Context, vehicle *domain.Vehicle) (*domain.Vehicle, error) {
	args := m.Called(ctx, vehicle)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Vehicle), args.Error(1)
}

func (m *mockVehicleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// mockWorkshopServiceRepo mocks repository.WorkshopServiceRepository (minimal — only FindByID used)
type mockWorkshopServiceRepo struct {
	mock.Mock
}

func (m *mockWorkshopServiceRepo) Create(ctx context.Context, ws *domain.WorkshopService) (*domain.WorkshopService, error) {
	args := m.Called(ctx, ws)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkshopService), args.Error(1)
}

func (m *mockWorkshopServiceRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.WorkshopService, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkshopService), args.Error(1)
}

func (m *mockWorkshopServiceRepo) List(ctx context.Context, filters domain.WorkshopServiceListFilters) ([]domain.WorkshopService, int, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).([]domain.WorkshopService), args.Int(1), args.Error(2)
}

func (m *mockWorkshopServiceRepo) Update(ctx context.Context, ws *domain.WorkshopService) (*domain.WorkshopService, error) {
	args := m.Called(ctx, ws)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkshopService), args.Error(1)
}

func (m *mockWorkshopServiceRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockWorkshopServiceRepo) Deactivate(ctx context.Context, id uuid.UUID) (*domain.WorkshopService, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkshopService), args.Error(1)
}

func (m *mockWorkshopServiceRepo) ExistsByTitle(ctx context.Context, title string, excludeID *uuid.UUID) (bool, error) {
	args := m.Called(ctx, title, excludeID)
	return args.Bool(0), args.Error(1)
}

func (m *mockWorkshopServiceRepo) HasWorkOrderLinks(ctx context.Context, id uuid.UUID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

func (m *mockWorkshopServiceRepo) GetAvgExecutionTime(ctx context.Context, filters domain.AvgExecutionTimeFilters) ([]domain.AvgExecutionTimeResult, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).([]domain.AvgExecutionTimeResult), args.Error(1)
}

func (m *mockWorkshopServiceRepo) SubtractSuppliesFromStock(ctx context.Context, serviceID uuid.UUID) error {
	return m.Called(ctx, serviceID).Error(0)
}

// mockStatusService mocks WorkOrderStatusService
type mockStatusService struct {
	mock.Mock
}

func (m *mockStatusService) TransitionTo(ctx context.Context, workOrderID uuid.UUID, newStatus domain.WorkOrderStatus) (*domain.WorkOrder, error) {
	args := m.Called(ctx, workOrderID, newStatus)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkOrder), args.Error(1)
}

func (m *mockStatusService) IsValidTransition(from, to domain.WorkOrderStatus) bool {
	return m.Called(from, to).Bool(0)
}

type mockBudgetServiceUseCase struct {
	mock.Mock
}

func (m *mockBudgetServiceUseCase) GenerateAndSendBudget(ctx context.Context, workOrderID uuid.UUID, previousStatus *domain.WorkOrderStatus) error {
	return m.Called(ctx, workOrderID, previousStatus).Error(0)
}

// mockSupplyRepo mocks repository.SupplyRepository
type mockSupplyRepo struct {
	mock.Mock
}

func (m *mockSupplyRepo) Create(ctx context.Context, supply *domain.Supply) (*domain.Supply, error) {
	args := m.Called(ctx, supply)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Supply), args.Error(1)
}

func (m *mockSupplyRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Supply, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Supply), args.Error(1)
}

func (m *mockSupplyRepo) FindAll(ctx context.Context) ([]domain.Supply, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Supply), args.Error(1)
}

func (m *mockSupplyRepo) Update(ctx context.Context, supply *domain.Supply) (*domain.Supply, error) {
	args := m.Called(ctx, supply)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Supply), args.Error(1)
}

func (m *mockSupplyRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockSupplyRepo) DecrementStockForService(ctx context.Context, workOrderServiceID uuid.UUID) error {
	return m.Called(ctx, workOrderServiceID).Error(0)
}
