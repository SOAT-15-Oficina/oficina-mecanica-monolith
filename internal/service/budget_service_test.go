package service

import (
	"context"
	"errors"
	"testing"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockBudgetNotifier struct {
	mock.Mock
}

func (m *mockBudgetNotifier) SendBudget(ctx context.Context, notification application.BudgetNotification) error {
	return m.Called(ctx, notification).Error(0)
}

func newBudgetService(woRepo *mockWorkOrderRepo, wosRepo *mockWorkOrderServiceRepo, custRepo *mockCustomerRepo, notifier *mockBudgetNotifier) BudgetService {
	return NewBudgetService(woRepo, wosRepo, custRepo, notifier, "http://localhost:3000")
}

func TestGenerateAndSendBudget_Success(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	custRepo := new(mockCustomerRepo)
	notifier := new(mockBudgetNotifier)
	svc := newBudgetService(woRepo, wosRepo, custRepo, notifier)
	ctx := context.Background()

	woID := uuid.New()
	custID := uuid.New()
	previous := domain.WorkOrderStatusInDiagnosis

	svcItem := domain.WorkOrderService{
		ID:                        uuid.New(),
		WorkOrderID:               woID,
		ServiceTitleSnapshot:      "Troca de óleo",
		ServicePriceCentsSnapshot: 5000,
	}
	wo := &domain.WorkOrder{
		ID:         woID,
		Code:       "WO-001",
		CustomerID: custID,
	}
	customer := &domain.Customer{
		ID:    custID,
		Name:  "Maria",
		Email: "maria@example.com",
	}

	wosRepo.On("FindByWorkOrderID", ctx, woID).Return([]domain.WorkOrderService{svcItem}, nil)
	wosRepo.On("FindSupplyShortagesByWorkOrderID", ctx, woID).Return(map[uuid.UUID]bool{}, nil)
	wosRepo.On("CalculateTotalForWorkOrder", ctx, woID).Return(5000, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	custRepo.On("FindByID", ctx, custID).Return(customer, nil)
	notifier.On("SendBudget", ctx, mock.AnythingOfType("application.BudgetNotification")).Return(nil)
	woRepo.On("Update", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)

	err := svc.GenerateAndSendBudget(ctx, woID, &previous)
	require.NoError(t, err)
	notifier.AssertExpectations(t)
	woRepo.AssertCalled(t, "Update", ctx, mock.AnythingOfType("*domain.WorkOrder"))
}

func TestGenerateAndSendBudget_FindServicesFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	custRepo := new(mockCustomerRepo)
	notifier := new(mockBudgetNotifier)
	svc := newBudgetService(woRepo, wosRepo, custRepo, notifier)
	ctx := context.Background()
	woID := uuid.New()

	wosRepo.On("FindByWorkOrderID", ctx, woID).Return(nil, errors.New("db error"))

	err := svc.GenerateAndSendBudget(ctx, woID, nil)
	assert.Error(t, err)
	notifier.AssertNotCalled(t, "SendBudget")
}

func TestGenerateAndSendBudget_EmailFails_BestEffort(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	custRepo := new(mockCustomerRepo)
	notifier := new(mockBudgetNotifier)
	svc := newBudgetService(woRepo, wosRepo, custRepo, notifier)
	ctx := context.Background()

	woID := uuid.New()
	custID := uuid.New()

	wo := &domain.WorkOrder{ID: woID, Code: "WO-001", CustomerID: custID}
	customer := &domain.Customer{ID: custID, Name: "Maria", Email: "maria@example.com"}

	wosRepo.On("FindByWorkOrderID", ctx, woID).Return([]domain.WorkOrderService{}, nil)
	wosRepo.On("FindSupplyShortagesByWorkOrderID", ctx, woID).Return(map[uuid.UUID]bool{}, nil)
	wosRepo.On("CalculateTotalForWorkOrder", ctx, woID).Return(0, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	custRepo.On("FindByID", ctx, custID).Return(customer, nil)
	notifier.On("SendBudget", ctx, mock.AnythingOfType("application.BudgetNotification")).Return(errors.New("smtp error"))

	err := svc.GenerateAndSendBudget(ctx, woID, nil)
	require.NoError(t, err)
	woRepo.AssertNotCalled(t, "Update")
}

func TestGenerateAndSendBudget_AddsTwoDaysWhenSupplyIsShort(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	custRepo := new(mockCustomerRepo)
	notifier := new(mockBudgetNotifier)
	svc := newBudgetService(woRepo, wosRepo, custRepo, notifier)
	ctx := context.Background()

	woID := uuid.New()
	custID := uuid.New()
	serviceID := uuid.New()

	svcItem := domain.WorkOrderService{
		ID:                                  serviceID,
		WorkOrderID:                         woID,
		ServiceTitleSnapshot:                "Troca de filtro",
		ServicePriceCentsSnapshot:           7000,
		ServiceEstimatedTimeMinutesSnapshot: 60,
	}
	wo := &domain.WorkOrder{ID: woID, Code: "WO-002", CustomerID: custID}
	customer := &domain.Customer{ID: custID, Name: "Maria", Email: "maria@example.com"}

	wosRepo.On("FindByWorkOrderID", ctx, woID).Return([]domain.WorkOrderService{svcItem}, nil)
	wosRepo.On("FindSupplyShortagesByWorkOrderID", ctx, woID).Return(map[uuid.UUID]bool{serviceID: true}, nil)
	wosRepo.On("CalculateTotalForWorkOrder", ctx, woID).Return(7000, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	custRepo.On("FindByID", ctx, custID).Return(customer, nil)
	notifier.On("SendBudget", ctx, mock.AnythingOfType("application.BudgetNotification")).Return(nil)
	woRepo.On("Update", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)

	err := svc.GenerateAndSendBudget(ctx, woID, nil)
	require.NoError(t, err)

	notification := notifier.Calls[0].Arguments.Get(1).(application.BudgetNotification)
	assert.Equal(t, "2 dias e 1 hora", notification.Services[0].Estimated)
}

func TestGenerateAndSendBudget_IncludesStatusAndApprovalLinks(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	custRepo := new(mockCustomerRepo)
	notifier := new(mockBudgetNotifier)
	svc := newBudgetService(woRepo, wosRepo, custRepo, notifier)
	ctx := context.Background()

	woID := uuid.New()
	custID := uuid.New()
	wosID := uuid.New()
	previous := domain.WorkOrderStatusInDiagnosis

	svcItem := domain.WorkOrderService{
		ID:                        wosID,
		WorkOrderID:               woID,
		ServiceTitleSnapshot:      "Alinhamento",
		ServicePriceCentsSnapshot: 12000,
	}
	wo := &domain.WorkOrder{ID: woID, Code: "WO-010", CustomerID: custID}
	customer := &domain.Customer{ID: custID, Name: "João", Email: "joao@example.com"}

	wosRepo.On("FindByWorkOrderID", ctx, woID).Return([]domain.WorkOrderService{svcItem}, nil)
	wosRepo.On("FindSupplyShortagesByWorkOrderID", ctx, woID).Return(map[uuid.UUID]bool{}, nil)
	wosRepo.On("CalculateTotalForWorkOrder", ctx, woID).Return(12000, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	custRepo.On("FindByID", ctx, custID).Return(customer, nil)
	notifier.On("SendBudget", ctx, mock.AnythingOfType("application.BudgetNotification")).Return(nil)
	woRepo.On("Update", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)

	err := svc.GenerateAndSendBudget(ctx, woID, &previous)
	require.NoError(t, err)

	notification := notifier.Calls[0].Arguments.Get(1).(application.BudgetNotification)
	assert.Equal(t, "WO-010", notification.WorkOrderCode)
	assert.Equal(t, "Em diagnóstico", notification.PreviousStatusLabel)
	assert.Equal(t, "Aguardando aprovação", notification.NewStatusLabel)
	assert.Contains(t, notification.Services[0].ApproveLink, wosID.String()+"/approve")
	assert.Contains(t, notification.Services[0].RejectLink, wosID.String()+"/reject")
	assert.Contains(t, notification.ApproveAllLink, woID.String()+"/approve-all")
	assert.Contains(t, notification.RejectAllLink, woID.String()+"/reject-all")
}

func TestGenerateAndSendBudget_FindSupplyShortagesFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	custRepo := new(mockCustomerRepo)
	notifier := new(mockBudgetNotifier)
	svc := newBudgetService(woRepo, wosRepo, custRepo, notifier)
	ctx := context.Background()
	woID := uuid.New()

	wosRepo.On("FindByWorkOrderID", ctx, woID).Return([]domain.WorkOrderService{}, nil)
	wosRepo.On("FindSupplyShortagesByWorkOrderID", ctx, woID).Return(nil, errors.New("db error"))

	err := svc.GenerateAndSendBudget(ctx, woID, nil)
	assert.Error(t, err)
	notifier.AssertNotCalled(t, "SendBudget")
}

func TestFormatCents(t *testing.T) {
	tests := []struct {
		cents    int
		expected string
	}{
		{0, "R$ 0,00"},
		{100, "R$ 1,00"},
		{5050, "R$ 50,50"},
		{10099, "R$ 100,99"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatCents(tt.cents))
		})
	}
}

func TestFormatEstimatedTimeMinutes(t *testing.T) {
	tests := []struct {
		minutes  int
		expected string
	}{
		{0, "0 min"},
		{1, "1 min"},
		{60, "1 hora"},
		{61, "1 hora e 1 min"},
		{1440, "1 dia"},
		{1500, "1 dia e 1 hora"},
		{2940, "2 dias e 1 hora"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatEstimatedTimeMinutes(tt.minutes))
		})
	}
}
