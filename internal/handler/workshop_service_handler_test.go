package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockWorkshopServiceService struct {
	mock.Mock
}

func (m *mockWorkshopServiceService) Create(ctx context.Context, ws *domain.WorkshopService) (*domain.WorkshopService, error) {
	args := m.Called(ctx, ws)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkshopService), args.Error(1)
}

func (m *mockWorkshopServiceService) GetByID(ctx context.Context, id uuid.UUID) (*domain.WorkshopService, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkshopService), args.Error(1)
}

func (m *mockWorkshopServiceService) List(ctx context.Context, filters domain.WorkshopServiceListFilters) ([]domain.WorkshopService, int, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).([]domain.WorkshopService), args.Int(1), args.Error(2)
}

func (m *mockWorkshopServiceService) Update(ctx context.Context, id uuid.UUID, input service.WorkshopServiceUpdateInput) (*domain.WorkshopService, error) {
	args := m.Called(ctx, id, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkshopService), args.Error(1)
}

func (m *mockWorkshopServiceService) Delete(ctx context.Context, id uuid.UUID) (*service.DeleteWorkshopServiceResult, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.DeleteWorkshopServiceResult), args.Error(1)
}

func (m *mockWorkshopServiceService) GetAvgExecutionTime(ctx context.Context, filters domain.AvgExecutionTimeFilters) ([]domain.AvgExecutionTimeResult, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).([]domain.AvgExecutionTimeResult), args.Error(1)
}

func setupTestApp(svc service.WorkshopServiceManager) *fiber.App {
	app := fiber.New()
	h := NewWorkshopServiceHandler(svc)
	h.RegisterRoutes(app)
	return app
}

