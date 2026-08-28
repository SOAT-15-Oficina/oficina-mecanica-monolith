package service

import (
	"context"
	"errors"
	"testing"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func validVehicle() *domain.Vehicle {
	return &domain.Vehicle{
		LicensePlate: "ABC-1234",
		CustomerID:   uuid.New(),
		Model:        "Civic",
		Year:         2020,
		Brand:        "Honda",
	}
}

func savedVehicle() *domain.Vehicle {
	return &domain.Vehicle{
		ID:           uuid.New(),
		LicensePlate: "ABC1234",
		CustomerID:   uuid.New(),
		Model:        "Civic",
		Year:         2020,
		Brand:        "Honda",
	}
}

func TestVehicleCreate_Valid(t *testing.T) {
	repo := new(mockVehicleRepo)
	svc := NewVehicleService(repo)
	ctx := context.Background()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Vehicle")).Return(savedVehicle(), nil)

	result, err := svc.Create(ctx, validVehicle())
	require.NoError(t, err)
	assert.NotNil(t, result)
	repo.AssertExpectations(t)
}

func TestVehicleCreate_InvalidPlate(t *testing.T) {
	repo := new(mockVehicleRepo)
	svc := NewVehicleService(repo)
	ctx := context.Background()

	v := validVehicle()
	v.LicensePlate = "INVALID"

	result, err := svc.Create(ctx, v)
	assert.Error(t, err)
	assert.Nil(t, result)
	repo.AssertNotCalled(t, "Create")
}

func TestVehicleCreate_MissingCustomerID(t *testing.T) {
	repo := new(mockVehicleRepo)
	svc := NewVehicleService(repo)
	ctx := context.Background()

	v := validVehicle()
	v.CustomerID = uuid.Nil

	result, err := svc.Create(ctx, v)
	assert.Error(t, err)
	assert.Nil(t, result)
	repo.AssertNotCalled(t, "Create")
}

func TestVehicleCreate_NormalizesPlate(t *testing.T) {
	repo := new(mockVehicleRepo)
	svc := NewVehicleService(repo)
	ctx := context.Background()
	v := validVehicle()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Vehicle")).Return(savedVehicle(), nil)

	_, err := svc.Create(ctx, v)
	require.NoError(t, err)
	assert.Equal(t, "ABC1234", v.LicensePlate)
}

func TestVehicleGetByID_Success(t *testing.T) {
	repo := new(mockVehicleRepo)
	svc := NewVehicleService(repo)
	ctx := context.Background()
	id := uuid.New()

	repo.On("FindByID", ctx, id).Return(savedVehicle(), nil)

	result, err := svc.GetByID(ctx, id)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestVehicleGetByID_NotFound(t *testing.T) {
	repo := new(mockVehicleRepo)
	svc := NewVehicleService(repo)
	ctx := context.Background()
	id := uuid.New()

	repo.On("FindByID", ctx, id).Return(nil, errors.New("not found"))

	result, err := svc.GetByID(ctx, id)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestVehicleGetAll(t *testing.T) {
	repo := new(mockVehicleRepo)
	svc := NewVehicleService(repo)
	ctx := context.Background()

	repo.On("FindAll", ctx).Return([]domain.Vehicle{*savedVehicle()}, nil)

	results, err := svc.GetAll(ctx)
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestVehicleUpdate_Valid(t *testing.T) {
	repo := new(mockVehicleRepo)
	svc := NewVehicleService(repo)
	ctx := context.Background()

	v := savedVehicle()
	repo.On("Update", ctx, mock.AnythingOfType("*domain.Vehicle")).Return(v, nil)

	result, err := svc.Update(ctx, v)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestVehicleUpdate_InvalidPlate(t *testing.T) {
	repo := new(mockVehicleRepo)
	svc := NewVehicleService(repo)
	ctx := context.Background()

	v := savedVehicle()
	v.LicensePlate = "BAD"

	result, err := svc.Update(ctx, v)
	assert.Error(t, err)
	assert.Nil(t, result)
	repo.AssertNotCalled(t, "Update")
}

func TestVehicleDelete(t *testing.T) {
	repo := new(mockVehicleRepo)
	svc := NewVehicleService(repo)
	ctx := context.Background()
	id := uuid.New()

	repo.On("Delete", ctx, id).Return(nil)

	err := svc.Delete(ctx, id)
	assert.NoError(t, err)
}

func TestVehicleGetAllWithFilters_Success(t *testing.T) {
	repo := new(mockVehicleRepo)
	svc := NewVehicleService(repo)
	ctx := context.Background()

	filters := domain.VehicleListFilters{CustomerID: uuid.New()}
	vehicles := []domain.Vehicle{{ID: uuid.New(), LicensePlate: "ABC1234"}}
	repo.On("FindAllWithFilters", ctx, filters).Return(vehicles, nil)

	results, err := svc.GetAllWithFilters(ctx, filters)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestVehicleGetAllWithFilters_Error(t *testing.T) {
	repo := new(mockVehicleRepo)
	svc := NewVehicleService(repo)
	ctx := context.Background()

	filters := domain.VehicleListFilters{}
	repo.On("FindAllWithFilters", ctx, filters).Return(nil, errors.New("db error"))

	results, err := svc.GetAllWithFilters(ctx, filters)
	assert.Error(t, err)
	assert.Nil(t, results)
}
