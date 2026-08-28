package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/repository"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupIntegrationApp(t *testing.T) (*fiber.App, *pgxpool.Pool) {
	t.Helper()

	host := os.Getenv("DATABASE_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("DATABASE_PORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("DATABASE_USER")
	if user == "" {
		user = "techchallenge"
	}
	password := os.Getenv("DATABASE_PASSWORD")
	if password == "" {
		t.Skip("skipping integration test: DATABASE_PASSWORD not set")
	}
	dbName := os.Getenv("DATABASE_NAME")
	if dbName == "" {
		dbName = "techchallenge-db"
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, password, host, port, dbName)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("skipping integration test: cannot connect to database: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Skipf("skipping integration test: cannot ping database: %v", err)
	}

	setupSchema(t, pool)

	repo := repository.NewWorkshopServiceRepository(pool)
	svc := service.NewWorkshopServiceManager(repo)
	h := NewWorkshopServiceHandler(svc)

	app := fiber.New()
	h.RegisterRoutes(app)

	return app, pool
}

func setupSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	pool.Exec(ctx, `DROP TABLE IF EXISTS work_order_services CASCADE`)
	pool.Exec(ctx, `DROP TABLE IF EXISTS work_orders CASCADE`)
	pool.Exec(ctx, `DROP TABLE IF EXISTS services CASCADE`)

	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS "services" (
			"id" uuid PRIMARY KEY,
			"title" varchar(120) NOT NULL,
			"description" text,
			"price_cents" int NOT NULL,
			"estimated_time_minutes" int NOT NULL,
			"status" varchar(30) NOT NULL DEFAULT 'ATIVO',
			"active" boolean NOT NULL DEFAULT true,
			"created_at" timestamp NOT NULL,
			"updated_at" timestamp NOT NULL
		)`)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_services_title_unique ON services (LOWER(title))`)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS "work_orders" (
			"id" uuid PRIMARY KEY,
			"code" varchar(30) NOT NULL,
			"title" varchar(150) NOT NULL,
			"description" text,
			"customer_id" uuid NOT NULL,
			"vehicle_id" uuid NOT NULL,
			"opened_by_user_id" uuid NOT NULL,
			"assigned_technician_id" uuid,
			"status" varchar(30) NOT NULL,
			"total_estimated_price_cents" int NOT NULL DEFAULT 0,
			"received_at" timestamp NOT NULL,
			"quote_sent_at" timestamp,
			"approved_at" timestamp,
			"started_at" timestamp,
			"finished_at" timestamp,
			"delivered_at" timestamp,
			"created_at" timestamp NOT NULL,
			"updated_at" timestamp NOT NULL
		)`)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS "work_order_services" (
			"id" uuid PRIMARY KEY,
			"work_order_id" uuid NOT NULL REFERENCES "work_orders" ("id"),
			"service_id" uuid NOT NULL,
			"service_title_snapshot" varchar(120) NOT NULL,
			"service_description_snapshot" text,
			"service_price_cents_snapshot" int NOT NULL,
			"service_estimated_time_minutes_snapshot" int NOT NULL,
			"approval_status" varchar(20) NOT NULL DEFAULT 'PENDENTE',
			"status" varchar(30) NOT NULL DEFAULT 'PENDENTE',
			"started_at" timestamp,
			"finished_at" timestamp,
			"created_at" timestamp NOT NULL,
			"updated_at" timestamp NOT NULL
		)`)
	require.NoError(t, err)

	t.Cleanup(func() {
		pool.Exec(ctx, `DROP TABLE IF EXISTS work_order_services CASCADE`)
		pool.Exec(ctx, `DROP TABLE IF EXISTS work_orders CASCADE`)
		pool.Exec(ctx, `DROP TABLE IF EXISTS services CASCADE`)
		pool.Close()
	})
}

func postJSON(app *fiber.App, path string, body any) (*http.Response, error) {
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	return app.Test(req)
}

func putJSON(app *fiber.App, path string, body any) (*http.Response, error) {
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("PUT", path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	return app.Test(req)
}

func readBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()
	var result map[string]any
	require.NoError(t, json.Unmarshal(data, &result))
	return result
}

func readBodyArray(t *testing.T, resp *http.Response) []map[string]any {
	t.Helper()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()
	var result []map[string]any
	require.NoError(t, json.Unmarshal(data, &result))
	return result
}

