package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockStatusSender struct {
	mock.Mock
}

func (m *mockStatusSender) SendStatusChange(ctx context.Context, notification application.StatusChangeNotification) error {
	return m.Called(ctx, notification).Error(0)
}

func TestWorkOrderStatusNotifier_SendsStatusEmail(t *testing.T) {
	custRepo := new(mockCustomerRepo)
	statusSender := new(mockStatusSender)
	budgetSvc := new(mockBudgetServiceUseCase)
	notifier := NewWorkOrderStatusNotifier(custRepo, statusSender, budgetSvc)
	ctx := context.Background()

	custID := uuid.New()
	wo := &domain.WorkOrder{
		ID:         uuid.New(),
		Code:       "WO-100",
		CustomerID: custID,
		Status:     domain.WorkOrderStatusInProgress,
	}
	customer := &domain.Customer{ID: custID, Name: "Ana", Email: "ana@example.com"}

	custRepo.On("FindByID", ctx, custID).Return(customer, nil)
	statusSender.On("SendStatusChange", ctx, mock.AnythingOfType("application.StatusChangeNotification")).Return(nil)

	notifier.NotifyTransition(ctx, wo, domain.WorkOrderStatusApproved)

	statusSender.AssertExpectations(t)
	notification := statusSender.Calls[0].Arguments.Get(1).(application.StatusChangeNotification)
	assert.Equal(t, "ana@example.com", notification.CustomerEmail)
	assert.Equal(t, "WO-100", notification.WorkOrderCode)
	assert.Equal(t, "Aprovada", notification.PreviousStatusLabel)
	assert.Equal(t, "Em execução", notification.NewStatusLabel)
}

func TestWorkOrderStatusNotifier_WaitingApprovalUsesBudgetService(t *testing.T) {
	custRepo := new(mockCustomerRepo)
	statusSender := new(mockStatusSender)
	budgetSvc := new(mockBudgetServiceUseCase)
	notifier := NewWorkOrderStatusNotifier(custRepo, statusSender, budgetSvc)
	ctx := context.Background()

	woID := uuid.New()
	wo := &domain.WorkOrder{
		ID:     woID,
		Status: domain.WorkOrderStatusWaitingApproval,
	}

	budgetSvc.On("GenerateAndSendBudget", ctx, woID, mock.AnythingOfType("*domain.WorkOrderStatus")).Return(nil)

	notifier.NotifyTransition(ctx, wo, domain.WorkOrderStatusInDiagnosis)

	budgetSvc.AssertExpectations(t)
	statusSender.AssertNotCalled(t, "SendStatusChange")
}

func TestWorkOrderStatusNotifier_CanceledMessage(t *testing.T) {
	custRepo := new(mockCustomerRepo)
	statusSender := new(mockStatusSender)
	budgetSvc := new(mockBudgetServiceUseCase)
	notifier := NewWorkOrderStatusNotifier(custRepo, statusSender, budgetSvc)
	ctx := context.Background()

	custID := uuid.New()
	wo := &domain.WorkOrder{
		ID:         uuid.New(),
		Code:       "WO-200",
		CustomerID: custID,
		Status:     domain.WorkOrderStatusCanceled,
	}
	customer := &domain.Customer{ID: custID, Name: "Pedro", Email: "pedro@example.com"}

	custRepo.On("FindByID", ctx, custID).Return(customer, nil)
	statusSender.On("SendStatusChange", ctx, mock.AnythingOfType("application.StatusChangeNotification")).Return(nil)

	notifier.NotifyTransition(ctx, wo, domain.WorkOrderStatusWaitingApproval)

	notification := statusSender.Calls[0].Arguments.Get(1).(application.StatusChangeNotification)
	assert.True(t, strings.Contains(notification.Message, "recusado"))
}

func TestWorkOrderStatusNotifier_EmailFailureIsBestEffort(t *testing.T) {
	custRepo := new(mockCustomerRepo)
	statusSender := new(mockStatusSender)
	budgetSvc := new(mockBudgetServiceUseCase)
	notifier := NewWorkOrderStatusNotifier(custRepo, statusSender, budgetSvc)
	ctx := context.Background()

	custID := uuid.New()
	wo := &domain.WorkOrder{
		ID:         uuid.New(),
		Code:       "WO-300",
		CustomerID: custID,
		Status:     domain.WorkOrderStatusDelivered,
	}
	customer := &domain.Customer{ID: custID, Name: "Luiza", Email: "luiza@example.com"}

	custRepo.On("FindByID", ctx, custID).Return(customer, nil)
	statusSender.On("SendStatusChange", ctx, mock.AnythingOfType("application.StatusChangeNotification")).Return(errors.New("smtp down"))

	require.NotPanics(t, func() {
		notifier.NotifyTransition(ctx, wo, domain.WorkOrderStatusFinished)
	})
}
