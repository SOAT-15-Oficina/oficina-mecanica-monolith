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

type mockVehicleService struct {
	mock.Mock
}

func (m *mockVehicleService) Create(ctx context.Context, v *domain.Vehicle) (*domain.Vehicle, error) {
	args := m.Called(ctx, v)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Vehicle), args.Error(1)
}

func (m *mockVehicleService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Vehicle, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Vehicle), args.Error(1)
}

func (m *mockVehicleService) GetAll(ctx context.Context) ([]domain.Vehicle, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Vehicle), args.Error(1)
}

func (m *mockVehicleService) GetAllWithFilters(ctx context.Context, filters domain.VehicleListFilters) ([]domain.Vehicle, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Vehicle), args.Error(1)
}

func (m *mockVehicleService) Update(ctx context.Context, v *domain.Vehicle) (*domain.Vehicle, error) {
	args := m.Called(ctx, v)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Vehicle), args.Error(1)
}

func (m *mockVehicleService) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func setupVehicleApp(svc *mockVehicleService) *fiber.App {
	app := fiber.New()
	h := NewVehicleHandler(svc)
	app.Post("/vehicles", h.Create)
	app.Get("/vehicles", h.GetAll)
	app.Get("/vehicles/:id", h.GetByID)
	app.Put("/vehicles/:id", h.Update)
	app.Delete("/vehicles/:id", h.Delete)
	return app
}

func sampleVehicle() *domain.Vehicle {
	return &domain.Vehicle{
		ID:           uuid.New(),
		LicensePlate: "ABC1234",
		CustomerID:   uuid.New(),
		Model:        "Civic",
		Year:         2020,
		Brand:        "Honda",
	}
}

func vehicleJSON(v *domain.Vehicle) []byte {
	b, _ := json.Marshal(v)
	return b
}

