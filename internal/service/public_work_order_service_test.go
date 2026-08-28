package service

import (
	"context"
	"testing"
	"time"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGetPublicStatus_Success(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	custRepo := new(mockCustomerRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	svc := NewPublicWorkOrderService(woRepo, custRepo, wosRepo)

	ctx := context.Background()
	customerID := uuid.New()
	woID := uuid.New()
	now := time.Now()

	wo := &domain.WorkOrder{
		ID:         woID,
		Code:       "OS-20260504-A1B2",
		Status:     domain.WorkOrderStatusInProgress,
		CustomerID: customerID,
		ReceivedAt: now,
	}

	customer := &domain.Customer{
		ID:       customerID,
		Document: "12345678901",
	}

	services := []domain.WorkOrderService{
		{
			ServiceTitleSnapshot: "Troca de óleo",
			Status:               domain.WorkOrderServiceStatusInProgress,
			ApprovalStatus:       domain.WorkOrderServiceApprovalApproved,
		},
	}

	woRepo.On("FindByCode", ctx, "OS-20260504-A1B2").Return(wo, nil)
	custRepo.On("FindByID", ctx, customerID).Return(customer, nil)
	wosRepo.On("FindByWorkOrderID", ctx, woID).Return(services, nil)

	view, err := svc.GetPublicStatus(ctx, "OS-20260504-A1B2", "123.456.789-01")
	assert.NoError(t, err)
	assert.Equal(t, "OS-20260504-A1B2", view.Code)
	assert.Equal(t, domain.WorkOrderStatusInProgress, view.Status)
	assert.Len(t, view.Services, 1)
	assert.Equal(t, "Troca de óleo", view.Services[0].Title)
}

func TestGetPublicStatus_WorkOrderNotFound(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	custRepo := new(mockCustomerRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	svc := NewPublicWorkOrderService(woRepo, custRepo, wosRepo)

	ctx := context.Background()

	woRepo.On("FindByCode", ctx, "OS-INVALID").Return(nil, application.ErrNotFound)

	view, err := svc.GetPublicStatus(ctx, "OS-INVALID", "12345678901")
	assert.ErrorIs(t, err, ErrWorkOrderNotFound)
	assert.Nil(t, view)
}

func TestGetPublicStatus_DocumentMismatch(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	custRepo := new(mockCustomerRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	svc := NewPublicWorkOrderService(woRepo, custRepo, wosRepo)

	ctx := context.Background()
	customerID := uuid.New()

	wo := &domain.WorkOrder{
		ID:         uuid.New(),
		Code:       "OS-20260504-A1B2",
		CustomerID: customerID,
	}

	customer := &domain.Customer{
		ID:       customerID,
		Document: "12345678901",
	}

	woRepo.On("FindByCode", ctx, "OS-20260504-A1B2").Return(wo, nil)
	custRepo.On("FindByID", ctx, customerID).Return(customer, nil)

	view, err := svc.GetPublicStatus(ctx, "OS-20260504-A1B2", "99999999999")
	assert.ErrorIs(t, err, ErrWorkOrderNotFound)
	assert.Nil(t, view)
}
