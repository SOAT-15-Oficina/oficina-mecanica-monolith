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

func sampleSupply() *domain.Supply {
	return &domain.Supply{
		ID:            uuid.New(),
		Title:         "Óleo Motor",
		Type:          "oil",
		PriceCents:    2500,
		StockQuantity: 10,
		Active:        true,
	}
}

func TestSupplyCreate_Success(t *testing.T) {
	repo := new(mockSupplyRepo)
	svc := NewSupplyService(repo, new(mockWorkOrderServiceRepo))
	ctx := context.Background()
	s := sampleSupply()

	repo.On("Create", ctx, s).Return(s, nil)

	result, err := svc.Create(ctx, s)
	require.NoError(t, err)
	assert.Equal(t, s.ID, result.ID)
	repo.AssertExpectations(t)
}

func TestSupplyCreate_RepoError(t *testing.T) {
	repo := new(mockSupplyRepo)
	svc := NewSupplyService(repo, new(mockWorkOrderServiceRepo))
	ctx := context.Background()
	s := sampleSupply()

	repo.On("Create", ctx, s).Return(nil, errors.New("db error"))

	result, err := svc.Create(ctx, s)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestSupplyGetByID_Success(t *testing.T) {
	repo := new(mockSupplyRepo)
	svc := NewSupplyService(repo, new(mockWorkOrderServiceRepo))
	ctx := context.Background()
	s := sampleSupply()

	repo.On("FindByID", ctx, s.ID).Return(s, nil)

	result, err := svc.GetByID(ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, s.ID, result.ID)
}

func TestSupplyGetByID_NotFound(t *testing.T) {
	repo := new(mockSupplyRepo)
	svc := NewSupplyService(repo, new(mockWorkOrderServiceRepo))
	ctx := context.Background()
	id := uuid.New()

	repo.On("FindByID", ctx, id).Return(nil, errors.New("not found"))

	result, err := svc.GetByID(ctx, id)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestSupplyGetAll_Success(t *testing.T) {
	repo := new(mockSupplyRepo)
	svc := NewSupplyService(repo, new(mockWorkOrderServiceRepo))
	ctx := context.Background()

	repo.On("FindAll", ctx).Return([]domain.Supply{*sampleSupply()}, nil)

	results, err := svc.GetAll(ctx)
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestSupplyGetAll_Empty(t *testing.T) {
	repo := new(mockSupplyRepo)
	svc := NewSupplyService(repo, new(mockWorkOrderServiceRepo))
	ctx := context.Background()

	repo.On("FindAll", ctx).Return(nil, errors.New("db error"))

	results, err := svc.GetAll(ctx)
	assert.Error(t, err)
	assert.Nil(t, results)
}

func TestSupplyUpdate_Success(t *testing.T) {
	repo := new(mockSupplyRepo)
	svc := NewSupplyService(repo, new(mockWorkOrderServiceRepo))
	ctx := context.Background()
	s := sampleSupply()

	repo.On("Update", ctx, mock.AnythingOfType("*domain.Supply")).Return(s, nil)

	result, err := svc.Update(ctx, s)
	require.NoError(t, err)
	assert.Equal(t, s.ID, result.ID)
}

func TestSupplyDelete_Success(t *testing.T) {
	repo := new(mockSupplyRepo)
	svc := NewSupplyService(repo, new(mockWorkOrderServiceRepo))
	ctx := context.Background()
	id := uuid.New()

	repo.On("Delete", ctx, id).Return(nil)

	err := svc.Delete(ctx, id)
	assert.NoError(t, err)
}

func TestSupplyDelete_RepoError(t *testing.T) {
	repo := new(mockSupplyRepo)
	svc := NewSupplyService(repo, new(mockWorkOrderServiceRepo))
	ctx := context.Background()
	id := uuid.New()

	repo.On("Delete", ctx, id).Return(errors.New("db error"))

	err := svc.Delete(ctx, id)
	assert.Error(t, err)
}

func TestSupplyPendingPurchases_Success(t *testing.T) {
	repo := new(mockSupplyRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	svc := NewSupplyService(repo, wosRepo)
	ctx := context.Background()

	expected := []application.SupplyShortageAlert{
		{WorkOrderCode: "WO-001", SupplyTitle: "Óleo Motor", Required: 10, InStock: 3},
	}
	wosRepo.On("FindApprovedServicesWithShortages", ctx).Return(expected, nil)

	result, err := svc.PendingPurchases(ctx)
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestSupplyPendingPurchases_NilNormalizedToEmpty(t *testing.T) {
	repo := new(mockSupplyRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	svc := NewSupplyService(repo, wosRepo)
	ctx := context.Background()

	wosRepo.On("FindApprovedServicesWithShortages", ctx).Return(nil, nil)

	result, err := svc.PendingPurchases(ctx)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestSupplyPendingPurchases_RepoError(t *testing.T) {
	repo := new(mockSupplyRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	svc := NewSupplyService(repo, wosRepo)
	ctx := context.Background()

	wosRepo.On("FindApprovedServicesWithShortages", ctx).Return(nil, errors.New("db error"))

	result, err := svc.PendingPurchases(ctx)
	assert.Error(t, err)
	assert.Nil(t, result)
}
