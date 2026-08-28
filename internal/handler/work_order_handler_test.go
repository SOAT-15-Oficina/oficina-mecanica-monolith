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
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/auth"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockWorkOrderService struct {
	mock.Mock
}

func (m *mockWorkOrderService) Create(ctx context.Context, wo *domain.WorkOrder) (*domain.WorkOrder, error) {
	args := m.Called(ctx, wo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkOrder), args.Error(1)
}

func (m *mockWorkOrderService) GetByID(ctx context.Context, id uuid.UUID) (*domain.WorkOrder, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkOrder), args.Error(1)
}

func (m *mockWorkOrderService) GetAll(ctx context.Context) ([]domain.WorkOrder, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.WorkOrder), args.Error(1)
}

func (m *mockWorkOrderService) GetAllWithFilters(ctx context.Context, filters application.WorkOrderListFilters) (*application.WorkOrderListResponse, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*application.WorkOrderListResponse), args.Error(1)
}

func (m *mockWorkOrderService) Update(ctx context.Context, wo *domain.WorkOrder) (*domain.WorkOrder, error) {
	args := m.Called(ctx, wo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkOrder), args.Error(1)
}

// --- Additional mocks for full work order handler testing ---

type mockCreationService struct{ mock.Mock }

func (m *mockCreationService) AddServices(ctx context.Context, woID uuid.UUID, items []service.AddWorkOrderServiceInput) ([]domain.WorkOrderService, error) {
	args := m.Called(ctx, woID, items)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.WorkOrderService), args.Error(1)
}
func (m *mockCreationService) AddSupplies(ctx context.Context, woID, wosID uuid.UUID, items []service.AddWorkOrderSupplyInput) ([]domain.WorkOrderServiceSupply, error) {
	args := m.Called(ctx, woID, wosID, items)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.WorkOrderServiceSupply), args.Error(1)
}
func (m *mockCreationService) RemoveSupplyFromService(ctx context.Context, woID, wosID, supplyID uuid.UUID) error {
	return m.Called(ctx, woID, wosID, supplyID).Error(0)
}
func (m *mockCreationService) RemoveService(ctx context.Context, woID, wosID uuid.UUID) error {
	return m.Called(ctx, woID, wosID).Error(0)
}
func (m *mockCreationService) StartService(ctx context.Context, woID, wosID uuid.UUID) error {
	return m.Called(ctx, woID, wosID).Error(0)
}
func (m *mockCreationService) FinalizeService(ctx context.Context, woID, wosID uuid.UUID) error {
	return m.Called(ctx, woID, wosID).Error(0)
}

type mockStatusSvc struct{ mock.Mock }

func (m *mockStatusSvc) TransitionTo(ctx context.Context, workOrderID uuid.UUID, newStatus domain.WorkOrderStatus) (*domain.WorkOrder, error) {
	args := m.Called(ctx, workOrderID, newStatus)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkOrder), args.Error(1)
}
func (m *mockStatusSvc) IsValidTransition(from, to domain.WorkOrderStatus) bool {
	return m.Called(from, to).Bool(0)
}

type mockUserRepo struct{ mock.Mock }

func (m *mockUserRepo) Register(ctx context.Context, username, password string, role domain.UserRole) (*domain.User, error) {
	args := m.Called(ctx, username, password, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *mockUserRepo) Login(ctx context.Context, username, password string) (string, error) {
	args := m.Called(ctx, username, password)
	return args.String(0), args.Error(1)
}
func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *mockUserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *mockUserRepo) GetAll(ctx context.Context) ([]domain.User, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.User), args.Error(1)
}
func (m *mockUserRepo) Update(ctx context.Context, id uuid.UUID, username string, role domain.UserRole) (*domain.User, error) {
	args := m.Called(ctx, id, username, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *mockUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *mockUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *mockUserRepo) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *mockUserRepo) FindAll(ctx context.Context) ([]domain.User, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.User), args.Error(1)
}

// --- Full setup ---

type woTestDeps struct {
	woSvc       *mockWorkOrderService
	creationSvc *mockCreationService
	statusSvc   *mockStatusSvc
	userRepo    *mockUserRepo
}

