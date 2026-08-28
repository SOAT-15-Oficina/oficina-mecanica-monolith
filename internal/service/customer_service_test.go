package service

import (
	"context"
	"errors"
	"testing"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockCustomerRepo struct {
	mock.Mock
}

func (m *mockCustomerRepo) Create(ctx context.Context, c *domain.Customer) (*domain.Customer, error) {
	args := m.Called(ctx, c)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Customer), args.Error(1)
}

func (m *mockCustomerRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Customer), args.Error(1)
}

func (m *mockCustomerRepo) FindAll(ctx context.Context) ([]domain.Customer, error) {
	args := m.Called(ctx)
	return args.Get(0).([]domain.Customer), args.Error(1)
}

func (m *mockCustomerRepo) FindAllWithFilters(ctx context.Context, filters domain.CustomerListFilters) ([]domain.Customer, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Customer), args.Error(1)
}

func (m *mockCustomerRepo) Update(ctx context.Context, c *domain.Customer) (*domain.Customer, error) {
	args := m.Called(ctx, c)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Customer), args.Error(1)
}

func (m *mockCustomerRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func validCustomer() *domain.Customer {
	return &domain.Customer{
		Name:         "João Silva",
		Email:        "joao@example.com",
		Document:     "111.444.777-35",
		DocumentType: domain.DocumentTypeCPF,
	}
}

func savedCustomer() *domain.Customer {
	return &domain.Customer{
		ID:           uuid.New(),
		Name:         "João Silva",
		Email:        "joao@example.com",
		Document:     "11144477735",
		DocumentType: domain.DocumentTypeCPF,
	}
}

func TestCustomerCreate_ValidCPF(t *testing.T) {
	repo := new(mockCustomerRepo)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	customer := validCustomer()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Customer")).Return(savedCustomer(), nil)

	result, err := svc.Create(ctx, customer)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "11144477735", customer.Document)
	repo.AssertExpectations(t)
}

func TestCustomerCreate_NormalizesDocumentMask(t *testing.T) {
	repo := new(mockCustomerRepo)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	customer := validCustomer()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Customer")).Return(savedCustomer(), nil)

	_, err := svc.Create(ctx, customer)
	assert.NoError(t, err)
	assert.Equal(t, "11144477735", customer.Document)
}

func TestCustomerCreate_InvalidCPF(t *testing.T) {
	repo := new(mockCustomerRepo)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	customer := validCustomer()
	customer.Document = "111.444.777-00"

	result, err := svc.Create(ctx, customer)
	assert.ErrorIs(t, err, domain.ErrCustomerInvalidCPFChecksum)
	assert.Nil(t, result)
	repo.AssertNotCalled(t, "Create")
}

func TestCustomerCreate_MissingName(t *testing.T) {
	repo := new(mockCustomerRepo)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	customer := validCustomer()
	customer.Name = ""

	result, err := svc.Create(ctx, customer)
	assert.ErrorIs(t, err, domain.ErrCustomerNameRequired)
	assert.Nil(t, result)
	repo.AssertNotCalled(t, "Create")
}

func TestCustomerCreate_InvalidDocumentType(t *testing.T) {
	repo := new(mockCustomerRepo)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	customer := validCustomer()
	customer.DocumentType = "RG"

	result, err := svc.Create(ctx, customer)
	assert.ErrorIs(t, err, domain.ErrCustomerInvalidDocumentType)
	assert.Nil(t, result)
	repo.AssertNotCalled(t, "Create")
}

func TestCustomerUpdate_ValidCPF(t *testing.T) {
	repo := new(mockCustomerRepo)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	customer := validCustomer()
	customer.ID = uuid.New()

	repo.On("Update", ctx, mock.AnythingOfType("*domain.Customer")).Return(savedCustomer(), nil)

	result, err := svc.Update(ctx, customer)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	repo.AssertExpectations(t)
}

func TestCustomerUpdate_InvalidCPF(t *testing.T) {
	repo := new(mockCustomerRepo)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	customer := validCustomer()
	customer.Document = "000.000.000-00"

	result, err := svc.Update(ctx, customer)
	assert.ErrorIs(t, err, domain.ErrCustomerInvalidCPFChecksum)
	assert.Nil(t, result)
	repo.AssertNotCalled(t, "Update")
}

func TestCustomerGetByID_Success(t *testing.T) {
	repo := new(mockCustomerRepo)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	c := savedCustomer()

	repo.On("FindByID", ctx, c.ID).Return(c, nil)

	result, err := svc.GetByID(ctx, c.ID)
	assert.NoError(t, err)
	assert.Equal(t, c.ID, result.ID)
}

func TestCustomerGetByID_NotFound(t *testing.T) {
	repo := new(mockCustomerRepo)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	id := uuid.New()

	repo.On("FindByID", ctx, id).Return(nil, pgx.ErrNoRows)

	result, err := svc.GetByID(ctx, id)
	assert.ErrorIs(t, err, pgx.ErrNoRows)
	assert.Nil(t, result)
}

func TestCustomerGetAll_Success(t *testing.T) {
	repo := new(mockCustomerRepo)
	svc := NewCustomerService(repo)
	ctx := context.Background()

	repo.On("FindAll", ctx).Return([]domain.Customer{*savedCustomer()}, nil)

	results, err := svc.GetAll(ctx)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestCustomerDelete_Success(t *testing.T) {
	repo := new(mockCustomerRepo)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	id := uuid.New()

	repo.On("Delete", ctx, id).Return(nil)

	err := svc.Delete(ctx, id)
	assert.NoError(t, err)
}

func TestCustomerDelete_Error(t *testing.T) {
	repo := new(mockCustomerRepo)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	id := uuid.New()

	repo.On("Delete", ctx, id).Return(errors.New("db error"))

	err := svc.Delete(ctx, id)
	assert.Error(t, err)
}

func TestCustomerGetAllWithFilters_Success(t *testing.T) {
	repo := new(mockCustomerRepo)
	svc := NewCustomerService(repo)
	ctx := context.Background()

	filters := domain.CustomerListFilters{Document: "11144477735"}
	repo.On("FindAllWithFilters", ctx, filters).Return([]domain.Customer{*savedCustomer()}, nil)

	results, err := svc.GetAllWithFilters(ctx, filters)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestCustomerGetAllWithFilters_Error(t *testing.T) {
	repo := new(mockCustomerRepo)
	svc := NewCustomerService(repo)
	ctx := context.Background()

	filters := domain.CustomerListFilters{}
	repo.On("FindAllWithFilters", ctx, filters).Return(nil, errors.New("db error"))

	results, err := svc.GetAllWithFilters(ctx, filters)
	assert.Error(t, err)
	assert.Nil(t, results)
}

func TestCustomerCreate_RepoError(t *testing.T) {
	repo := new(mockCustomerRepo)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	customer := validCustomer()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Customer")).Return(nil, errors.New("db error"))

	result, err := svc.Create(ctx, customer)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestCustomerUpdate_RepoError(t *testing.T) {
	repo := new(mockCustomerRepo)
	svc := NewCustomerService(repo)
	ctx := context.Background()
	customer := validCustomer()
	customer.ID = uuid.New()

	repo.On("Update", ctx, mock.AnythingOfType("*domain.Customer")).Return(nil, errors.New("db error"))

	result, err := svc.Update(ctx, customer)
	assert.Error(t, err)
	assert.Nil(t, result)
}
