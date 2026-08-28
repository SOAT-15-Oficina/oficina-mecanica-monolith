package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockCustomerService struct {
	mock.Mock
}

func (m *mockCustomerService) Create(ctx context.Context, c *domain.Customer) (*domain.Customer, error) {
	args := m.Called(ctx, c)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Customer), args.Error(1)
}

func (m *mockCustomerService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Customer), args.Error(1)
}

func (m *mockCustomerService) GetAll(ctx context.Context) ([]domain.Customer, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Customer), args.Error(1)
}

func (m *mockCustomerService) GetAllWithFilters(ctx context.Context, filters domain.CustomerListFilters) ([]domain.Customer, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Customer), args.Error(1)
}

func (m *mockCustomerService) Update(ctx context.Context, c *domain.Customer) (*domain.Customer, error) {
	args := m.Called(ctx, c)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Customer), args.Error(1)
}

func (m *mockCustomerService) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func setupCustomerApp(svc *mockCustomerService) *fiber.App {
	app := fiber.New()
	h := NewCustomerHandler(svc)
	app.Post("/customers", h.Create)
	app.Get("/customers", h.GetAll)
	app.Get("/customers/:id", h.GetByID)
	app.Put("/customers/:id", h.Update)
	app.Delete("/customers/:id", h.Delete)
	return app
}

func customerJSON(c *domain.Customer) []byte {
	b, _ := json.Marshal(c)
	return b
}

func sampleCustomer() *domain.Customer {
	return &domain.Customer{
		ID:           uuid.New(),
		Name:         "João Silva",
		Email:        "joao@example.com",
		Document:     "11144477735",
		DocumentType: domain.DocumentTypeCPF,
	}
}