func setupFullWorkOrderApp() (*fiber.App, *woTestDeps) {
	deps := &woTestDeps{
		woSvc:       new(mockWorkOrderService),
		creationSvc: new(mockCreationService),
		statusSvc:   new(mockStatusSvc),
		userRepo:    new(mockUserRepo),
	}
	app := fiber.New()
	h := NewWorkOrderHandler(deps.woSvc, deps.creationSvc, deps.statusSvc, deps.userRepo)
	g := app.Group("/work-orders")
	g.Get("/", h.GetAll)
	g.Get("/:id", h.GetByID)
	g.Post("/", func(c fiber.Ctx) error {
		c.Locals("token", &auth.AppClaims{User: "testuser", Role: "admin"})
		return c.Next()
	}, h.Create)
	g.Put("/:id", h.Update)
	g.Post("/:id/services", h.AddServices)
	g.Delete("/:id/services/:wosId", h.RemoveService)
	g.Post("/:id/services/:wosId/supplies", h.AddSupplies)
	g.Delete("/:id/services/:wosId/supplies/:supplyId", h.RemoveSupplyFromService)
	g.Post("/:id/services/:wosId/start", h.StartService)
	g.Post("/:id/services/:wosId/finalize", h.FinalizeService)
	return app, deps
}

// --- GetAll tests ---

func TestWorkOrder_GetAll_Success(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	resp := &application.WorkOrderListResponse{
		Data: []domain.WorkOrder{{ID: uuid.New(), Title: "test"}}, Total: 1, Page: 1, Limit: 10, TotalPages: 1,
	}
	deps.woSvc.On("GetAllWithFilters", mock.Anything, mock.Anything).Return(resp, nil)

	req := httptest.NewRequest(http.MethodGet, "/work-orders/", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, r.StatusCode)
}

func TestWorkOrder_GetAll_WithFilters(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	resp := &application.WorkOrderListResponse{Data: []domain.WorkOrder{}, Total: 0, Page: 2, Limit: 5, TotalPages: 0}
	deps.woSvc.On("GetAllWithFilters", mock.Anything, mock.Anything).Return(resp, nil)

	custID := uuid.New()
	vehID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/work-orders/?page=2&limit=5&status=RECEBIDA&customerId="+custID.String()+"&vehicleId="+vehID.String()+"&from=2026-01-01&to=2026-12-31", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, r.StatusCode)
}