func TestVehicleCreate_Success(t *testing.T) {
	svc := new(mockVehicleService)
	app := setupVehicleApp(svc)
	v := sampleVehicle()

	svc.On("Create", mock.Anything, mock.AnythingOfType("*domain.Vehicle")).Return(v, nil)

	req := httptest.NewRequest(http.MethodPost, "/vehicles", bytes.NewReader(vehicleJSON(v)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
}

func TestVehicleCreate_ValidationError_400(t *testing.T) {
	svc := new(mockVehicleService)
	app := setupVehicleApp(svc)

	svc.On("Create", mock.Anything, mock.AnythingOfType("*domain.Vehicle")).
		Return(nil, &domain.VehicleValidationError{Err: domain.ErrInvalidLicensePlate})

	v := sampleVehicle()
	v.LicensePlate = "INVALID"
	req := httptest.NewRequest(http.MethodPost, "/vehicles", bytes.NewReader(vehicleJSON(v)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestVehicleGetAll_Success(t *testing.T) {
	svc := new(mockVehicleService)
	app := setupVehicleApp(svc)

	svc.On("GetAllWithFilters", mock.Anything, domain.VehicleListFilters{}).Return([]domain.Vehicle{*sampleVehicle()}, nil)

	req := httptest.NewRequest(http.MethodGet, "/vehicles", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestVehicleGetAll_FilterByCustomerID(t *testing.T) {
	svc := new(mockVehicleService)
	app := setupVehicleApp(svc)
	v := sampleVehicle()
	customerID := uuid.New()

	svc.On("GetAllWithFilters", mock.Anything, domain.VehicleListFilters{CustomerID: customerID}).Return([]domain.Vehicle{*v}, nil)

	req := httptest.NewRequest(http.MethodGet, "/vehicles?customerId="+customerID.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestVehicleGetAll_FilterByCanonicalCustomerID(t *testing.T) {
	svc := new(mockVehicleService)
	app := setupVehicleApp(svc)
	customerID := uuid.New()

	svc.On("GetAllWithFilters", mock.Anything, domain.VehicleListFilters{CustomerID: customerID}).Return([]domain.Vehicle{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/vehicles?customer_id="+customerID.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestVehicleGetAll_RejectsInvalidOrConflictingCustomerID(t *testing.T) {
	urls := []string{
		"/vehicles?customer_id=invalid",
		"/vehicles?customer_id=" + uuid.NewString() + "&customerId=" + uuid.NewString(),
	}
	for _, url := range urls {
		svc := new(mockVehicleService)
		app := setupVehicleApp(svc)
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, url, nil))
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	}
}

func TestVehicleGetByID_Success(t *testing.T) {
	svc := new(mockVehicleService)
	app := setupVehicleApp(svc)
	v := sampleVehicle()

	svc.On("GetByID", mock.Anything, v.ID).Return(v, nil)

	req := httptest.NewRequest(http.MethodGet, "/vehicles/"+v.ID.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestVehicleGetByID_NotFound_404(t *testing.T) {
	svc := new(mockVehicleService)
	app := setupVehicleApp(svc)
	id := uuid.New()

	svc.On("GetByID", mock.Anything, id).Return(nil, pgx.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/vehicles/"+id.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestVehicleGetByID_InvalidID_400(t *testing.T) {
	svc := new(mockVehicleService)
	app := setupVehicleApp(svc)

	req := httptest.NewRequest(http.MethodGet, "/vehicles/not-a-uuid", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestVehicleDelete_Success(t *testing.T) {
	svc := new(mockVehicleService)
	app := setupVehicleApp(svc)
	id := uuid.New()

	svc.On("Delete", mock.Anything, id).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/vehicles/"+id.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestVehicleDelete_InvalidID_400(t *testing.T) {
	svc := new(mockVehicleService)
	app := setupVehicleApp(svc)

	req := httptest.NewRequest(http.MethodDelete, "/vehicles/bad-id", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestVehicleUpdate_Success(t *testing.T) {
	svc := new(mockVehicleService)
	app := setupVehicleApp(svc)
	v := sampleVehicle()

	svc.On("Update", mock.Anything, mock.AnythingOfType("*domain.Vehicle")).Return(v, nil)

	req := httptest.NewRequest(http.MethodPut, "/vehicles/"+v.ID.String(), bytes.NewReader(vehicleJSON(v)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestVehicleUpdate_InvalidID(t *testing.T) {
	svc := new(mockVehicleService)
	app := setupVehicleApp(svc)

	req := httptest.NewRequest(http.MethodPut, "/vehicles/bad-id", bytes.NewReader(vehicleJSON(sampleVehicle())))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestVehicleUpdate_NotFound(t *testing.T) {
	svc := new(mockVehicleService)
	app := setupVehicleApp(svc)
	id := uuid.New()

	svc.On("Update", mock.Anything, mock.AnythingOfType("*domain.Vehicle")).Return(nil, pgx.ErrNoRows)

	body := vehicleJSON(sampleVehicle())
	req := httptest.NewRequest(http.MethodPut, "/vehicles/"+id.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestVehicleUpdate_ValidationError(t *testing.T) {
	svc := new(mockVehicleService)
	app := setupVehicleApp(svc)
	id := uuid.New()

	svc.On("Update", mock.Anything, mock.AnythingOfType("*domain.Vehicle")).Return(nil, &domain.VehicleValidationError{Err: errors.New("invalid")})

	body := vehicleJSON(sampleVehicle())
	req := httptest.NewRequest(http.MethodPut, "/vehicles/"+id.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestVehicleDelete_Error(t *testing.T) {
	svc := new(mockVehicleService)
	app := setupVehicleApp(svc)
	id := uuid.New()

	svc.On("Delete", mock.Anything, id).Return(errors.New("db error"))

	req := httptest.NewRequest(http.MethodDelete, "/vehicles/"+id.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestVehicleGetByID_ServerError(t *testing.T) {
	svc := new(mockVehicleService)
	app := setupVehicleApp(svc)
	id := uuid.New()

	svc.On("GetByID", mock.Anything, id).Return(nil, errors.New("unexpected"))

	req := httptest.NewRequest(http.MethodGet, "/vehicles/"+id.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}