func sampleService() *domain.WorkshopService {
	return &domain.WorkshopService{
		ID:                   uuid.New(),
		Title:                "Oil Change",
		Description:          "Full oil change",
		PriceCents:           5000,
		EstimatedTimeMinutes: 30,
		Active:               true,
		CreatedAt:            time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:            time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func parseBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()
	var result map[string]any
	require.NoError(t, json.Unmarshal(body, &result))
	return result
}

func parseBodyArray(t *testing.T, resp *http.Response) []map[string]any {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()
	var result []map[string]any
	require.NoError(t, json.Unmarshal(body, &result))
	return result
}

func TestCreateRoute_201(t *testing.T) {
	// POST /services should return 201 with valid data
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)

	created := sampleService()
	svcMock.On("Create", mock.Anything, mock.AnythingOfType("*domain.WorkshopService")).Return(created, nil)

	body, _ := json.Marshal(map[string]any{
		"title":                  "Oil Change",
		"description":            "Full oil change",
		"price_cents":            5000,
		"estimated_time_minutes": 30,
	})

	req, _ := http.NewRequest("POST", "/services", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	result := parseBody(t, resp)
	assert.Equal(t, "Oil Change", result["title"])
	assert.Equal(t, float64(5000), result["price_cents"])
}

func TestCreateRoute_400_MissingFields(t *testing.T) {
	// POST /services should return 400 when required fields are missing
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)

	body, _ := json.Marshal(map[string]any{"description": "only description"})
	req, _ := http.NewRequest("POST", "/services", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestCreateRoute_400_InvalidBody(t *testing.T) {
	// POST /services should return 400 with malformed JSON
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)

	req, _ := http.NewRequest("POST", "/services", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestCreateRoute_409_DuplicateTitle(t *testing.T) {
	// POST /services should return 409 when title already exists
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)

	svcMock.On("Create", mock.Anything, mock.Anything).Return(nil, service.ErrWorkshopServiceTitleAlreadyExists)

	body, _ := json.Marshal(map[string]any{
		"title":                  "Duplicate",
		"price_cents":            1000,
		"estimated_time_minutes": 15,
	})
	req, _ := http.NewRequest("POST", "/services", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
}

func TestCreateRoute_400_ValidationError(t *testing.T) {
	// POST /services should return 400 when domain validation fails
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)

	svcMock.On("Create", mock.Anything, mock.Anything).Return(nil, domain.ErrWorkshopServicePriceMustBePositive)

	body, _ := json.Marshal(map[string]any{
		"title":                  "Test",
		"price_cents":            -100,
		"estimated_time_minutes": 30,
	})
	req, _ := http.NewRequest("POST", "/services", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGetAllRoute_200(t *testing.T) {
	// GET /services should return 200 with paginated list
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)

	items := []domain.WorkshopService{*sampleService()}
	svcMock.On("List", mock.Anything, mock.AnythingOfType("domain.WorkshopServiceListFilters")).Return(items, 1, nil)

	req, _ := http.NewRequest("GET", "/services", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	result := parseBody(t, resp)
	assert.Equal(t, float64(1), result["total"])
	assert.Equal(t, float64(1), result["total_pages"])
	assert.Len(t, result["data"], 1)
}

func TestGetAllRoute_200_EmptyList(t *testing.T) {
	// GET /services should return 200 with empty items when no services exist
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)

	svcMock.On("List", mock.Anything, mock.Anything).Return([]domain.WorkshopService{}, 0, nil)

	req, _ := http.NewRequest("GET", "/services", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	result := parseBody(t, resp)
	assert.Equal(t, float64(0), result["total"])
	assert.Len(t, result["data"], 0)
}

func TestGetAllRoute_200_WithFilters(t *testing.T) {
	// GET /services?active=true&title=oil should pass filters correctly
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)

	svcMock.On("List", mock.Anything, mock.AnythingOfType("domain.WorkshopServiceListFilters")).Return([]domain.WorkshopService{}, 0, nil)

	req, _ := http.NewRequest("GET", "/services?active=true&title=oil&page=2&limit=5", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	result := parseBody(t, resp)
	assert.Equal(t, float64(2), result["page"])
	assert.Equal(t, float64(5), result["limit"])
}

func TestGetAllRoute_400_InvalidPage(t *testing.T) {
	// GET /services?page=-1 should return 400
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)

	req, _ := http.NewRequest("GET", "/services?page=-1", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGetAllRoute_400_InvalidActive(t *testing.T) {
	// GET /services?active=invalid should return 400
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)

	req, _ := http.NewRequest("GET", "/services?active=invalid", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGetByIDRoute_200(t *testing.T) {
	// GET /services/:id should return 200 with the service
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)
	ws := sampleService()

	svcMock.On("GetByID", mock.Anything, ws.ID).Return(ws, nil)

	req, _ := http.NewRequest("GET", "/services/"+ws.ID.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	result := parseBody(t, resp)
	assert.Equal(t, ws.ID.String(), result["id"])
}

func TestGetByIDRoute_404(t *testing.T) {
	// GET /services/:id should return 404 when not found
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)
	id := uuid.New()

	svcMock.On("GetByID", mock.Anything, id).Return(nil, pgx.ErrNoRows)

	req, _ := http.NewRequest("GET", "/services/"+id.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestGetByIDRoute_400_InvalidID(t *testing.T) {
	// GET /services/:id should return 400 for invalid UUID
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)

	req, _ := http.NewRequest("GET", "/services/not-a-uuid", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestUpdateRoute_200(t *testing.T) {
	// PUT /services/:id should return 200 with updated service
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)
	ws := sampleService()

	updated := *ws
	updated.Title = "Alignment"
	svcMock.On("Update", mock.Anything, ws.ID, mock.AnythingOfType("service.WorkshopServiceUpdateInput")).Return(&updated, nil)

	body, _ := json.Marshal(map[string]any{"title": "Alignment"})
	req, _ := http.NewRequest("PUT", "/services/"+ws.ID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	result := parseBody(t, resp)
	assert.Equal(t, "Alignment", result["title"])
}

func TestUpdateRoute_404(t *testing.T) {
	// PUT /services/:id should return 404 when service not found
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)
	id := uuid.New()

	svcMock.On("Update", mock.Anything, id, mock.Anything).Return(nil, pgx.ErrNoRows)

	body, _ := json.Marshal(map[string]any{"title": "Test"})
	req, _ := http.NewRequest("PUT", "/services/"+id.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestUpdateRoute_400_EmptyBody(t *testing.T) {
	// PUT /services/:id should return 400 when no fields provided
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)
	id := uuid.New()

	body, _ := json.Marshal(map[string]any{})
	req, _ := http.NewRequest("PUT", "/services/"+id.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestUpdateRoute_409_DuplicateTitle(t *testing.T) {
	// PUT /services/:id should return 409 when updated title conflicts
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)
	id := uuid.New()

	svcMock.On("Update", mock.Anything, id, mock.Anything).Return(nil, service.ErrWorkshopServiceTitleAlreadyExists)

	body, _ := json.Marshal(map[string]any{"title": "Duplicate"})
	req, _ := http.NewRequest("PUT", "/services/"+id.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
}

func TestDeleteRoute_204(t *testing.T) {
	// DELETE /services/:id should return 204 when hard deleted
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)
	id := uuid.New()

	svcMock.On("Delete", mock.Anything, id).Return(&service.DeleteWorkshopServiceResult{Deleted: true}, nil)

	req, _ := http.NewRequest("DELETE", "/services/"+id.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestDeleteRoute_200_Deactivated(t *testing.T) {
	// DELETE /services/:id should return 200 with message when soft deleted
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)
	ws := sampleService()
	ws.Active = false

	svcMock.On("Delete", mock.Anything, ws.ID).Return(&service.DeleteWorkshopServiceResult{
		Deactivated:         true,
		DeactivatedResource: ws,
	}, nil)

	req, _ := http.NewRequest("DELETE", "/services/"+ws.ID.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	result := parseBody(t, resp)
	assert.Contains(t, result["message"], "deactivated")
	assert.NotNil(t, result["service"])
}

func TestDeleteRoute_404(t *testing.T) {
	// DELETE /services/:id should return 404 when service not found
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)
	id := uuid.New()

	svcMock.On("Delete", mock.Anything, id).Return(nil, pgx.ErrNoRows)

	req, _ := http.NewRequest("DELETE", "/services/"+id.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestDeleteRoute_400_InvalidID(t *testing.T) {
	// DELETE /services/:id should return 400 for invalid UUID
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)

	req, _ := http.NewRequest("DELETE", "/services/invalid", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGetAvgExecutionTime_200(t *testing.T) {
	// GET /services/avg-execution-time should return 200 with results
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)

	results := []domain.AvgExecutionTimeResult{
		{
			ServiceID:            uuid.New(),
			Title:                "Oil Change",
			EstimatedTimeMinutes: 30,
			AvgRealTimeMinutes:   25.5,
			ExecutionCount:       3,
		},
	}
	svcMock.On("GetAvgExecutionTime", mock.Anything, domain.AvgExecutionTimeFilters{}).Return(results, nil)

	req, _ := http.NewRequest("GET", "/services/avg-execution-time", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	result := parseBody(t, resp)
	items := result["data"].([]any)
	assert.Len(t, items, 1)
	first := items[0].(map[string]any)
	assert.Equal(t, "Oil Change", first["title"])
	assert.Equal(t, float64(25.5), first["avg_real_time_minutes"])
	assert.Equal(t, float64(3), first["execution_count"])
	assert.Equal(t, float64(-4.5), first["difference_minutes"])
}

func TestGetAvgExecutionTime_200_Empty(t *testing.T) {
	// GET /services/avg-execution-time should return 200 with empty array when no data
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)

	svcMock.On("GetAvgExecutionTime", mock.Anything, domain.AvgExecutionTimeFilters{}).Return([]domain.AvgExecutionTimeResult{}, nil)

	req, _ := http.NewRequest("GET", "/services/avg-execution-time", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	result := parseBody(t, resp)
	items := result["data"].([]any)
	assert.Len(t, items, 0)
}

func TestGetAvgExecutionTime_400_InvalidFromDate(t *testing.T) {
	// GET /services/avg-execution-time?from=bad should return 400
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)

	req, _ := http.NewRequest("GET", "/services/avg-execution-time?from=bad-date", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGetAvgExecutionTime_400_InvalidToDate(t *testing.T) {
	// GET /services/avg-execution-time?to=bad should return 400
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)

	req, _ := http.NewRequest("GET", "/services/avg-execution-time?to=bad-date", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGetAvgExecutionTime_400_InvalidTechnicianId(t *testing.T) {
	// GET /services/avg-execution-time?technicianId=bad should return 400
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)

	req, _ := http.NewRequest("GET", "/services/avg-execution-time?technicianId=not-uuid", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGetAvgExecutionTime_WithFilters(t *testing.T) {
	// GET /services/avg-execution-time with valid filters should pass them through
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)

	svcMock.On("GetAvgExecutionTime", mock.Anything, mock.AnythingOfType("domain.AvgExecutionTimeFilters")).Return([]domain.AvgExecutionTimeResult{}, nil)

	techID := uuid.New()
	req, _ := http.NewRequest("GET", "/services/avg-execution-time?from=2026-01-01&to=2026-12-31&technicianId="+techID.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestGetAvgExecutionTime_WithCanonicalTechnicianID(t *testing.T) {
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)
	techID := uuid.New()
	svcMock.On("GetAvgExecutionTime", mock.Anything, mock.MatchedBy(func(filters domain.AvgExecutionTimeFilters) bool {
		return filters.TechnicianID != nil && *filters.TechnicianID == techID
	})).Return([]domain.AvgExecutionTimeResult{}, nil)

	req, _ := http.NewRequest("GET", "/services/avg-execution-time?technician_id="+techID.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestGetAvgExecutionTime_RejectsConflictingTechnicianAliases(t *testing.T) {
	svcMock := new(mockWorkshopServiceService)
	app := setupTestApp(svcMock)
	req, _ := http.NewRequest("GET", "/services/avg-execution-time?technician_id="+uuid.NewString()+"&technicianId="+uuid.NewString(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}
