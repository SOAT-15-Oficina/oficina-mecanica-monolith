package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- mock SupplyService ---

type mockSupplyService struct {
	mock.Mock
}

func (m *mockSupplyService) Create(ctx context.Context, s *domain.Supply) (*domain.Supply, error) {
	args := m.Called(ctx, s)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Supply), args.Error(1)
}

func (m *mockSupplyService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Supply, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Supply), args.Error(1)
}

func (m *mockSupplyService) GetAll(ctx context.Context) ([]domain.Supply, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Supply), args.Error(1)
}

func (m *mockSupplyService) Update(ctx context.Context, s *domain.Supply) (*domain.Supply, error) {
	args := m.Called(ctx, s)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Supply), args.Error(1)
}

func (m *mockSupplyService) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockSupplyService) PendingPurchases(ctx context.Context) ([]application.SupplyShortageAlert, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]application.SupplyShortageAlert), args.Error(1)
}

// --- helpers ---

func setupSupplyApp(svc *mockSupplyService) *fiber.App {
	app := fiber.New()
	h := NewSupplyHandler(svc)
	app.Post("/supplies", h.Create)
	app.Get("/supplies/pending-purchases", h.PendingPurchases)
	app.Get("/supplies", h.GetAll)
	app.Get("/supplies/:id", h.GetByID)
	app.Put("/supplies/:id", h.Update)
	app.Delete("/supplies/:id", h.Delete)
	return app
}

func supplyJSON(s *domain.Supply) []byte {
	b, _ := json.Marshal(s)
	return b
}

func sampleSupply() *domain.Supply {
	return &domain.Supply{
		ID:            uuid.New(),
		Title:         "Parafuso M6",
		Type:          "material",
		PriceCents:    150,
		StockQuantity: 100,
		MinimumStock:  10,
		Active:        true,
	}
}

// --- tests ---

func TestSupplyCreate_Success(t *testing.T) {
	svc := new(mockSupplyService)
	app := setupSupplyApp(svc)
	s := sampleSupply()

	svc.On("Create", mock.Anything, mock.AnythingOfType("*domain.Supply")).Return(s, nil)

	req := httptest.NewRequest(http.MethodPost, "/supplies", bytes.NewReader(supplyJSON(s)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
}

func TestSupplyCreate_InvalidJSON(t *testing.T) {
	svc := new(mockSupplyService)
	app := setupSupplyApp(svc)

	req := httptest.NewRequest(http.MethodPost, "/supplies", bytes.NewReader([]byte("{invalid")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestSupplyCreate_ServiceError(t *testing.T) {
	svc := new(mockSupplyService)
	app := setupSupplyApp(svc)
	s := sampleSupply()

	svc.On("Create", mock.Anything, mock.AnythingOfType("*domain.Supply")).Return(nil, errors.New("db error"))

	req := httptest.NewRequest(http.MethodPost, "/supplies", bytes.NewReader(supplyJSON(s)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestSupplyGetAll_Success(t *testing.T) {
	svc := new(mockSupplyService)
	app := setupSupplyApp(svc)

	svc.On("GetAll", mock.Anything).Return([]domain.Supply{*sampleSupply()}, nil)

	req := httptest.NewRequest(http.MethodGet, "/supplies", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestSupplyGetAll_Error(t *testing.T) {
	svc := new(mockSupplyService)
	app := setupSupplyApp(svc)

	svc.On("GetAll", mock.Anything).Return(nil, errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/supplies", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestSupplyGetByID_Success(t *testing.T) {
	svc := new(mockSupplyService)
	app := setupSupplyApp(svc)
	s := sampleSupply()

	svc.On("GetByID", mock.Anything, s.ID).Return(s, nil)

	req := httptest.NewRequest(http.MethodGet, "/supplies/"+s.ID.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestSupplyGetByID_InvalidID(t *testing.T) {
	svc := new(mockSupplyService)
	app := setupSupplyApp(svc)

	req := httptest.NewRequest(http.MethodGet, "/supplies/not-a-uuid", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestSupplyGetByID_NotFound(t *testing.T) {
	svc := new(mockSupplyService)
	app := setupSupplyApp(svc)
	id := uuid.New()

	svc.On("GetByID", mock.Anything, id).Return(nil, pgx.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/supplies/"+id.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestSupplyUpdate_Success(t *testing.T) {
	svc := new(mockSupplyService)
	app := setupSupplyApp(svc)
	s := sampleSupply()

	svc.On("Update", mock.Anything, mock.AnythingOfType("*domain.Supply")).Return(s, nil)

	req := httptest.NewRequest(http.MethodPut, "/supplies/"+s.ID.String(), bytes.NewReader(supplyJSON(s)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestSupplyUpdate_InvalidID(t *testing.T) {
	svc := new(mockSupplyService)
	app := setupSupplyApp(svc)

	req := httptest.NewRequest(http.MethodPut, "/supplies/not-a-uuid", bytes.NewReader(supplyJSON(sampleSupply())))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestSupplyUpdate_InvalidJSON(t *testing.T) {
	svc := new(mockSupplyService)
	app := setupSupplyApp(svc)
	id := uuid.New()

	req := httptest.NewRequest(http.MethodPut, "/supplies/"+id.String(), bytes.NewReader([]byte("{invalid")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestSupplyDelete_Success(t *testing.T) {
	svc := new(mockSupplyService)
	app := setupSupplyApp(svc)
	id := uuid.New()

	svc.On("Delete", mock.Anything, id).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/supplies/"+id.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestSupplyDelete_InvalidID(t *testing.T) {
	svc := new(mockSupplyService)
	app := setupSupplyApp(svc)

	req := httptest.NewRequest(http.MethodDelete, "/supplies/bad-id", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestSupplyDelete_Error(t *testing.T) {
	svc := new(mockSupplyService)
	app := setupSupplyApp(svc)
	id := uuid.New()

	svc.On("Delete", mock.Anything, id).Return(errors.New("db error"))

	req := httptest.NewRequest(http.MethodDelete, "/supplies/"+id.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestSupplyPendingPurchases_Success(t *testing.T) {
	svc := new(mockSupplyService)
	app := setupSupplyApp(svc)

	alerts := []application.SupplyShortageAlert{
		{
			WorkOrderCode:  "WO-001",
			WorkOrderTitle: "Test WO",
			ServiceTitle:   "Service A",
			SupplyTitle:    "Supply X",
			SupplyID:       uuid.New(),
			Required:       10,
			InStock:        3,
		},
	}
	svc.On("PendingPurchases", mock.Anything).Return(alerts, nil)

	req := httptest.NewRequest(http.MethodGet, "/supplies/pending-purchases", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var payload map[string][]map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	item := payload["data"][0]
	assert.Equal(t, "WO-001", item["work_order_code"])
	assert.Equal(t, "Test WO", item["work_order_title"])
	assert.Equal(t, "Service A", item["service_title"])
	assert.Equal(t, "Supply X", item["supply_title"])
	assert.NotEmpty(t, item["supply_id"])
	assert.Equal(t, float64(10), item["required"])
	assert.Equal(t, float64(3), item["in_stock"])
}

func TestSupplyPendingPurchases_Error(t *testing.T) {
	svc := new(mockSupplyService)
	app := setupSupplyApp(svc)

	svc.On("PendingPurchases", mock.Anything).Return(nil, errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/supplies/pending-purchases", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}