func TestIntegration_CreateAndGetByID(t *testing.T) {
	// should create a service and retrieve it by ID
	app, _ := setupIntegrationApp(t)

	resp, err := postJSON(app, "/services", map[string]any{
		"title":                  "Oil Change",
		"description":            "Full engine oil change",
		"price_cents":            5000,
		"estimated_time_minutes": 30,
	})
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	created := readBody(t, resp)
	assert.Equal(t, "Oil Change", created["title"])
	assert.Equal(t, float64(5000), created["price_cents"])
	assert.Equal(t, float64(30), created["estimated_time_minutes"])
	assert.Equal(t, true, created["active"])

	id := created["id"].(string)
	req, _ := http.NewRequest("GET", "/services/"+id, nil)
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	fetched := readBody(t, resp)
	assert.Equal(t, id, fetched["id"])
	assert.Equal(t, "Oil Change", fetched["title"])
}

func TestIntegration_CreateDuplicateTitle(t *testing.T) {
	// should return 409 when creating a service with duplicate title
	app, _ := setupIntegrationApp(t)

	body := map[string]any{
		"title":                  "Wheel Alignment",
		"price_cents":            8000,
		"estimated_time_minutes": 45,
	}

	resp, err := postJSON(app, "/services", body)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	resp, err = postJSON(app, "/services", body)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
}

