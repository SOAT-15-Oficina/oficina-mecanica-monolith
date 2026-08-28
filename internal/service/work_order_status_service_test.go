package service

import (
	"context"
	"testing"
	"time"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTransitionTo_ValidTransition_UpdatesStatus(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	svc := NewWorkOrderStatusService(woRepo, nil)
	ctx := context.Background()

	woID := uuid.New()
	wo := &domain.WorkOrder{
		ID:        woID,
		Status:    domain.WorkOrderStatusReceived,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	updated := &domain.WorkOrder{
		ID:     woID,
		Status: domain.WorkOrderStatusInDiagnosis,
	}

	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	woRepo.On("TransitionStatus", ctx, mock.AnythingOfType("application.WorkOrderStatusTransitionInput")).Return(updated, true, nil)

	result, err := svc.TransitionTo(ctx, woID, domain.WorkOrderStatusInDiagnosis)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, domain.WorkOrderStatusInDiagnosis, result.Status)
}

func TestTransitionTo_InvalidTransition_ReturnsError(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	svc := NewWorkOrderStatusService(woRepo, nil)
	ctx := context.Background()

	woID := uuid.New()
	wo := &domain.WorkOrder{ID: woID, Status: domain.WorkOrderStatusDelivered}

	woRepo.On("FindByID", ctx, woID).Return(wo, nil)

	_, err := svc.TransitionTo(ctx, woID, domain.WorkOrderStatusReceived)
	assert.ErrorIs(t, err, ErrInvalidStatusTransition)
	woRepo.AssertNotCalled(t, "TransitionStatus")
}

func TestTransitionTo_SameStatus_Idempotent(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	svc := NewWorkOrderStatusService(woRepo, nil)
	ctx := context.Background()

	woID := uuid.New()
	wo := &domain.WorkOrder{ID: woID, Status: domain.WorkOrderStatusReceived}

	woRepo.On("FindByID", ctx, woID).Return(wo, nil)

	result, err := svc.TransitionTo(ctx, woID, domain.WorkOrderStatusReceived)
	assert.NoError(t, err)
	assert.Equal(t, wo, result)
	woRepo.AssertNotCalled(t, "TransitionStatus")
}

func TestTransitionTo_SetsTimestamps(t *testing.T) {
	tests := []struct {
		name         string
		from         domain.WorkOrderStatus
		to           domain.WorkOrderStatus
		checkFunc    func(t *testing.T, input application.WorkOrderStatusTransitionInput)
	}{
		{
			name: "approved sets approved_at",
			from: domain.WorkOrderStatusWaitingApproval,
			to:   domain.WorkOrderStatusApproved,
			checkFunc: func(t *testing.T, input application.WorkOrderStatusTransitionInput) {
				assert.Equal(t, domain.WorkOrderStatusApproved, input.ToStatus)
			},
		},
		{
			name: "in_progress sets started_at",
			from: domain.WorkOrderStatusApproved,
			to:   domain.WorkOrderStatusInProgress,
			checkFunc: func(t *testing.T, input application.WorkOrderStatusTransitionInput) {
				assert.Equal(t, domain.WorkOrderStatusInProgress, input.ToStatus)
			},
		},
		{
			name: "finished sets finished_at",
			from: domain.WorkOrderStatusInProgress,
			to:   domain.WorkOrderStatusFinished,
			checkFunc: func(t *testing.T, input application.WorkOrderStatusTransitionInput) {
				assert.Equal(t, domain.WorkOrderStatusFinished, input.ToStatus)
			},
		},
		{
			name: "delivered sets delivered_at",
			from: domain.WorkOrderStatusFinished,
			to:   domain.WorkOrderStatusDelivered,
			checkFunc: func(t *testing.T, input application.WorkOrderStatusTransitionInput) {
				assert.Equal(t, domain.WorkOrderStatusDelivered, input.ToStatus)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			woRepo := new(mockWorkOrderRepo)
			svc := NewWorkOrderStatusService(woRepo, nil)
			ctx := context.Background()

			woID := uuid.New()
			wo := &domain.WorkOrder{ID: woID, Status: tt.from}
			updated := &domain.WorkOrder{ID: woID, Status: tt.to}

			woRepo.On("FindByID", ctx, woID).Return(wo, nil)
			woRepo.On("TransitionStatus", ctx, mock.AnythingOfType("application.WorkOrderStatusTransitionInput")).Return(updated, true, nil)

			_, err := svc.TransitionTo(ctx, woID, tt.to)
			assert.NoError(t, err)

			transitionCall := woRepo.Calls[len(woRepo.Calls)-1]
			input := transitionCall.Arguments[1].(application.WorkOrderStatusTransitionInput)
			tt.checkFunc(t, input)
		})
	}
}

func TestTransitionTo_NotifiesOnlyWhenTransitionApplied(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	notifier := new(mockStatusNotifier)
	svc := NewWorkOrderStatusService(woRepo, notifier)
	ctx := context.Background()

	woID := uuid.New()
	wo := &domain.WorkOrder{ID: woID, Status: domain.WorkOrderStatusReceived, Code: "WO-001", CustomerID: uuid.New()}
	updated := &domain.WorkOrder{ID: woID, Status: domain.WorkOrderStatusInDiagnosis, Code: "WO-001", CustomerID: wo.CustomerID}

	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	woRepo.On("TransitionStatus", ctx, mock.AnythingOfType("application.WorkOrderStatusTransitionInput")).Return(updated, true, nil)
	notifier.On("NotifyTransition", ctx, updated, domain.WorkOrderStatusReceived).Once()

	_, err := svc.TransitionTo(ctx, woID, domain.WorkOrderStatusInDiagnosis)
	assert.NoError(t, err)
	notifier.AssertExpectations(t)
}

func TestTransitionTo_IdempotentConcurrent_DoesNotNotify(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	notifier := new(mockStatusNotifier)
	svc := NewWorkOrderStatusService(woRepo, notifier)
	ctx := context.Background()

	woID := uuid.New()
	wo := &domain.WorkOrder{ID: woID, Status: domain.WorkOrderStatusInDiagnosis}
	updated := &domain.WorkOrder{ID: woID, Status: domain.WorkOrderStatusInDiagnosis}

	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	woRepo.On("TransitionStatus", ctx, mock.AnythingOfType("application.WorkOrderStatusTransitionInput")).Return(updated, false, nil)

	_, err := svc.TransitionTo(ctx, woID, domain.WorkOrderStatusWaitingApproval)
	assert.NoError(t, err)
	notifier.AssertNotCalled(t, "NotifyTransition")
}

func TestIsValidTransition_AllowedTransitions(t *testing.T) {
	svc := NewWorkOrderStatusService(nil, nil)

	valid := []struct{ from, to domain.WorkOrderStatus }{
		{domain.WorkOrderStatusReceived, domain.WorkOrderStatusInDiagnosis},
		{domain.WorkOrderStatusReceived, domain.WorkOrderStatusCanceled},
		{domain.WorkOrderStatusInDiagnosis, domain.WorkOrderStatusWaitingApproval},
		{domain.WorkOrderStatusWaitingApproval, domain.WorkOrderStatusApproved},
		{domain.WorkOrderStatusWaitingApproval, domain.WorkOrderStatusCanceled},
		{domain.WorkOrderStatusApproved, domain.WorkOrderStatusInProgress},
		{domain.WorkOrderStatusInProgress, domain.WorkOrderStatusFinished},
		{domain.WorkOrderStatusFinished, domain.WorkOrderStatusDelivered},
	}
	for _, tc := range valid {
		assert.True(t, svc.IsValidTransition(tc.from, tc.to), "%s -> %s should be valid", tc.from, tc.to)
	}

	invalid := []struct{ from, to domain.WorkOrderStatus }{
		{domain.WorkOrderStatusDelivered, domain.WorkOrderStatusReceived},
		{domain.WorkOrderStatusReceived, domain.WorkOrderStatusFinished},
		{domain.WorkOrderStatusCanceled, domain.WorkOrderStatusReceived},
		{domain.WorkOrderStatusFinished, domain.WorkOrderStatusInProgress},
	}
	for _, tc := range invalid {
		assert.False(t, svc.IsValidTransition(tc.from, tc.to), "%s -> %s should be invalid", tc.from, tc.to)
	}
}

type mockStatusNotifier struct {
	mock.Mock
}

func (m *mockStatusNotifier) NotifyTransition(ctx context.Context, workOrder *domain.WorkOrder, previousStatus domain.WorkOrderStatus) {
	m.Called(ctx, workOrder, previousStatus)
}
