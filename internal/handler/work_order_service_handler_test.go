package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- mock WorkOrderItemService ---

type mockWorkOrderItemService struct {
	mock.Mock
}

func (m *mockWorkOrderItemService) ApproveService(ctx context.Context, workOrderServiceID uuid.UUID) error {
	return m.Called(ctx, workOrderServiceID).Error(0)
}

func (m *mockWorkOrderItemService) RejectService(ctx context.Context, workOrderServiceID uuid.UUID) error {
	return m.Called(ctx, workOrderServiceID).Error(0)
}

func (m *mockWorkOrderItemService) ApproveAllByWorkOrder(ctx context.Context, workOrderID uuid.UUID) error {
	return m.Called(ctx, workOrderID).Error(0)
}

func (m *mockWorkOrderItemService) RejectAllByWorkOrder(ctx context.Context, workOrderID uuid.UUID) error {
	return m.Called(ctx, workOrderID).Error(0)
}

// --- helpers ---

func setupWorkOrderServiceApp(svc *mockWorkOrderItemService) *fiber.App {
	app := fiber.New()
	h := NewWorkOrderServiceHandler(svc)
	app.Post("/work-order-services/:workOrderServiceId/approve", h.Approve)
	app.Post("/work-order-services/:workOrderServiceId/reject", h.Reject)
	app.Post("/work-orders/:workOrderId/services/approve-all", h.ApproveAll)
	app.Post("/work-orders/:workOrderId/services/reject-all", h.RejectAll)
	return app
}

// --- Approve tests ---

func TestApprove_Success(t *testing.T) {
	svc := new(mockWorkOrderItemService)
	app := setupWorkOrderServiceApp(svc)
	id := uuid.New()

	svc.On("ApproveService", mock.Anything, id).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/work-order-services/"+id.String()+"/approve", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestApprove_InvalidID(t *testing.T) {
	svc := new(mockWorkOrderItemService)
	app := setupWorkOrderServiceApp(svc)

	req := httptest.NewRequest(http.MethodPost, "/work-order-services/not-a-uuid/approve", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestApprove_NotFound(t *testing.T) {
	svc := new(mockWorkOrderItemService)
	app := setupWorkOrderServiceApp(svc)
	id := uuid.New()

	svc.On("ApproveService", mock.Anything, id).Return(pgx.ErrNoRows)

	req := httptest.NewRequest(http.MethodPost, "/work-order-services/"+id.String()+"/approve", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestApprove_Error(t *testing.T) {
	svc := new(mockWorkOrderItemService)
	app := setupWorkOrderServiceApp(svc)
	id := uuid.New()

	svc.On("ApproveService", mock.Anything, id).Return(errors.New("internal error"))

	req := httptest.NewRequest(http.MethodPost, "/work-order-services/"+id.String()+"/approve", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

// --- Reject tests ---

func TestReject_Success(t *testing.T) {
	svc := new(mockWorkOrderItemService)
	app := setupWorkOrderServiceApp(svc)
	id := uuid.New()

	svc.On("RejectService", mock.Anything, id).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/work-order-services/"+id.String()+"/reject", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestReject_InvalidID(t *testing.T) {
	svc := new(mockWorkOrderItemService)
	app := setupWorkOrderServiceApp(svc)

	req := httptest.NewRequest(http.MethodPost, "/work-order-services/not-a-uuid/reject", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestReject_NotFound(t *testing.T) {
	svc := new(mockWorkOrderItemService)
	app := setupWorkOrderServiceApp(svc)
	id := uuid.New()

	svc.On("RejectService", mock.Anything, id).Return(pgx.ErrNoRows)

	req := httptest.NewRequest(http.MethodPost, "/work-order-services/"+id.String()+"/reject", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestReject_Error(t *testing.T) {
	svc := new(mockWorkOrderItemService)
	app := setupWorkOrderServiceApp(svc)
	id := uuid.New()

	svc.On("RejectService", mock.Anything, id).Return(errors.New("internal error"))

	req := httptest.NewRequest(http.MethodPost, "/work-order-services/"+id.String()+"/reject", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

// --- ApproveAll tests ---

func TestApproveAll_Success(t *testing.T) {
	svc := new(mockWorkOrderItemService)
	app := setupWorkOrderServiceApp(svc)
	id := uuid.New()

	svc.On("ApproveAllByWorkOrder", mock.Anything, id).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+id.String()+"/services/approve-all", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestApproveAll_InvalidID(t *testing.T) {
	svc := new(mockWorkOrderItemService)
	app := setupWorkOrderServiceApp(svc)

	req := httptest.NewRequest(http.MethodPost, "/work-orders/not-a-uuid/services/approve-all", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestApproveAll_Error(t *testing.T) {
	svc := new(mockWorkOrderItemService)
	app := setupWorkOrderServiceApp(svc)
	id := uuid.New()

	svc.On("ApproveAllByWorkOrder", mock.Anything, id).Return(errors.New("internal error"))

	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+id.String()+"/services/approve-all", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

// --- RejectAll tests ---

func TestRejectAll_Success(t *testing.T) {
	svc := new(mockWorkOrderItemService)
	app := setupWorkOrderServiceApp(svc)
	id := uuid.New()

	svc.On("RejectAllByWorkOrder", mock.Anything, id).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+id.String()+"/services/reject-all", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestRejectAll_InvalidID(t *testing.T) {
	svc := new(mockWorkOrderItemService)
	app := setupWorkOrderServiceApp(svc)

	req := httptest.NewRequest(http.MethodPost, "/work-orders/not-a-uuid/services/reject-all", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestRejectAll_Error(t *testing.T) {
	svc := new(mockWorkOrderItemService)
	app := setupWorkOrderServiceApp(svc)
	id := uuid.New()

	svc.On("RejectAllByWorkOrder", mock.Anything, id).Return(errors.New("internal error"))

	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+id.String()+"/services/reject-all", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}