func TestIntegration_CreateValidationError(t *testing.T) {
	// should return 400 when required fields are missing
	app, _ := setupIntegrationApp(t)

	resp, err := postJSON(app, "/services", map[string]any{"description": "only description"})
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestIntegration_ListPaginated(t *testing.T) {
	// should list services with correct pagination
	app, _ := setupIntegrationApp(t)

	for i := 1; i <= 3; i++ {
		resp, err := postJSON(app, "/services", map[string]any{
			"title":                  fmt.Sprintf("Service %d", i),
			"price_cents":            i * 1000,
			"estimated_time_minutes": i * 15,
		})
		require.NoError(t, err)
		require.Equal(t, fiber.StatusCreated, resp.StatusCode)
	}

	req, _ := http.NewRequest("GET", "/services?page=1&limit=2", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	result := readBody(t, resp)
	assert.Equal(t, float64(3), result["total"])
	assert.Len(t, result["data"], 2)
	assert.Equal(t, float64(1), result["page"])
	assert.Equal(t, float64(2), result["limit"])
}

func TestIntegration_ListFilterByTitle(t *testing.T) {
	// should filter services by partial title match
	app, _ := setupIntegrationApp(t)

	for _, title := range []string{"Oil Change", "Alignment", "Tire Change"} {
		resp, _ := postJSON(app, "/services", map[string]any{
			"title":                  title,
			"price_cents":            5000,
			"estimated_time_minutes": 30,
		})
		require.Equal(t, fiber.StatusCreated, resp.StatusCode)
	}

	req, _ := http.NewRequest("GET", "/services?title=Change", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	result := readBody(t, resp)
	assert.Equal(t, float64(2), result["total"])
}

func TestIntegration_ListFilterByActive(t *testing.T) {
	// should filter services by active status
	app, _ := setupIntegrationApp(t)

	resp, _ := postJSON(app, "/services", map[string]any{
		"title": "Active One", "price_cents": 1000, "estimated_time_minutes": 10,
	})
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)
	created := readBody(t, resp)
	id := created["id"].(string)

	resp, _ = putJSON(app, "/services/"+id, map[string]any{"active": false})
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	resp, _ = postJSON(app, "/services", map[string]any{
		"title": "Active Two", "price_cents": 2000, "estimated_time_minutes": 20,
	})
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)

	req, _ := http.NewRequest("GET", "/services?active=true", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	result := readBody(t, resp)
	assert.Equal(t, float64(1), result["total"])
}

func TestIntegration_GetByID_NotFound(t *testing.T) {
	// should return 404 for non-existent ID
	app, _ := setupIntegrationApp(t)

	req, _ := http.NewRequest("GET", "/services/"+uuid.New().String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestIntegration_Update(t *testing.T) {
	// should update fields and persist changes
	app, _ := setupIntegrationApp(t)

	resp, _ := postJSON(app, "/services", map[string]any{
		"title": "Original", "price_cents": 1000, "estimated_time_minutes": 10,
	})
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)
	created := readBody(t, resp)
	id := created["id"].(string)

	resp, err := putJSON(app, "/services/"+id, map[string]any{
		"title":       "Updated",
		"price_cents": 9999,
	})
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	updated := readBody(t, resp)
	assert.Equal(t, "Updated", updated["title"])
	assert.Equal(t, float64(9999), updated["price_cents"])

	req, _ := http.NewRequest("GET", "/services/"+id, nil)
	resp, _ = app.Test(req)
	fetched := readBody(t, resp)
	assert.Equal(t, "Updated", fetched["title"])
}

func TestIntegration_Update_NotFound(t *testing.T) {
	// should return 404 when updating non-existent service
	app, _ := setupIntegrationApp(t)

	resp, err := putJSON(app, "/services/"+uuid.New().String(), map[string]any{"title": "Test"})
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestIntegration_DeleteHard(t *testing.T) {
	// should hard delete service with no work order links and return 204
	app, _ := setupIntegrationApp(t)

	resp, _ := postJSON(app, "/services", map[string]any{
		"title": "To Delete", "price_cents": 1000, "estimated_time_minutes": 10,
	})
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)
	created := readBody(t, resp)
	id := created["id"].(string)

	req, _ := http.NewRequest("DELETE", "/services/"+id, nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)

	req, _ = http.NewRequest("GET", "/services/"+id, nil)
	resp, _ = app.Test(req)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestIntegration_DeleteSoft(t *testing.T) {
	// should soft delete (deactivate) service that has work order links and return 200
	app, pool := setupIntegrationApp(t)

	resp, _ := postJSON(app, "/services", map[string]any{
		"title": "Linked Service", "price_cents": 1000, "estimated_time_minutes": 10,
	})
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)
	created := readBody(t, resp)
	serviceID := created["id"].(string)

	woID := uuid.New()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO work_orders
			(id, code, title, customer_id, vehicle_id, opened_by_user_id, status, received_at, created_at, updated_at)
		VALUES ($1, 'WO-TEST', 'test', $2, $2, $2, 'RECEBIDA', NOW(), NOW(), NOW())`, woID, uuid.New())
	require.NoError(t, err)

	_, err = pool.Exec(context.Background(), `
		INSERT INTO work_order_services
			(id, work_order_id, service_id, service_title_snapshot, service_price_cents_snapshot, service_estimated_time_minutes_snapshot, created_at, updated_at)
		VALUES ($1, $2, $3, 'snap', 1000, 30, NOW(), NOW())`,
		uuid.New(), woID, serviceID,
	)
	require.NoError(t, err)

	req, _ := http.NewRequest("DELETE", "/services/"+serviceID, nil)
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	result := readBody(t, resp)
	assert.Contains(t, result["message"], "deactivated")

	req, _ = http.NewRequest("GET", "/services/"+serviceID, nil)
	resp, _ = app.Test(req)
	fetched := readBody(t, resp)
	assert.Equal(t, false, fetched["active"])
}

func TestIntegration_Delete_NotFound(t *testing.T) {
	// should return 404 when deleting non-existent service
	app, _ := setupIntegrationApp(t)

	req, _ := http.NewRequest("DELETE", "/services/"+uuid.New().String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestIntegration_AvgExecutionTime(t *testing.T) {
	// should return avg execution time based on finished work order services
	app, pool := setupIntegrationApp(t)
	ctx := context.Background()

	resp, _ := postJSON(app, "/services", map[string]any{
		"title": "Oil Change", "price_cents": 5000, "estimated_time_minutes": 30,
	})
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)
	created := readBody(t, resp)
	serviceID := created["id"].(string)

	techID := uuid.New()
	woID1 := uuid.New()
	woID2 := uuid.New()

	// create work orders for the services
	for _, woID := range []uuid.UUID{woID1, woID2} {
		_, err := pool.Exec(ctx, `
			INSERT INTO work_orders
				(id, code, title, customer_id, vehicle_id, opened_by_user_id, assigned_technician_id, status, received_at, created_at, updated_at)
			VALUES ($1, $2, 'test', $3, $3, $3, $4, 'RECEBIDA', NOW(), NOW(), NOW())`,
			woID, "WO-"+woID.String()[:8], uuid.New(), techID)
		require.NoError(t, err)
	}

	// insert two finished work order services with known durations
	_, err := pool.Exec(ctx, `
		INSERT INTO work_order_services
			(id, work_order_id, service_id, service_title_snapshot, service_price_cents_snapshot,
			 service_estimated_time_minutes_snapshot, status, started_at, finished_at, created_at, updated_at)
		VALUES
			($1, $2, $3, 'Oil Change', 5000, 30, $4,
			 '2026-04-01 10:00:00', '2026-04-01 10:25:00', NOW(), NOW()),
			($5, $6, $3, 'Oil Change', 5000, 30, $4,
			 '2026-04-02 14:00:00', '2026-04-02 14:35:00', NOW(), NOW())`,
		uuid.New(), woID1, serviceID, string(domain.WorkOrderServiceStatusFinished),
		uuid.New(), woID2,
	)
	require.NoError(t, err)

	req, _ := http.NewRequest("GET", "/services/avg-execution-time", nil)
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	result := readBody(t, resp)
	items := result["data"].([]any)
	require.Len(t, items, 1)
	first := items[0].(map[string]any)
	assert.Equal(t, "Oil Change", first["title"])
	assert.Equal(t, float64(2), first["execution_count"])
	assert.Equal(t, float64(30), first["avg_real_time_minutes"])
	assert.Equal(t, float64(30), first["estimated_time_minutes"])
}

func TestIntegration_AvgExecutionTime_ConvertsSecondsToMinutes(t *testing.T) {
	// should expose real execution time in minutes, not raw seconds
	app, pool := setupIntegrationApp(t)
	ctx := context.Background()

	resp, _ := postJSON(app, "/services", map[string]any{
		"title": "Quick Check", "price_cents": 1500, "estimated_time_minutes": 2,
	})
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)
	created := readBody(t, resp)
	serviceID := created["id"].(string)

	woID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO work_orders
			(id, code, title, customer_id, vehicle_id, opened_by_user_id, status, received_at, created_at, updated_at)
		VALUES ($1, 'WO-SECONDS', 'test', $2, $2, $2, 'RECEBIDA', NOW(), NOW(), NOW())`,
		woID, uuid.New())
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO work_order_services
			(id, work_order_id, service_id, service_title_snapshot, service_price_cents_snapshot,
			 service_estimated_time_minutes_snapshot, status, started_at, finished_at, created_at, updated_at)
		VALUES
			($1, $2, $3, 'Quick Check', 1500, 2, $4,
			 '2026-04-01 10:00:00', '2026-04-01 10:01:30', NOW(), NOW())`,
		uuid.New(), woID, serviceID, string(domain.WorkOrderServiceStatusFinished),
	)
	require.NoError(t, err)

	req, _ := http.NewRequest("GET", "/services/avg-execution-time", nil)
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	result := readBody(t, resp)
	items := result["data"].([]any)
	require.Len(t, items, 1)
	first := items[0].(map[string]any)
	assert.Equal(t, float64(1.5), first["avg_real_time_minutes"])
	assert.Equal(t, float64(-0.5), first["difference_minutes"])
}

func TestIntegration_AvgExecutionTime_FilterByTechnician(t *testing.T) {
	// should filter avg execution time by technician ID
	app, pool := setupIntegrationApp(t)
	ctx := context.Background()

	resp, _ := postJSON(app, "/services", map[string]any{
		"title": "Alignment", "price_cents": 8000, "estimated_time_minutes": 60,
	})
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)
	created := readBody(t, resp)
	serviceID := created["id"].(string)

	techA := uuid.New()
	techB := uuid.New()
	woID1 := uuid.New()
	woID2 := uuid.New()

	// create work orders assigned to different technicians
	_, err := pool.Exec(ctx, `
		INSERT INTO work_orders
			(id, code, title, customer_id, vehicle_id, opened_by_user_id, assigned_technician_id, status, received_at, created_at, updated_at)
		VALUES
			($1, 'WO-A', 'test', $3, $3, $3, $4, 'RECEBIDA', NOW(), NOW(), NOW()),
			($2, 'WO-B', 'test', $3, $3, $3, $5, 'RECEBIDA', NOW(), NOW(), NOW())`,
		woID1, woID2, uuid.New(), techA, techB)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO work_order_services
			(id, work_order_id, service_id, service_title_snapshot, service_price_cents_snapshot,
			 service_estimated_time_minutes_snapshot, status, started_at, finished_at, created_at, updated_at)
		VALUES
			($1, $2, $3, 'Alignment', 8000, 60, $5,
			 '2026-04-01 10:00:00', '2026-04-01 11:00:00', NOW(), NOW()),
			($4, $6, $3, 'Alignment', 8000, 60, $5,
			 '2026-04-02 10:00:00', '2026-04-02 10:45:00', NOW(), NOW())`,
		uuid.New(), woID1, serviceID, uuid.New(), string(domain.WorkOrderServiceStatusFinished), woID2,
	)
	require.NoError(t, err)

	req, _ := http.NewRequest("GET", "/services/avg-execution-time?technicianId="+techA.String(), nil)
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	result := readBody(t, resp)
	items := result["data"].([]any)
	require.Len(t, items, 1)
	first := items[0].(map[string]any)
	assert.Equal(t, float64(1), first["execution_count"])
	assert.Equal(t, float64(60), first["avg_real_time_minutes"])
}

func TestIntegration_AvgExecutionTime_Empty(t *testing.T) {
	// should return empty array when no finished services exist
	app, _ := setupIntegrationApp(t)

	req, _ := http.NewRequest("GET", "/services/avg-execution-time", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	result := readBody(t, resp)
	items := result["data"].([]any)
	assert.Len(t, items, 0)
}
