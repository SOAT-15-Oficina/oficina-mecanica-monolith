package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- mock PublicWorkOrderService ---

type mockPublicWorkOrderService struct {
	mock.Mock
}

func (m *mockPublicWorkOrderService) GetPublicStatus(ctx context.Context, code, document string) (*service.PublicWorkOrderView, error) {
	args := m.Called(ctx, code, document)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.PublicWorkOrderView), args.Error(1)
}

// --- helpers ---

func setupPublicWorkOrderApp(svc *mockPublicWorkOrderService) *fiber.App {
	app := fiber.New()
	h := NewPublicWorkOrderHandler(svc)
	app.Get("/work-orders/:code", h.GetByCode)
	return app
}

func samplePublicWorkOrderView() *service.PublicWorkOrderView {
	return &service.PublicWorkOrderView{
		Code:                     "WO-001",
		Status:                   domain.WorkOrderStatusApproved,
		TotalEstimatedPriceCents: 5000,
		ReceivedAt:               time.Now(),
		Services: []service.PublicServiceView{
			{
				Title:          "Troca de tela",
				Status:         "PENDENTE",
				ApprovalStatus: "PENDENTE",
			},
		},
	}
}

// --- tests ---

func TestGetByCode_Success(t *testing.T) {
	svc := new(mockPublicWorkOrderService)
	app := setupPublicWorkOrderApp(svc)
	view := samplePublicWorkOrderView()

	svc.On("GetPublicStatus", mock.Anything, "WO-001", "11144477735").Return(view, nil)

	req := httptest.NewRequest(http.MethodGet, "/work-orders/WO-001?document=11144477735", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestGetByCode_MissingDocument(t *testing.T) {
	svc := new(mockPublicWorkOrderService)
	app := setupPublicWorkOrderApp(svc)

	req := httptest.NewRequest(http.MethodGet, "/work-orders/WO-001", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGetByCode_NotFound(t *testing.T) {
	svc := new(mockPublicWorkOrderService)
	app := setupPublicWorkOrderApp(svc)

	svc.On("GetPublicStatus", mock.Anything, "WO-999", "11144477735").Return(nil, service.ErrWorkOrderNotFound)

	req := httptest.NewRequest(http.MethodGet, "/work-orders/WO-999?document=11144477735", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestGetByCode_ServerError(t *testing.T) {
	svc := new(mockPublicWorkOrderService)
	app := setupPublicWorkOrderApp(svc)

	svc.On("GetPublicStatus", mock.Anything, "WO-001", "11144477735").Return(nil, errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/work-orders/WO-001?document=11144477735", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}