func TestCustomerCreate_Success(t *testing.T) {
	svc := new(mockCustomerService)
	app := setupCustomerApp(svc)
	c := sampleCustomer()

	svc.On("Create", mock.Anything, mock.AnythingOfType("*domain.Customer")).Return(c, nil)

	req := httptest.NewRequest(http.MethodPost, "/customers", bytes.NewReader(customerJSON(c)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
}

func TestCustomerCreate_InvalidPayload_400(t *testing.T) {
	svc := new(mockCustomerService)
	app := setupCustomerApp(svc)

	svc.On("Create", mock.Anything, mock.AnythingOfType("*domain.Customer")).Return(nil, domain.ErrCustomerNameRequired)

	req := httptest.NewRequest(http.MethodPost, "/customers", bytes.NewReader(customerJSON(&domain.Customer{})))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestCustomerGetAll_Success(t *testing.T) {
	svc := new(mockCustomerService)
	app := setupCustomerApp(svc)

	svc.On("GetAllWithFilters", mock.Anything, domain.CustomerListFilters{}).Return([]domain.Customer{*sampleCustomer()}, nil)

	req := httptest.NewRequest(http.MethodGet, "/customers", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestCustomerGetAll_FilterByDocument(t *testing.T) {
	svc := new(mockCustomerService)
	app := setupCustomerApp(svc)
	c := sampleCustomer()

	svc.On("GetAllWithFilters", mock.Anything, domain.CustomerListFilters{Document: "11144477735"}).Return([]domain.Customer{*c}, nil)

	req := httptest.NewRequest(http.MethodGet, "/customers?document=111.444.777-35", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestCustomerGetByID_Success(t *testing.T) {
	svc := new(mockCustomerService)
	app := setupCustomerApp(svc)
	c := sampleCustomer()

	svc.On("GetByID", mock.Anything, c.ID).Return(c, nil)

	req := httptest.NewRequest(http.MethodGet, "/customers/"+c.ID.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestCustomerGetByID_NotFound_404(t *testing.T) {
	svc := new(mockCustomerService)
	app := setupCustomerApp(svc)
	id := uuid.New()

	svc.On("GetByID", mock.Anything, id).Return(nil, pgx.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/customers/"+id.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestCustomerGetByID_InvalidID_400(t *testing.T) {
	svc := new(mockCustomerService)
	app := setupCustomerApp(svc)

	req := httptest.NewRequest(http.MethodGet, "/customers/not-a-uuid", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestCustomerDelete_Success(t *testing.T) {
	svc := new(mockCustomerService)
	app := setupCustomerApp(svc)
	id := uuid.New()

	svc.On("Delete", mock.Anything, id).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/customers/"+id.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestCustomerDelete_InvalidID_400(t *testing.T) {
	svc := new(mockCustomerService)
	app := setupCustomerApp(svc)

	req := httptest.NewRequest(http.MethodDelete, "/customers/bad-id", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestCustomerUpdate_Success(t *testing.T) {
	svc := new(mockCustomerService)
	app := setupCustomerApp(svc)
	c := sampleCustomer()

	svc.On("Update", mock.Anything, mock.AnythingOfType("*domain.Customer")).Return(c, nil)

	req := httptest.NewRequest(http.MethodPut, "/customers/"+c.ID.String(), bytes.NewReader(customerJSON(c)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestCustomerUpdate_InvalidID(t *testing.T) {
	svc := new(mockCustomerService)
	app := setupCustomerApp(svc)

	req := httptest.NewRequest(http.MethodPut, "/customers/bad-id", bytes.NewReader(customerJSON(sampleCustomer())))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestCustomerUpdate_NotFound(t *testing.T) {
	svc := new(mockCustomerService)
	app := setupCustomerApp(svc)
	id := uuid.New()

	svc.On("Update", mock.Anything, mock.AnythingOfType("*domain.Customer")).Return(nil, pgx.ErrNoRows)

	body := customerJSON(&domain.Customer{Name: "Test", Email: "t@t.com", Document: "11144477735", DocumentType: domain.DocumentTypeCPF})
	req := httptest.NewRequest(http.MethodPut, "/customers/"+id.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestCustomerUpdate_ValidationError(t *testing.T) {
	svc := new(mockCustomerService)
	app := setupCustomerApp(svc)
	id := uuid.New()

	svc.On("Update", mock.Anything, mock.AnythingOfType("*domain.Customer")).Return(nil, domain.ErrCustomerNameRequired)

	body := customerJSON(&domain.Customer{})
	req := httptest.NewRequest(http.MethodPut, "/customers/"+id.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestCustomerDelete_NotFound(t *testing.T) {
	svc := new(mockCustomerService)
	app := setupCustomerApp(svc)
	id := uuid.New()

	svc.On("Delete", mock.Anything, id).Return(errors.New("db error"))

	req := httptest.NewRequest(http.MethodDelete, "/customers/"+id.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestCustomerGetAll_Error(t *testing.T) {
	svc := new(mockCustomerService)
	app := setupCustomerApp(svc)

	svc.On("GetAllWithFilters", mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/customers", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestCustomerGetByID_ServerError(t *testing.T) {
	svc := new(mockCustomerService)
	app := setupCustomerApp(svc)
	id := uuid.New()

	svc.On("GetByID", mock.Anything, id).Return(nil, errors.New("unexpected error"))

	req := httptest.NewRequest(http.MethodGet, "/customers/"+id.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestCustomerCreate_ServerError(t *testing.T) {
	svc := new(mockCustomerService)
	app := setupCustomerApp(svc)

	svc.On("Create", mock.Anything, mock.AnythingOfType("*domain.Customer")).Return(nil, errors.New("unexpected error"))

	req := httptest.NewRequest(http.MethodPost, "/customers", bytes.NewReader(customerJSON(sampleCustomer())))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestCustomerGetAll_NilResult(t *testing.T) {
	svc := new(mockCustomerService)
	app := setupCustomerApp(svc)

	svc.On("GetAllWithFilters", mock.Anything, domain.CustomerListFilters{}).Return(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/customers", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}
