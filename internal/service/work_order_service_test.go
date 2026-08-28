package service

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var codePattern = regexp.MustCompile(`^OS-\d{8}-[0-9A-F]{4}$`)

func newWOInput(customerID, vehicleID, userID uuid.UUID) *domain.WorkOrder {
	return &domain.WorkOrder{
		Title:          "Revisão geral",
		CustomerID:     customerID,
		VehicleID:      vehicleID,
		OpenedByUserID: userID,
	}
}

func savedWO(id, customerID, vehicleID, userID uuid.UUID) *domain.WorkOrder {
	wo := newWOInput(customerID, vehicleID, userID)
	wo.ID = id
	wo.Code = "OS-20260504-AB12"
	wo.Status = domain.WorkOrderStatusReceived
	return wo
}

func TestCreate_AutoGeneratesCode(t *testing.T) {
	// code must be generated when not provided, matching OS-YYYYMMDD-XXXX
	woRepo := new(mockWorkOrderRepo)
	vehicleRepo := new(mockVehicleRepo)
	svc := NewWorkOrderService(woRepo, vehicleRepo)
	ctx := context.Background()

	customerID := uuid.New()
	vehicleID := uuid.New()
	userID := uuid.New()
	input := newWOInput(customerID, vehicleID, userID)

	vehicle := &domain.Vehicle{ID: vehicleID, CustomerID: customerID}
	result := savedWO(uuid.New(), customerID, vehicleID, userID)

	vehicleRepo.On("FindByID", ctx, vehicleID).Return(vehicle, nil)
	woRepo.On("Create", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(result, nil)
	woRepo.On("FindByID", ctx, result.ID).Return(result, nil)

	out, err := svc.Create(ctx, input)
	assert.NoError(t, err)
	assert.NotNil(t, out)

	// inspect the code that was set on the input before repo.Create was called
	call := woRepo.Calls[0]
	submitted := call.Arguments[1].(*domain.WorkOrder)
	assert.Regexp(t, codePattern, submitted.Code)
}

func TestCreate_VehicleNotBelongingToCustomer_ReturnsError(t *testing.T) {
	// must reject when vehicle.CustomerID != workOrder.CustomerID
	woRepo := new(mockWorkOrderRepo)
	vehicleRepo := new(mockVehicleRepo)
	svc := NewWorkOrderService(woRepo, vehicleRepo)
	ctx := context.Background()

	customerID := uuid.New()
	otherCustomerID := uuid.New()
	vehicleID := uuid.New()
	userID := uuid.New()
	input := newWOInput(customerID, vehicleID, userID)

	vehicle := &domain.Vehicle{ID: vehicleID, CustomerID: otherCustomerID}
	vehicleRepo.On("FindByID", ctx, vehicleID).Return(vehicle, nil)

	out, err := svc.Create(ctx, input)
	assert.ErrorIs(t, err, ErrVehicleNotBelongingToCustomer)
	assert.Nil(t, out)
	woRepo.AssertNotCalled(t, "Create")
}

func TestCreate_VehicleNotFound_ReturnsError(t *testing.T) {
	// must propagate error when vehicle does not exist
	woRepo := new(mockWorkOrderRepo)
	vehicleRepo := new(mockVehicleRepo)
	svc := NewWorkOrderService(woRepo, vehicleRepo)
	ctx := context.Background()

	customerID := uuid.New()
	vehicleID := uuid.New()
	userID := uuid.New()
	input := newWOInput(customerID, vehicleID, userID)

	vehicleRepo.On("FindByID", ctx, vehicleID).Return(nil, errors.New("not found"))

	out, err := svc.Create(ctx, input)
	assert.Error(t, err)
	assert.Nil(t, out)
	woRepo.AssertNotCalled(t, "Create")
}

func TestCreate_MissingTitle_ReturnsError(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	vehicleRepo := new(mockVehicleRepo)
	svc := NewWorkOrderService(woRepo, vehicleRepo)
	ctx := context.Background()

	input := &domain.WorkOrder{
		CustomerID:     uuid.New(),
		VehicleID:      uuid.New(),
		OpenedByUserID: uuid.New(),
	}

	out, err := svc.Create(ctx, input)
	assert.Error(t, err)
	assert.Nil(t, out)
}

func TestCreate_MissingCustomerID_ReturnsError(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	vehicleRepo := new(mockVehicleRepo)
	svc := NewWorkOrderService(woRepo, vehicleRepo)
	ctx := context.Background()

	input := &domain.WorkOrder{
		Title:          "Revisão",
		VehicleID:      uuid.New(),
		OpenedByUserID: uuid.New(),
	}

	out, err := svc.Create(ctx, input)
	assert.Error(t, err)
	assert.Nil(t, out)
}

func TestWorkOrderGetByID_Success(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	vehicleRepo := new(mockVehicleRepo)
	svc := NewWorkOrderService(woRepo, vehicleRepo)
	ctx := context.Background()
	id := uuid.New()
	wo := makeWO(id, domain.WorkOrderStatusReceived)

	woRepo.On("FindByID", ctx, id).Return(wo, nil)

	result, err := svc.GetByID(ctx, id)
	assert.NoError(t, err)
	assert.Equal(t, id, result.ID)
}

func TestWorkOrderGetByID_NotFound(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	vehicleRepo := new(mockVehicleRepo)
	svc := NewWorkOrderService(woRepo, vehicleRepo)
	ctx := context.Background()
	id := uuid.New()

	woRepo.On("FindByID", ctx, id).Return(nil, application.ErrNotFound)

	result, err := svc.GetByID(ctx, id)
	assert.ErrorIs(t, err, application.ErrNotFound)
	assert.Nil(t, result)
}

func TestWorkOrderGetAll_Success(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	vehicleRepo := new(mockVehicleRepo)
	svc := NewWorkOrderService(woRepo, vehicleRepo)
	ctx := context.Background()
	id := uuid.New()

	woRepo.On("FindAll", ctx).Return([]domain.WorkOrder{*makeWO(id, domain.WorkOrderStatusReceived)}, nil)

	results, err := svc.GetAll(ctx)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestWorkOrderUpdate_Success(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	vehicleRepo := new(mockVehicleRepo)
	svc := NewWorkOrderService(woRepo, vehicleRepo)
	ctx := context.Background()
	id := uuid.New()

	existing := makeWO(id, domain.WorkOrderStatusReceived)
	existing.Title = "Revisão Original"
	updated := makeWO(id, domain.WorkOrderStatusInDiagnosis)

	woRepo.On("FindByID", ctx, id).Return(existing, nil).Once()
	woRepo.On("Update", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(updated, nil)
	woRepo.On("FindByID", ctx, id).Return(updated, nil).Once()

	input := &domain.WorkOrder{ID: id, Title: "Revisão Atualizada", Status: domain.WorkOrderStatusInDiagnosis}
	result, err := svc.Update(ctx, input)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWorkOrderUpdate_NotFound(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	vehicleRepo := new(mockVehicleRepo)
	svc := NewWorkOrderService(woRepo, vehicleRepo)
	ctx := context.Background()
	id := uuid.New()

	woRepo.On("FindByID", ctx, id).Return(nil, application.ErrNotFound)

	result, err := svc.Update(ctx, &domain.WorkOrder{ID: id})
	assert.ErrorIs(t, err, application.ErrNotFound)
	assert.Nil(t, result)
}

func TestCreate_MissingOpenedByUserID_ReturnsError(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	vehicleRepo := new(mockVehicleRepo)
	svc := NewWorkOrderService(woRepo, vehicleRepo)
	ctx := context.Background()

	input := &domain.WorkOrder{
		Title:      "Revisão",
		CustomerID: uuid.New(),
		VehicleID:  uuid.New(),
	}

	out, err := svc.Create(ctx, input)
	assert.Error(t, err)
	assert.Nil(t, out)
}

func TestCreate_VehicleNotBelongingToCustomer(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	vehicleRepo := new(mockVehicleRepo)
	svc := NewWorkOrderService(woRepo, vehicleRepo)
	ctx := context.Background()

	customerID := uuid.New()
	vehicleID := uuid.New()
	input := &domain.WorkOrder{
		Title:          "Revisão",
		CustomerID:     customerID,
		VehicleID:      vehicleID,
		OpenedByUserID: uuid.New(),
	}
	vehicle := &domain.Vehicle{ID: vehicleID, CustomerID: uuid.New()} // diferente customerID

	vehicleRepo.On("FindByID", ctx, vehicleID).Return(vehicle, nil)

	out, err := svc.Create(ctx, input)
	assert.ErrorIs(t, err, ErrVehicleNotBelongingToCustomer)
	assert.Nil(t, out)
}

func TestWorkOrderUpdate_AllFields(t *testing.T) {
	// cobre todos os branches opcionais do Update
	woRepo := new(mockWorkOrderRepo)
	vehicleRepo := new(mockVehicleRepo)
	svc := NewWorkOrderService(woRepo, vehicleRepo)
	ctx := context.Background()
	id := uuid.New()

	existing := makeWO(id, domain.WorkOrderStatusReceived)
	updated := makeWO(id, domain.WorkOrderStatusInDiagnosis)

	desc := "nova descrição"
	techID := uuid.New()
	custID := uuid.New()
	vehID := uuid.New()
	now := time.Now()

	input := &domain.WorkOrder{
		ID:                       id,
		Code:                     "OS-NOVO",
		Title:                    "Revisão Completa",
		Description:              &desc,
		CustomerID:               custID,
		VehicleID:                vehID,
		AssignedTechnicianID:     &techID,
		Status:                   domain.WorkOrderStatusInDiagnosis,
		TotalEstimatedPriceCents: 9999,
		ReceivedAt:               now,
		QuoteSentAt:              &now,
		ApprovedAt:               &now,
		StartedAt:                &now,
		FinishedAt:               &now,
		DeliveredAt:              &now,
	}

	woRepo.On("FindByID", ctx, id).Return(existing, nil).Once()
	woRepo.On("Update", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(updated, nil)
	woRepo.On("FindByID", ctx, id).Return(updated, nil).Once()

	result, err := svc.Update(ctx, input)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllWithFilters_DefaultsPage(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	vehicleRepo := new(mockVehicleRepo)
	svc := NewWorkOrderService(woRepo, vehicleRepo)
	ctx := context.Background()

	resp := &application.WorkOrderListResponse{
		Data:  []domain.WorkOrder{},
		Total: 0, Page: 1, Limit: 10, TotalPages: 0,
	}
	woRepo.On("FindAllWithFilters", ctx, mock.MatchedBy(func(f application.WorkOrderListFilters) bool {
		return f.Page == 1 && f.Limit == 10
	})).Return(resp, nil)

	result, err := svc.GetAllWithFilters(ctx, application.WorkOrderListFilters{Page: 0, Limit: 0})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllWithFilters_LimitTooHigh(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	vehicleRepo := new(mockVehicleRepo)
	svc := NewWorkOrderService(woRepo, vehicleRepo)
	ctx := context.Background()

	resp := &application.WorkOrderListResponse{
		Data:  []domain.WorkOrder{},
		Total: 0, Page: 1, Limit: 10, TotalPages: 0,
	}
	woRepo.On("FindAllWithFilters", ctx, mock.MatchedBy(func(f application.WorkOrderListFilters) bool {
		return f.Limit == 10
	})).Return(resp, nil)

	result, err := svc.GetAllWithFilters(ctx, application.WorkOrderListFilters{Page: 1, Limit: 200})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllWithFilters_ValidFilters(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	vehicleRepo := new(mockVehicleRepo)
	svc := NewWorkOrderService(woRepo, vehicleRepo)
	ctx := context.Background()

	resp := &application.WorkOrderListResponse{
		Data:  []domain.WorkOrder{*makeWO(uuid.New(), domain.WorkOrderStatusReceived)},
		Total: 1, Page: 2, Limit: 5, TotalPages: 1,
	}
	woRepo.On("FindAllWithFilters", ctx, mock.MatchedBy(func(f application.WorkOrderListFilters) bool {
		return f.Page == 2 && f.Limit == 5
	})).Return(resp, nil)

	result, err := svc.GetAllWithFilters(ctx, application.WorkOrderListFilters{Page: 2, Limit: 5})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(result.Data))
}

func TestGetAllWithFilters_Error(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	vehicleRepo := new(mockVehicleRepo)
	svc := NewWorkOrderService(woRepo, vehicleRepo)
	ctx := context.Background()

	woRepo.On("FindAllWithFilters", ctx, mock.Anything).Return(nil, errors.New("db error"))

	result, err := svc.GetAllWithFilters(ctx, application.WorkOrderListFilters{Page: 1, Limit: 10})
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestGetAllWithFilters_NoStatusFilter_PassesThroughToRepo(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	vehicleRepo := new(mockVehicleRepo)
	svc := NewWorkOrderService(woRepo, vehicleRepo)
	ctx := context.Background()

	resp := &application.WorkOrderListResponse{
		Data: []domain.WorkOrder{}, Total: 0, Page: 1, Limit: 10, TotalPages: 0,
	}
	woRepo.On("FindAllWithFilters", ctx, mock.MatchedBy(func(f application.WorkOrderListFilters) bool {
		return f.Status == "" && f.Page == 1 && f.Limit == 10
	})).Return(resp, nil)

	result, err := svc.GetAllWithFilters(ctx, application.WorkOrderListFilters{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	woRepo.AssertExpectations(t)
}

func TestCreate_MissingVehicleID_ReturnsError(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	vehicleRepo := new(mockVehicleRepo)
	svc := NewWorkOrderService(woRepo, vehicleRepo)
	ctx := context.Background()

	input := &domain.WorkOrder{
		Title:          "Revisão",
		CustomerID:     uuid.New(),
		OpenedByUserID: uuid.New(),
	}

	out, err := svc.Create(ctx, input)
	assert.Error(t, err)
	assert.Nil(t, out)
}