func TestWorkOrder_GetAll_WithCanonicalFilters(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	customerID := uuid.New()
	vehicleID := uuid.New()
	deps.woSvc.On("GetAllWithFilters", mock.Anything, mock.MatchedBy(func(filters application.WorkOrderListFilters) bool {
		return filters.CustomerID == customerID && filters.VehicleID == vehicleID
	})).Return(&application.WorkOrderListResponse{Data: []domain.WorkOrder{}, Page: 1, Limit: 10}, nil)

	req := httptest.NewRequest(http.MethodGet, "/work-orders/?customer_id="+customerID.String()+"&vehicle_id="+vehicleID.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestWorkOrder_GetAll_RejectsConflictingAliases(t *testing.T) {
	app, _ := setupFullWorkOrderApp()
	req := httptest.NewRequest(http.MethodGet, "/work-orders/?customer_id="+uuid.NewString()+"&customerId="+uuid.NewString(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestWorkOrder_GetAll_RejectsInvalidFilters(t *testing.T) {
	invalidURLs := []string{
		"/work-orders/?page=0",
		"/work-orders/?limit=101",
		"/work-orders/?status=DESCONHECIDA",
		"/work-orders/?customer_id=invalid",
		"/work-orders/?vehicle_id=invalid",
		"/work-orders/?from=13-07-2026",
		"/work-orders/?to=13-07-2026",
		"/work-orders/?from=2026-07-14&to=2026-07-13",
	}

	for _, url := range invalidURLs {
		t.Run(url, func(t *testing.T) {
			app, _ := setupFullWorkOrderApp()
			resp, err := app.Test(httptest.NewRequest(http.MethodGet, url, nil))
			require.NoError(t, err)
			assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
		})
	}
}

func TestWorkOrder_GetAll_NilResult(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	deps.woSvc.On("GetAllWithFilters", mock.Anything, mock.Anything).Return(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/work-orders/", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, r.StatusCode)
}

func TestWorkOrder_GetAll_Error(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	deps.woSvc.On("GetAllWithFilters", mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/work-orders/", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, r.StatusCode)
}

// --- GetByID tests ---

func TestWorkOrder_GetByID_Success(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	id := uuid.New()
	wo := &domain.WorkOrder{ID: id, Title: "test"}
	deps.woSvc.On("GetByID", mock.Anything, id).Return(wo, nil)

	req := httptest.NewRequest(http.MethodGet, "/work-orders/"+id.String(), nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, r.StatusCode)
}

func TestWorkOrder_GetByID_InvalidID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()
	req := httptest.NewRequest(http.MethodGet, "/work-orders/bad-id", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_GetByID_NotFound(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	id := uuid.New()
	deps.woSvc.On("GetByID", mock.Anything, id).Return(nil, pgx.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/work-orders/"+id.String(), nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, r.StatusCode)
}

func TestWorkOrder_GetByID_ServerError(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	id := uuid.New()
	deps.woSvc.On("GetByID", mock.Anything, id).Return(nil, errors.New("unexpected"))

	req := httptest.NewRequest(http.MethodGet, "/work-orders/"+id.String(), nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, r.StatusCode)
}

// --- Create tests ---

func TestWorkOrder_Create_Success(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	user := &domain.User{ID: uuid.New(), Username: "testuser"}
	deps.userRepo.On("GetByUsername", mock.Anything, "testuser").Return(user, nil)
	wo := &domain.WorkOrder{ID: uuid.New(), Title: "test", Code: "OS-123"}
	deps.woSvc.On("Create", mock.Anything, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)

	body, _ := json.Marshal(map[string]any{"title": "test", "customer_id": uuid.New(), "vehicle_id": uuid.New()})
	req := httptest.NewRequest(http.MethodPost, "/work-orders/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, r.StatusCode)
}

func TestWorkOrder_Create_NoToken(t *testing.T) {
	deps := &woTestDeps{
		woSvc: new(mockWorkOrderService),
		creationSvc: new(mockCreationService), statusSvc: new(mockStatusSvc), userRepo: new(mockUserRepo),
	}
	app := fiber.New()
	h := NewWorkOrderHandler(deps.woSvc, deps.creationSvc, deps.statusSvc, deps.userRepo)
	app.Post("/work-orders", h.Create) // no middleware to set token

	body, _ := json.Marshal(map[string]any{"title": "test"})
	req := httptest.NewRequest(http.MethodPost, "/work-orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, r.StatusCode)
}

func TestWorkOrder_Create_UserRepoError(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	deps.userRepo.On("GetByUsername", mock.Anything, "testuser").Return(nil, errors.New("db error"))

	body, _ := json.Marshal(map[string]any{"title": "test", "customer_id": uuid.New(), "vehicle_id": uuid.New()})
	req := httptest.NewRequest(http.MethodPost, "/work-orders/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, r.StatusCode)
}

func TestWorkOrder_Create_ServiceError(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	user := &domain.User{ID: uuid.New(), Username: "testuser"}
	deps.userRepo.On("GetByUsername", mock.Anything, "testuser").Return(user, nil)
	deps.woSvc.On("Create", mock.Anything, mock.AnythingOfType("*domain.WorkOrder")).Return(nil, application.NewValidationError("validation error"))

	body, _ := json.Marshal(map[string]any{"title": "test", "customer_id": uuid.New(), "vehicle_id": uuid.New()})
	req := httptest.NewRequest(http.MethodPost, "/work-orders/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_InternalErrorDoesNotLeakDetails(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	deps.woSvc.On("GetAllWithFilters", mock.Anything, mock.Anything).Return(nil, errors.New("password authentication failed for database techchallenge"))

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/work-orders/", nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

	var payload map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	assert.Equal(t, "internal server error", payload["error"])
}

// --- Update tests ---

func TestWorkOrder_Update_Success(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	id := uuid.New()
	wo := &domain.WorkOrder{ID: id, Title: "updated"}
	deps.woSvc.On("Update", mock.Anything, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)

	body, _ := json.Marshal(map[string]any{"title": "updated"})
	req := httptest.NewRequest(http.MethodPut, "/work-orders/"+id.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, r.StatusCode)
}

func TestWorkOrder_Update_InvalidID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()

	body, _ := json.Marshal(map[string]any{"title": "updated"})
	req := httptest.NewRequest(http.MethodPut, "/work-orders/bad-id", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_Update_WithStatusTransition(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	id := uuid.New()
	wo := &domain.WorkOrder{ID: id, Status: domain.WorkOrderStatusInDiagnosis}
	deps.statusSvc.On("TransitionTo", mock.Anything, id, domain.WorkOrderStatusInDiagnosis).Return(wo, nil)
	deps.woSvc.On("Update", mock.Anything, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)

	body, _ := json.Marshal(map[string]any{"status": domain.WorkOrderStatusInDiagnosis})
	req := httptest.NewRequest(http.MethodPut, "/work-orders/"+id.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, r.StatusCode)
}

func TestWorkOrder_Update_StatusTransitionInvalid(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	id := uuid.New()
	deps.statusSvc.On("TransitionTo", mock.Anything, id, mock.Anything).Return(nil, service.ErrInvalidStatusTransition)

	body, _ := json.Marshal(map[string]any{"status": "ENTREGUE"})
	req := httptest.NewRequest(http.MethodPut, "/work-orders/"+id.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.StatusCode)
}

func TestWorkOrder_Update_StatusTransitionOnly(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	id := uuid.New()
	wo := &domain.WorkOrder{ID: id, Status: domain.WorkOrderStatusInDiagnosis}
	deps.statusSvc.On("TransitionTo", mock.Anything, id, domain.WorkOrderStatusInDiagnosis).Return(wo, nil)
	deps.woSvc.On("Update", mock.Anything, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)

	body, _ := json.Marshal(map[string]any{"status": domain.WorkOrderStatusInDiagnosis})
	req := httptest.NewRequest(http.MethodPut, "/work-orders/"+id.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, r.StatusCode)
}

func TestWorkOrder_Update_NotFound(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	id := uuid.New()
	deps.woSvc.On("Update", mock.Anything, mock.AnythingOfType("*domain.WorkOrder")).Return(nil, pgx.ErrNoRows)

	body, _ := json.Marshal(map[string]any{"title": "x"})
	req := httptest.NewRequest(http.MethodPut, "/work-orders/"+id.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, r.StatusCode)
}

// --- AddServices tests ---

func TestWorkOrder_AddServices_Success(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	result := []domain.WorkOrderService{{ID: uuid.New(), WorkOrderID: woID}}
	deps.creationSvc.On("AddServices", mock.Anything, woID, mock.Anything).Return(result, nil)

	body, _ := json.Marshal([]map[string]any{{"service_id": uuid.New()}})
	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+woID.String()+"/services", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, r.StatusCode)
}

func TestWorkOrder_AddServices_InvalidID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()
	req := httptest.NewRequest(http.MethodPost, "/work-orders/bad-id/services", bytes.NewReader([]byte("[]")))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_AddServices_InvalidStatus(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	deps.creationSvc.On("AddServices", mock.Anything, woID, mock.Anything).Return(nil, service.ErrWorkOrderInvalidStatusForItems)

	body, _ := json.Marshal([]map[string]any{{"service_id": uuid.New()}})
	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+woID.String()+"/services", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.StatusCode)
}

func TestWorkOrder_AddServices_InactiveService(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	deps.creationSvc.On("AddServices", mock.Anything, woID, mock.Anything).Return(nil, service.ErrWorkshopServiceInactive)

	body, _ := json.Marshal([]map[string]any{{"service_id": uuid.New()}})
	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+woID.String()+"/services", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.StatusCode)
}

// --- RemoveService tests ---

func TestWorkOrder_RemoveService_Success(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	deps.creationSvc.On("RemoveService", mock.Anything, woID, wosID).Return(nil)
	deps.woSvc.On("GetByID", mock.Anything, woID).Return(&domain.WorkOrder{ID: woID, Status: domain.WorkOrderStatusInDiagnosis}, nil)

	req := httptest.NewRequest(http.MethodDelete, "/work-orders/"+woID.String()+"/services/"+wosID.String(), nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, r.StatusCode)
}

func TestWorkOrder_RemoveService_InvalidWoID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()
	req := httptest.NewRequest(http.MethodDelete, "/work-orders/bad-id/services/"+uuid.New().String(), nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_RemoveService_InvalidWosID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()
	req := httptest.NewRequest(http.MethodDelete, "/work-orders/"+uuid.New().String()+"/services/bad-id", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_RemoveService_Ownership(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	deps.creationSvc.On("RemoveService", mock.Anything, woID, wosID).Return(service.ErrWorkOrderServiceOwnership)

	req := httptest.NewRequest(http.MethodDelete, "/work-orders/"+woID.String()+"/services/"+wosID.String(), nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.StatusCode)
}

func TestWorkOrder_RemoveService_InvalidStatus(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	deps.creationSvc.On("RemoveService", mock.Anything, woID, wosID).Return(service.ErrWorkOrderInvalidStatusForItems)

	req := httptest.NewRequest(http.MethodDelete, "/work-orders/"+woID.String()+"/services/"+wosID.String(), nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.StatusCode)
}

// --- AddSupplies tests ---

func TestWorkOrder_AddSupplies_Success(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	result := []domain.WorkOrderServiceSupply{{ID: uuid.New()}}
	deps.creationSvc.On("AddSupplies", mock.Anything, woID, wosID, mock.Anything).Return(result, nil)
	deps.woSvc.On("GetByID", mock.Anything, woID).Return(&domain.WorkOrder{ID: woID, Status: domain.WorkOrderStatusInDiagnosis}, nil)

	body, _ := json.Marshal([]map[string]any{{"supply_id": uuid.New(), "quantity": 2}})
	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+woID.String()+"/services/"+wosID.String()+"/supplies", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, r.StatusCode)
}

func TestWorkOrder_AddSupplies_InvalidWoID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()
	req := httptest.NewRequest(http.MethodPost, "/work-orders/bad/services/"+uuid.New().String()+"/supplies", bytes.NewReader([]byte("[]")))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_AddSupplies_InvalidWosID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()
	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+uuid.New().String()+"/services/bad/supplies", bytes.NewReader([]byte("[]")))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_AddSupplies_Ownership(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	deps.creationSvc.On("AddSupplies", mock.Anything, woID, wosID, mock.Anything).Return(nil, service.ErrWorkOrderServiceOwnership)

	body, _ := json.Marshal([]map[string]any{{"supply_id": uuid.New(), "quantity": 1}})
	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+woID.String()+"/services/"+wosID.String()+"/supplies", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.StatusCode)
}

// --- StartService tests ---

func TestWorkOrder_StartService_Success(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	deps.creationSvc.On("StartService", mock.Anything, woID, wosID).Return(nil)
	deps.woSvc.On("GetByID", mock.Anything, woID).Return(&domain.WorkOrder{ID: woID, Status: domain.WorkOrderStatusInProgress}, nil)

	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+woID.String()+"/services/"+wosID.String()+"/start", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, r.StatusCode)
}

func TestWorkOrder_StartService_WithDelay(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	deps.creationSvc.On("StartService", mock.Anything, woID, wosID).Return(nil)
	deps.woSvc.On("GetByID", mock.Anything, woID).Return(&domain.WorkOrder{ID: woID, Status: domain.WorkOrderStatusInProgress}, nil)

	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+woID.String()+"/services/"+wosID.String()+"/start", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, r.StatusCode)
}

func TestWorkOrder_StartService_InvalidWoID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()
	req := httptest.NewRequest(http.MethodPost, "/work-orders/bad/services/"+uuid.New().String()+"/start", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_StartService_InvalidWosID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()
	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+uuid.New().String()+"/services/bad/start", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_StartService_NotInProgress(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	deps.creationSvc.On("StartService", mock.Anything, woID, wosID).Return(service.ErrWorkOrderNotInProgress)

	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+woID.String()+"/services/"+wosID.String()+"/start", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.StatusCode)
}

func TestWorkOrder_StartService_NotApproved(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	deps.creationSvc.On("StartService", mock.Anything, woID, wosID).Return(service.ErrServiceNotApproved)

	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+woID.String()+"/services/"+wosID.String()+"/start", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.StatusCode)
}

func TestWorkOrder_StartService_NotPending(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	deps.creationSvc.On("StartService", mock.Anything, woID, wosID).Return(service.ErrServiceNotPending)

	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+woID.String()+"/services/"+wosID.String()+"/start", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.StatusCode)
}

// --- FinalizeService tests ---

func TestWorkOrder_FinalizeService_Success(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	deps.creationSvc.On("FinalizeService", mock.Anything, woID, wosID).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+woID.String()+"/services/"+wosID.String()+"/finalize", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, r.StatusCode)
}

func TestWorkOrder_FinalizeService_InvalidWoID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()
	req := httptest.NewRequest(http.MethodPost, "/work-orders/bad/services/"+uuid.New().String()+"/finalize", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_FinalizeService_InvalidWosID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()
	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+uuid.New().String()+"/services/bad/finalize", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_FinalizeService_NotInProgress(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	deps.creationSvc.On("FinalizeService", mock.Anything, woID, wosID).Return(service.ErrWorkOrderNotInProgress)

	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+woID.String()+"/services/"+wosID.String()+"/finalize", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.StatusCode)
}

func TestWorkOrder_FinalizeService_ServiceNotInProgress(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	deps.creationSvc.On("FinalizeService", mock.Anything, woID, wosID).Return(service.ErrServiceNotInProgress)

	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+woID.String()+"/services/"+wosID.String()+"/finalize", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.StatusCode)
}

// --- RemoveSupplyFromService tests ---

func TestWorkOrder_RemoveSupply_Success(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	supplyID := uuid.New()
	deps.creationSvc.On("RemoveSupplyFromService", mock.Anything, woID, wosID, supplyID).Return(nil)
	deps.woSvc.On("GetByID", mock.Anything, woID).Return(&domain.WorkOrder{ID: woID, Status: domain.WorkOrderStatusInDiagnosis}, nil)

	req := httptest.NewRequest(http.MethodDelete, "/work-orders/"+woID.String()+"/services/"+wosID.String()+"/supplies/"+supplyID.String(), nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, r.StatusCode)
}

func TestWorkOrder_RemoveSupply_InvalidWoID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()
	req := httptest.NewRequest(http.MethodDelete, "/work-orders/bad/services/"+uuid.New().String()+"/supplies/"+uuid.New().String(), nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_RemoveSupply_InvalidWosID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()
	req := httptest.NewRequest(http.MethodDelete, "/work-orders/"+uuid.New().String()+"/services/bad/supplies/"+uuid.New().String(), nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_RemoveSupply_InvalidSupplyID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()
	req := httptest.NewRequest(http.MethodDelete, "/work-orders/"+uuid.New().String()+"/services/"+uuid.New().String()+"/supplies/bad", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_RemoveSupply_Ownership(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	supplyID := uuid.New()
	deps.creationSvc.On("RemoveSupplyFromService", mock.Anything, woID, wosID, supplyID).Return(service.ErrWorkOrderServiceOwnership)

	req := httptest.NewRequest(http.MethodDelete, "/work-orders/"+woID.String()+"/services/"+wosID.String()+"/supplies/"+supplyID.String(), nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.StatusCode)
}

func TestWorkOrder_RemoveSupply_InvalidStatus(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	supplyID := uuid.New()
	deps.creationSvc.On("RemoveSupplyFromService", mock.Anything, woID, wosID, supplyID).Return(service.ErrWorkOrderInvalidStatusForItems)

	req := httptest.NewRequest(http.MethodDelete, "/work-orders/"+woID.String()+"/services/"+wosID.String()+"/supplies/"+supplyID.String(), nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.StatusCode)
}
