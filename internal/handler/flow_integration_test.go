package handler

// Integration tests that validate the complete workshop DDD flow
// defined in the event storming.
//
// Each test maps to a step (or group of steps) in the flow:
//
//   DDD Step                          → API Route
//   ─────────────────────────────────────────────────────────────
//   Consulta Cliente                  → GET  /customers?document=
//   Cadastra Cliente                  → POST /customers
//   Consulta Veículo                  → GET  /vehicles?customerId=
//   Cadastra Veículo                  → POST /vehicles
//   Cria OS (status RECEBIDA)         → POST /work-orders
//   Listagem de OS disponíveis        → GET  /work-orders?status=RECEBIDA
//   Pega OS (EM_DIAGNOSTICO)          → PUT  /work-orders/:id
//   Atualiza OS com serviços          → POST /work-orders/:id/services
//   Adiciona peças/insumos            → POST /work-orders/:id/services/:wosId/supplies
//   Gera orçamento (AGUARDANDO_APROV) → PUT  /work-orders/:id
//   Cliente consulta status           → GET  /public/work-orders/:code?document=
//   Aprova/Reprova serviços           → GET  /public/approvals/...
//   Em Execução                       → PUT  /work-orders/:id
//   Finalizada                        → PUT  /work-orders/:id
//   Entregue                          → PUT  /work-orders/:id

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"strings"
	"sync"

	dbmigrations "github.com/SOAT-15-Oficina/oficina-mecanica-monolith/database"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/auth"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/repository"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/service"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/packages/email"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// flowMockEmail captures all sent emails for assertion in tests.
type flowMockEmail struct {
	mu       sync.Mutex
	messages []email.Message
	failNext bool
}

func (m *flowMockEmail) Send(_ context.Context, msg email.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failNext {
		m.failNext = false
		return fmt.Errorf("smtp unavailable")
	}
	m.messages = append(m.messages, msg)
	return nil
}

func (m *flowMockEmail) Messages() []email.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]email.Message, len(m.messages))
	copy(cp, m.messages)
	return cp
}

func findLatestEmailBySubjectContains(msgs []email.Message, fragment string) (email.Message, bool) {
	var latest email.Message
	found := false
	for _, msg := range msgs {
		if strings.Contains(msg.Subject, fragment) {
			latest = msg
			found = true
		}
	}
	return latest, found
}

const testUsername = "mechanic_test"

// setupFlowApp creates a Fiber app wired with real repositories and services
// against a real PostgreSQL database, mirroring the full application setup.
// Auth middleware is replaced with a stub that injects test claims.
func setupFlowApp(t *testing.T) (*fiber.App, *pgxpool.Pool) {
	t.Helper()

	pool := connectTestDB(t)
	setupFlowSchema(t, pool)
	seedTestUser(t, pool)

	// Repositories
	customerRepo := repository.NewCustomerRepository(pool)
	vehicleRepo := repository.NewVehicleRepository(pool)
	supplyRepo := repository.NewSupplyRepository(pool)
	wsRepo := repository.NewWorkshopServiceRepository(pool)
	woRepo := repository.NewWorkOrderRepository(pool)
	wosRepo := repository.NewWorkOrderServiceRepository(pool)
	userRepo := repository.NewUserRepository(pool)

	// Services
	customerSvc := service.NewCustomerService(customerRepo)
	vehicleSvc := service.NewVehicleService(vehicleRepo)
	supplySvc := service.NewSupplyService(supplyRepo, wosRepo)
	wsSvc := service.NewWorkshopServiceManager(wsRepo)
	statusSvc := service.NewWorkOrderStatusServiceWithNotifications(woRepo, wosRepo, customerRepo, nil, "http://localhost:8080")
	woSvc := service.NewWorkOrderService(woRepo, vehicleRepo)
	budgetSvc := service.NewBudgetService(woRepo, wosRepo, customerRepo, nil, "http://localhost:8080")
	creationSvc := service.NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc, service.WithBudgetRefresh(budgetSvc))
	itemSvc := service.NewWorkOrderItemService(wosRepo, woRepo, statusSvc)
	userSvc := service.NewUserService(userRepo)
	publicSvc := service.NewPublicWorkOrderService(woRepo, customerRepo, wosRepo)

	// Handlers
	customerHandler := NewCustomerHandler(customerSvc)
	vehicleHandler := NewVehicleHandler(vehicleSvc)
	supplyHandler := NewSupplyHandler(supplySvc)
	wsHandler := NewWorkshopServiceHandler(wsSvc)
	woHandler := NewWorkOrderHandler(woSvc, creationSvc, statusSvc, userSvc)
	wosHandler := NewWorkOrderServiceHandler(itemSvc)
	publicWoHandler := NewPublicWorkOrderHandler(publicSvc)

	app := fiber.New()

	// Inject fake claims for protected routes
	fakeAuth := func(c fiber.Ctx) error {
		c.Locals("token", &auth.AppClaims{
			User: testUsername,
			Role: "employee",
		})
		return c.Next()
	}

	// Customer routes
	customers := app.Group("/customers", fakeAuth)
	customers.Post("/", customerHandler.Create)
	customers.Get("/", customerHandler.GetAll)
	customers.Get("/:id", customerHandler.GetByID)
	customers.Put("/:id", customerHandler.Update)
	customers.Delete("/:id", customerHandler.Delete)

	// Vehicle routes
	vehicles := app.Group("/vehicles", fakeAuth)
	vehicles.Post("/", vehicleHandler.Create)
	vehicles.Get("/", vehicleHandler.GetAll)
	vehicles.Get("/:id", vehicleHandler.GetByID)
	vehicles.Put("/:id", vehicleHandler.Update)
	vehicles.Delete("/:id", vehicleHandler.Delete)

	// Supply routes
	supplies := app.Group("/supplies", fakeAuth)
	supplies.Post("/", supplyHandler.Create)
	supplies.Get("/", supplyHandler.GetAll)
	supplies.Get("/:id", supplyHandler.GetByID)
	supplies.Put("/:id", supplyHandler.Update)
	supplies.Delete("/:id", supplyHandler.Delete)

	// Workshop service routes
	services := app.Group("/services", fakeAuth)
	services.Post("/", wsHandler.Create)
	services.Get("/", wsHandler.GetAll)
	services.Get("/:id", wsHandler.GetByID)
	services.Put("/:id", wsHandler.Update)
	services.Delete("/:id", wsHandler.Delete)

	// Work order routes
	workOrders := app.Group("/work-orders", fakeAuth)
	workOrders.Post("/", woHandler.Create)
	workOrders.Get("/", woHandler.GetAll)
	workOrders.Get("/:id", woHandler.GetByID)
	workOrders.Put("/:id", woHandler.Update)
	workOrders.Post("/:id/services", woHandler.AddServices)
	workOrders.Delete("/:id/services/:wosId", woHandler.RemoveService)
	workOrders.Put("/:id/services/:wosId/start", woHandler.StartService)
	workOrders.Put("/:id/services/:wosId/finalize", woHandler.FinalizeService)
	workOrders.Post("/:id/services/:wosId/supplies", woHandler.AddSupplies)
	workOrders.Delete("/:id/services/:wosId/supplies/:supplyId", woHandler.RemoveSupplyFromService)

	// Public approval routes (no auth)
	approval := app.Group("/public/approvals")
	approval.Get("/services/:workOrderServiceId/approve", wosHandler.Approve)
	approval.Get("/services/:workOrderServiceId/reject", wosHandler.Reject)
	approval.Get("/work-orders/:workOrderId/approve-all", wosHandler.ApproveAll)
	approval.Get("/work-orders/:workOrderId/reject-all", wosHandler.RejectAll)

	// Public work order status (no auth)
	publicWO := app.Group("/public/work-orders")
	publicWO.Get("/:code", publicWoHandler.GetByCode)

	return app, pool
}

// setupFlowAppWithBudget is like setupFlowApp but wires up the BudgetService
// with a mock email provider, so budget generation and notification tests work.
func setupFlowAppWithBudget(t *testing.T) (*fiber.App, *pgxpool.Pool, *flowMockEmail) {
	t.Helper()

	pool := connectTestDB(t)
	setupFlowSchema(t, pool)
	seedTestUser(t, pool)

	mockEmail := &flowMockEmail{}

	// Repositories
	customerRepo := repository.NewCustomerRepository(pool)
	vehicleRepo := repository.NewVehicleRepository(pool)
	supplyRepo := repository.NewSupplyRepository(pool)
	wsRepo := repository.NewWorkshopServiceRepository(pool)
	woRepo := repository.NewWorkOrderRepository(pool)
	wosRepo := repository.NewWorkOrderServiceRepository(pool)
	userRepo := repository.NewUserRepository(pool)

	// Services
	customerSvc := service.NewCustomerService(customerRepo)
	vehicleSvc := service.NewVehicleService(vehicleRepo)
	supplySvc := service.NewSupplyService(supplyRepo, wosRepo)
	wsSvc := service.NewWorkshopServiceManager(wsRepo)
	notificationSender := email.NewWorkOrderNotificationSender(mockEmail)
	budgetSvc := service.NewBudgetService(woRepo, wosRepo, customerRepo, notificationSender, "http://localhost:8080")
	statusSvc := service.NewWorkOrderStatusServiceWithNotifications(woRepo, wosRepo, customerRepo, mockEmail, "http://localhost:8080")
	woSvc := service.NewWorkOrderService(woRepo, vehicleRepo)
	creationSvc := service.NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc, service.WithBudgetRefresh(budgetSvc))
	itemSvc := service.NewWorkOrderItemService(wosRepo, woRepo, statusSvc)
	userSvc := service.NewUserService(userRepo)
	publicSvc := service.NewPublicWorkOrderService(woRepo, customerRepo, wosRepo)

	// Handlers
	customerHandler := NewCustomerHandler(customerSvc)
	vehicleHandler := NewVehicleHandler(vehicleSvc)
	supplyHandler := NewSupplyHandler(supplySvc)
	wsHandler := NewWorkshopServiceHandler(wsSvc)
	woHandler := NewWorkOrderHandler(woSvc, creationSvc, statusSvc, userSvc)
	wosHandler := NewWorkOrderServiceHandler(itemSvc)
	publicWoHandler := NewPublicWorkOrderHandler(publicSvc)

	app := fiber.New()

	fakeAuth := func(c fiber.Ctx) error {
		c.Locals("token", &auth.AppClaims{User: testUsername, Role: "employee"})
		return c.Next()
	}

	customers := app.Group("/customers", fakeAuth)
	customers.Post("/", customerHandler.Create)
	customers.Get("/", customerHandler.GetAll)
	customers.Get("/:id", customerHandler.GetByID)

	vehicles := app.Group("/vehicles", fakeAuth)
	vehicles.Post("/", vehicleHandler.Create)
	vehicles.Get("/", vehicleHandler.GetAll)

	supplies := app.Group("/supplies", fakeAuth)
	supplies.Post("/", supplyHandler.Create)

	services := app.Group("/services", fakeAuth)
	services.Post("/", wsHandler.Create)

	workOrders := app.Group("/work-orders", fakeAuth)
	workOrders.Post("/", woHandler.Create)
	workOrders.Get("/", woHandler.GetAll)
	workOrders.Get("/:id", woHandler.GetByID)
	workOrders.Put("/:id", woHandler.Update)
	workOrders.Post("/:id/services", woHandler.AddServices)
	workOrders.Put("/:id/services/:wosId/start", woHandler.StartService)
	workOrders.Put("/:id/services/:wosId/finalize", woHandler.FinalizeService)
	workOrders.Post("/:id/services/:wosId/supplies", woHandler.AddSupplies)

	approval := app.Group("/public/approvals")
	approval.Get("/services/:workOrderServiceId/approve", wosHandler.Approve)
	approval.Get("/services/:workOrderServiceId/reject", wosHandler.Reject)
	approval.Get("/work-orders/:workOrderId/approve-all", wosHandler.ApproveAll)
	approval.Get("/work-orders/:workOrderId/reject-all", wosHandler.RejectAll)

	publicWO := app.Group("/public/work-orders")
	publicWO.Get("/:code", publicWoHandler.GetByCode)

	return app, pool, mockEmail
}

func connectTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	host := envOrDefault("DATABASE_HOST", "localhost")
	port := envOrDefault("DATABASE_PORT", "5432")
	user := envOrDefault("DATABASE_USER", "techchallenge")
	password := os.Getenv("DATABASE_PASSWORD")
	if password == "" {
		t.Skip("skipping integration test: DATABASE_PASSWORD not set")
	}
	dbName := envOrDefault("DATABASE_NAME", "techchallenge-db")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, password, host, port, dbName)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("skipping integration test: cannot connect to database: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Skipf("skipping integration test: cannot ping database: %v", err)
	}

	return pool
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func setupFlowSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	// Drop all tables for a clean slate (order respects FK dependencies)
	tables := []string{
		"work_order_service_status_history",
		"work_order_service_supplies",
		"work_order_services",
		"work_orders",
		"supplies",
		"services",
		"vehicles",
		"customers",
		"users",
		"goose_db_version",
	}
	for _, table := range tables {
		pool.Exec(ctx, fmt.Sprintf(`DROP TABLE IF EXISTS %q CASCADE`, table))
	}

	// Apply all migrations via goose
	goose.SetBaseFS(dbmigrations.Migrations)
	require.NoError(t, goose.SetDialect("postgres"))
	db := stdlib.OpenDBFromPool(pool)
	require.NoError(t, goose.Up(db, "migrations"))

	t.Cleanup(func() {
		for _, table := range tables {
			pool.Exec(ctx, fmt.Sprintf(`DROP TABLE IF EXISTS %q CASCADE`, table))
		}
		db.Close()
		pool.Close()
	})
}

// seedTestUser insere direto na tabela o usuario que o middleware falso
// referencia. O cadastro de credencial vive na Lambda do
// oficina-mecanica-serverless, entao o monolito nao tem como criar o usuario
// por HTTP -- e nem precisa: ele so consome `users` para resolver quem abriu a
// OS. O hash abaixo e um argon2id valido e fixo, nunca verificado aqui.
const testUserPasswordHash = "$argon2id$v=19$m=65536,t=1,p=4$" +
	"c29tZS1maXhlZC1zYWx0MDA$" +
	"ZmFrZS1kaWdlc3QtZm9yLXRlc3RzLW9ubHktMzJieXRlcw"

func seedTestUser(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()

	var id uuid.UUID
	err := pool.QueryRow(context.Background(), `
		INSERT INTO "users" ("id", "username", "password_hash", "role", "created_at", "updated_at")
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT ("username") DO UPDATE SET "updated_at" = NOW()
		RETURNING "id"`,
		uuid.New(), testUsername, testUserPasswordHash, string(domain.UserRoleEmployee),
	).Scan(&id)
	require.NoError(t, err)

	return id.String()
}

// --- HTTP helpers ---

func flowPostJSON(app *fiber.App, path string, body any) (*http.Response, error) {
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	return app.Test(req)
}

func flowPutJSON(app *fiber.App, path string, body any) (*http.Response, error) {
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("PUT", path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	return app.Test(req)
}

func flowGet(app *fiber.App, path string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", path, nil)
	return app.Test(req)
}

func flowDelete(app *fiber.App, path string) (*http.Response, error) {
	req, _ := http.NewRequest("DELETE", path, nil)
	return app.Test(req)
}

func flowReadBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()
	var result map[string]any
	require.NoError(t, json.Unmarshal(data, &result))
	return result
}

func flowReadBodyArray(t *testing.T, resp *http.Response) []map[string]any {
	t.Helper()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()
	var result []map[string]any
	require.NoError(t, json.Unmarshal(data, &result))
	return result
}

// =============================================================================
// Test: Full Happy Path — Complete workshop flow from customer creation to
// vehicle delivery, validating every status transition matches the DDD flow.
// =============================================================================

func TestIntegration_Flow_HappyPath(t *testing.T) {
	app, _ := setupFlowApp(t)

	// ── Step 1: Atendente consulta cliente (não encontrado) ──
	var customerID string
	t.Run("Step1_ConsultaCliente_NaoEncontrado", func(t *testing.T) {
		resp, err := flowGet(app, "/customers?document=12345678909")
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		body := flowReadBody(t, resp)
		data := body["data"].([]any)
		assert.Len(t, data, 0, "customer should not exist yet")
	})

	// ── Step 2: Atendente cadastra cliente ──
	t.Run("Step2_CadastraCliente", func(t *testing.T) {
		resp, err := flowPostJSON(app, "/customers", map[string]any{
			"name":          "João da Silva",
			"email":         "joao@example.com",
			"document":      "12345678909",
			"document_type": "CPF",
		})
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		body := flowReadBody(t, resp)
		customerID = body["id"].(string)
		assert.NotEmpty(t, customerID)
		assert.Equal(t, "João da Silva", body["name"])
		assert.Equal(t, "12345678909", body["document"])
	})

	// ── Step 3: Atendente consulta veículo (não encontrado) ──
	var vehicleID string
	t.Run("Step3_ConsultaVeiculo_NaoEncontrado", func(t *testing.T) {
		resp, err := flowGet(app, "/vehicles?customerId="+customerID)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		body := flowReadBody(t, resp)
		data := body["data"].([]any)
		assert.Len(t, data, 0, "vehicle should not exist yet")
	})

	// ── Step 4: Atendente cadastra veículo ──
	t.Run("Step4_CadastraVeiculo", func(t *testing.T) {
		resp, err := flowPostJSON(app, "/vehicles", map[string]any{
			"license_plate": "ABC1D23",
			"customer_id":   customerID,
			"brand":         "Fiat",
			"model":         "Uno",
			"year":          2020,
		})
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		body := flowReadBody(t, resp)
		vehicleID = body["id"].(string)
		assert.NotEmpty(t, vehicleID)
		assert.Equal(t, "ABC1D23", body["license_plate"])
	})

	// ── Step 5: Atendente cria Ordem de Serviço (status: RECEBIDA) ──
	var workOrderID, workOrderCode string
	t.Run("Step5_CriaOrdemDeServico", func(t *testing.T) {
		resp, err := flowPostJSON(app, "/work-orders", map[string]any{
			"title":       "Revisão completa",
			"description": "Revisão dos 30.000 km",
			"customer_id": customerID,
			"vehicle_id":  vehicleID,
		})
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		body := flowReadBody(t, resp)
		workOrderID = body["id"].(string)
		workOrderCode = body["code"].(string)
		assert.NotEmpty(t, workOrderID)
		assert.NotEmpty(t, workOrderCode)
		assert.Equal(t, "RECEBIDA", body["status"])
	})

	// ── Step 6: Mecânico lista OS disponíveis (status RECEBIDA) ──
	t.Run("Step6_ListagemOSDisponiveisParaMecanico", func(t *testing.T) {
		resp, err := flowGet(app, "/work-orders?status=RECEBIDA")
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		body := flowReadBody(t, resp)
		data := body["data"].([]any)
		assert.GreaterOrEqual(t, len(data), 1, "should list at least the created work order")

		found := false
		for _, item := range data {
			wo := item.(map[string]any)
			if wo["id"] == workOrderID {
				found = true
				assert.Equal(t, "RECEBIDA", wo["status"])
			}
		}
		assert.True(t, found, "created work order should appear in the listing")
	})

	// ── Step 7: Mecânico pega a OS → status EM_DIAGNOSTICO ──
	t.Run("Step7_MecanicoPegaOS_EmDiagnostico", func(t *testing.T) {
		resp, err := flowPutJSON(app, "/work-orders/"+workOrderID, map[string]any{
			"status": "EM_DIAGNOSTICO",
		})
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		body := flowReadBody(t, resp)
		assert.Equal(t, "EM_DIAGNOSTICO", body["status"])
	})

	// ── Step 8: Cadastrar serviços de oficina (catálogo) ──
	var serviceOilChangeID, serviceAlignmentID string
	t.Run("Step8_CadastrarServicosOficina", func(t *testing.T) {
		// Service 1: Troca de óleo
		resp, err := flowPostJSON(app, "/services", map[string]any{
			"title":                  "Troca de Óleo",
			"description":            "Troca de óleo do motor completa",
			"price_cents":            15000,
			"estimated_time_minutes": 30,
		})
		require.NoError(t, err)
		require.Equal(t, fiber.StatusCreated, resp.StatusCode)
		body := flowReadBody(t, resp)
		serviceOilChangeID = body["id"].(string)

		// Service 2: Alinhamento
		resp, err = flowPostJSON(app, "/services", map[string]any{
			"title":                  "Alinhamento e Balanceamento",
			"description":            "Alinhamento e balanceamento das 4 rodas",
			"price_cents":            12000,
			"estimated_time_minutes": 60,
		})
		require.NoError(t, err)
		require.Equal(t, fiber.StatusCreated, resp.StatusCode)
		body = flowReadBody(t, resp)
		serviceAlignmentID = body["id"].(string)
	})

	// ── Step 9: Mecânico adiciona serviços à OS ──
	var wosOilChangeID, wosAlignmentID string
	t.Run("Step9_AdicionaServicosNaOS", func(t *testing.T) {
		resp, err := flowPostJSON(app, fmt.Sprintf("/work-orders/%s/services", workOrderID), []map[string]any{
			{"service_id": serviceOilChangeID},
			{"service_id": serviceAlignmentID},
		})
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		items := flowReadBodyArray(t, resp)
		require.Len(t, items, 2)

		for _, item := range items {
			assert.Equal(t, "PENDENTE", item["approval_status"])
			assert.Equal(t, "PENDENTE", item["status"])
			if item["service_id"] == serviceOilChangeID {
				wosOilChangeID = item["id"].(string)
				assert.Equal(t, "Troca de Óleo", item["service_title_snapshot"])
				assert.Equal(t, float64(15000), item["service_price_cents_snapshot"])
			}
			if item["service_id"] == serviceAlignmentID {
				wosAlignmentID = item["id"].(string)
				assert.Equal(t, "Alinhamento e Balanceamento", item["service_title_snapshot"])
			}
		}
		assert.NotEmpty(t, wosOilChangeID)
		assert.NotEmpty(t, wosAlignmentID)
	})

	// ── Step 10: Cadastrar insumos (peças) ──
	var supplyOilFilterID, supplyOilID string
	t.Run("Step10_CadastrarInsumos", func(t *testing.T) {
		resp, err := flowPostJSON(app, "/supplies", map[string]any{
			"title":          "Filtro de Óleo",
			"type":           "PECA",
			"price_cents":    3500,
			"stock_quantity": 10,
			"minimum_stock":  2,
		})
		require.NoError(t, err)
		require.Equal(t, fiber.StatusCreated, resp.StatusCode)
		body := flowReadBody(t, resp)
		supplyOilFilterID = body["id"].(string)

		resp, err = flowPostJSON(app, "/supplies", map[string]any{
			"title":          "Óleo Motor 5W30 1L",
			"type":           "INSUMO",
			"price_cents":    4500,
			"stock_quantity": 20,
			"minimum_stock":  5,
		})
		require.NoError(t, err)
		require.Equal(t, fiber.StatusCreated, resp.StatusCode)
		body = flowReadBody(t, resp)
		supplyOilID = body["id"].(string)
	})

	// ── Step 11: Mecânico adiciona peças/insumos aos serviços da OS ──
	t.Run("Step11_AdicionaPecasInsumosAosServicos", func(t *testing.T) {
		// Add supplies to the oil change service
		resp, err := flowPostJSON(app,
			fmt.Sprintf("/work-orders/%s/services/%s/supplies", workOrderID, wosOilChangeID),
			[]map[string]any{
				{"supply_id": supplyOilFilterID, "quantity": 1},
				{"supply_id": supplyOilID, "quantity": 4},
			},
		)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		items := flowReadBodyArray(t, resp)
		require.Len(t, items, 2)
		for _, item := range items {
			assert.NotEmpty(t, item["id"])
			assert.Equal(t, wosOilChangeID, item["work_order_service_id"])
		}
	})

	// ── Step 12: Transição para AGUARDANDO_APROVACAO (gerar orçamento) ──
	t.Run("Step12_TransicaoAguardandoAprovacao", func(t *testing.T) {
		resp, err := flowPutJSON(app, "/work-orders/"+workOrderID, map[string]any{
			"status": "AGUARDANDO_APROVACAO",
		})
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		body := flowReadBody(t, resp)
		assert.Equal(t, "AGUARDANDO_APROVACAO", body["status"])
	})

	// ── Step 13: Cliente consulta status da OS (rota pública) ──
	t.Run("Step13_ClienteConsultaStatusPublico", func(t *testing.T) {
		resp, err := flowGet(app, fmt.Sprintf("/public/work-orders/%s?document=12345678909", workOrderCode))
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		body := flowReadBody(t, resp)
		assert.Equal(t, workOrderCode, body["code"])
		assert.Equal(t, "AGUARDANDO_APROVACAO", body["status"])

		services := body["services"].([]any)
		assert.Len(t, services, 2, "should show both services in the public view")
	})

	// ── Step 14: Cliente aprova todos os serviços ──
	t.Run("Step14_ClienteAprovaTodosServicos", func(t *testing.T) {
		resp, err := flowGet(app, fmt.Sprintf("/public/approvals/work-orders/%s/approve-all", workOrderID))
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		body := flowReadBody(t, resp)
		assert.Contains(t, body["message"], "aprovados")

		// Verify the work order auto-transitioned to APROVADO
		resp, err = flowGet(app, "/work-orders/"+workOrderID)
		require.NoError(t, err)
		woBody := flowReadBody(t, resp)
		assert.Equal(t, "APROVADO", woBody["status"])
		assert.NotNil(t, woBody["approved_at"], "approved_at should be set")
	})

	// ── Step 15: Transição para EM_EXECUCAO ──
	t.Run("Step15_TransicaoEmExecucao", func(t *testing.T) {
		resp, err := flowPutJSON(app, "/work-orders/"+workOrderID, map[string]any{
			"status": "EM_EXECUCAO",
		})
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		body := flowReadBody(t, resp)
		assert.Equal(t, "EM_EXECUCAO", body["status"])
		assert.NotNil(t, body["started_at"], "started_at should be set")
	})

	// ── Step 16: Serviços continuam PENDENTE após EM_EXECUCAO ──
	t.Run("Step16_ServicosContinuamPendentes", func(t *testing.T) {
		resp, err := flowGet(app, "/work-orders/"+workOrderID)
		require.NoError(t, err)
		body := flowReadBody(t, resp)
		services := body["services"].([]any)
		for _, svc := range services {
			s := svc.(map[string]any)
			assert.Equal(t, "APROVADO", s["approval_status"])
			assert.Equal(t, "PENDENTE", s["status"],
				"services should remain PENDENTE after WO transitions to EM_EXECUCAO")
		}
	})

	// ── Step 17: Iniciar e finalizar cada serviço individualmente ──
	t.Run("Step17_IniciarServicosIndividualmente", func(t *testing.T) {
		// Start first service
		resp, err := flowPutJSON(app, fmt.Sprintf("/work-orders/%s/services/%s/start", workOrderID, wosOilChangeID), map[string]any{})
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Start second service
		resp, err = flowPutJSON(app, fmt.Sprintf("/work-orders/%s/services/%s/start", workOrderID, wosAlignmentID), map[string]any{})
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Step18_FinalizarPrimeiroServico", func(t *testing.T) {
		resp, err := flowPutJSON(app, fmt.Sprintf("/work-orders/%s/services/%s/finalize", workOrderID, wosOilChangeID), map[string]any{})
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// WO should still be EM_EXECUCAO (not all services finalized)
		resp, err = flowGet(app, "/work-orders/"+workOrderID)
		require.NoError(t, err)
		body := flowReadBody(t, resp)
		assert.Equal(t, "EM_EXECUCAO", body["status"],
			"WO should stay EM_EXECUCAO while there are unfinished services")
	})

	t.Run("Step19_FinalizarUltimoServico_AutoFinalizaOS", func(t *testing.T) {
		resp, err := flowPutJSON(app, fmt.Sprintf("/work-orders/%s/services/%s/finalize", workOrderID, wosAlignmentID), map[string]any{})
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// WO should auto-transition to FINALIZADA
		resp, err = flowGet(app, "/work-orders/"+workOrderID)
		require.NoError(t, err)
		body := flowReadBody(t, resp)
		assert.Equal(t, "FINALIZADA", body["status"],
			"WO should auto-transition to FINALIZADA when all services are finalized")
		assert.NotNil(t, body["finished_at"])
	})

	// ── Step 20: Transição para ENTREGUE (cliente retira veículo) ──
	t.Run("Step20_TransicaoEntregue_ClienteRetiraVeiculo", func(t *testing.T) {
		resp, err := flowPutJSON(app, "/work-orders/"+workOrderID, map[string]any{
			"status": "ENTREGUE",
		})
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		body := flowReadBody(t, resp)
		assert.Equal(t, "ENTREGUE", body["status"])
		assert.NotNil(t, body["delivered_at"], "delivered_at should be set")
	})
}

// =============================================================================
// Test: Partial Approval — Some services approved, some rejected.
// The work order should transition to APROVADO with only the approved total.
// =============================================================================

func TestIntegration_Flow_PartialApproval(t *testing.T) {
	app, _ := setupFlowApp(t)

	// Setup: create customer, vehicle, work order, services
	customerID := createTestCustomer(t, app, "Maria Souza", "maria@example.com", "98765432100", "CPF")
	vehicleID := createTestVehicle(t, app, customerID, "XYZ9A87", "Honda", "Civic", 2022)
	workOrderID := createTestWorkOrder(t, app, "Reparo freios", customerID, vehicleID)

	svc1ID := createTestWorkshopService(t, app, "Troca de Pastilha", 8000, 45)
	svc2ID := createTestWorkshopService(t, app, "Troca de Disco", 25000, 90)

	wos := addServicesToWorkOrder(t, app, workOrderID, []string{svc1ID, svc2ID})
	wos1ID := wos[0]
	wos2ID := wos[1]

	// Transition to AGUARDANDO_APROVACAO
	transitionWorkOrder(t, app, workOrderID, "AGUARDANDO_APROVACAO")

	// Client approves service 1, rejects service 2
	t.Run("AprovaServico1", func(t *testing.T) {
		resp, err := flowGet(app, fmt.Sprintf("/public/approvals/services/%s/approve", wos1ID))
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("ReprovaServico2", func(t *testing.T) {
		resp, err := flowGet(app, fmt.Sprintf("/public/approvals/services/%s/reject", wos2ID))
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	// Work order should be APROVADO (has at least one approved service)
	t.Run("VerificaStatusAprovado", func(t *testing.T) {
		resp, err := flowGet(app, "/work-orders/"+workOrderID)
		require.NoError(t, err)
		body := flowReadBody(t, resp)
		assert.Equal(t, "APROVADO", body["status"])
		// Total should only include the approved service (8000 cents)
		assert.Equal(t, float64(8000), body["total_estimated_price_cents"])
	})

	// Can continue flow: APROVADO → EM_EXECUCAO → start/finalize services → auto-FINALIZADA → ENTREGUE
	t.Run("FluxoContinuaAteEntrega", func(t *testing.T) {
		transitionWorkOrder(t, app, workOrderID, "EM_EXECUCAO")

		// Start and finalize the approved service individually
		startAndFinalizeService(t, app, workOrderID, wos1ID)

		// WO should auto-transition to FINALIZADA
		resp, err := flowGet(app, "/work-orders/"+workOrderID)
		require.NoError(t, err)
		body := flowReadBody(t, resp)
		assert.Equal(t, "FINALIZADA", body["status"])

		transitionWorkOrder(t, app, workOrderID, "ENTREGUE")

		resp, err = flowGet(app, "/work-orders/"+workOrderID)
		require.NoError(t, err)
		body = flowReadBody(t, resp)
		assert.Equal(t, "ENTREGUE", body["status"])
		assert.NotNil(t, body["delivered_at"])
	})
}

// =============================================================================
// Test: Full Rejection — All services rejected → OS canceled.
// Mapeia o passo do fluxo DDD: "Nenhum serviço foi aprovado e o cliente é
// notificado sobre a retirada do veículo"
// =============================================================================

func TestIntegration_Flow_FullRejection(t *testing.T) {
	app, _ := setupFlowApp(t)

	customerID := createTestCustomer(t, app, "Pedro Alves", "pedro@example.com", "52998224725", "CPF")
	vehicleID := createTestVehicle(t, app, customerID, "DEF4G56", "Toyota", "Corolla", 2021)
	workOrderID := createTestWorkOrder(t, app, "Revisão elétrica", customerID, vehicleID)

	svcID := createTestWorkshopService(t, app, "Revisão Elétrica", 20000, 120)
	addServicesToWorkOrder(t, app, workOrderID, []string{svcID})

	transitionWorkOrder(t, app, workOrderID, "AGUARDANDO_APROVACAO")

	// Client rejects all
	t.Run("ClienteReprovaServicos", func(t *testing.T) {
		resp, err := flowGet(app, fmt.Sprintf("/public/approvals/work-orders/%s/reject-all", workOrderID))
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	// Work order should be CANCELADA
	t.Run("VerificaStatusCancelada", func(t *testing.T) {
		resp, err := flowGet(app, "/work-orders/"+workOrderID)
		require.NoError(t, err)
		body := flowReadBody(t, resp)
		assert.Equal(t, "CANCELADA", body["status"])
	})

	// Cannot transition from CANCELADA
	t.Run("NaoPodeTransicionarDeCancelada", func(t *testing.T) {
		resp, err := flowPutJSON(app, "/work-orders/"+workOrderID, map[string]any{
			"status": "EM_EXECUCAO",
		})
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)
	})
}

// =============================================================================
// Test: Invalid Status Transitions — Verifies the state machine rejects
// transitions that violate the DDD flow sequence.
// =============================================================================

func TestIntegration_Flow_InvalidStatusTransitions(t *testing.T) {
	app, _ := setupFlowApp(t)

	customerID := createTestCustomer(t, app, "Ana Costa", "ana@example.com", "44011076910", "CPF")
	vehicleID := createTestVehicle(t, app, customerID, "GHI5J67", "VW", "Golf", 2023)
	workOrderID := createTestWorkOrder(t, app, "Teste transições", customerID, vehicleID)

	tests := []struct {
		name       string
		fromStatus string
		toStatus   string
	}{
		{"RECEBIDA para APROVADO", "", "APROVADO"},
		{"RECEBIDA para EM_EXECUCAO", "", "EM_EXECUCAO"},
		{"RECEBIDA para FINALIZADA", "", "FINALIZADA"},
		{"RECEBIDA para ENTREGUE", "", "ENTREGUE"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := flowPutJSON(app, "/work-orders/"+workOrderID, map[string]any{
				"status": tc.toStatus,
			})
			require.NoError(t, err)
			assert.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode,
				"should reject transition from RECEBIDA to %s", tc.toStatus)
		})
	}

	// Transition to EM_DIAGNOSTICO (valid), then test invalid from there
	transitionWorkOrder(t, app, workOrderID, "EM_DIAGNOSTICO")

	invalidFromDiag := []string{"APROVADO", "EM_EXECUCAO", "FINALIZADA", "ENTREGUE"}
	for _, status := range invalidFromDiag {
		t.Run("EM_DIAGNOSTICO_para_"+status, func(t *testing.T) {
			resp, err := flowPutJSON(app, "/work-orders/"+workOrderID, map[string]any{
				"status": status,
			})
			require.NoError(t, err)
			assert.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)
		})
	}
}

// =============================================================================
// Test: Adding services auto-transitions RECEBIDA → EM_DIAGNOSTICO
// This is the alternative path where the mechanic adds services directly
// without explicitly picking up the work order first.
// =============================================================================

func TestIntegration_Flow_AddServicesAutoTransition(t *testing.T) {
	app, _ := setupFlowApp(t)

	customerID := createTestCustomer(t, app, "Carlos Mendes", "carlos@example.com", "01498117481", "CPF")
	vehicleID := createTestVehicle(t, app, customerID, "JKL6M89", "Chevrolet", "Onix", 2024)
	workOrderID := createTestWorkOrder(t, app, "Auto transition test", customerID, vehicleID)

	// Verify initial status
	resp, err := flowGet(app, "/work-orders/"+workOrderID)
	require.NoError(t, err)
	body := flowReadBody(t, resp)
	assert.Equal(t, "RECEBIDA", body["status"])

	// Add services directly (without explicit transition to EM_DIAGNOSTICO)
	svcID := createTestWorkshopService(t, app, "Troca de Pneu", 10000, 20)
	addServicesToWorkOrder(t, app, workOrderID, []string{svcID})

	// Should have auto-transitioned to EM_DIAGNOSTICO
	t.Run("AutoTransitionToEmDiagnostico", func(t *testing.T) {
		resp, err := flowGet(app, "/work-orders/"+workOrderID)
		require.NoError(t, err)
		body := flowReadBody(t, resp)
		assert.Equal(t, "EM_DIAGNOSTICO", body["status"],
			"adding services to RECEBIDA work order should auto-transition to EM_DIAGNOSTICO")
	})
}

// =============================================================================
// Test: Cannot add services when work order is in terminal/non-editable status.
// =============================================================================

func TestIntegration_Flow_CannotAddServicesInWrongStatus(t *testing.T) {
	app, _ := setupFlowApp(t)

	customerID := createTestCustomer(t, app, "Roberto Lima", "roberto@example.com", "65464216740", "CPF")
	vehicleID := createTestVehicle(t, app, customerID, "MNO7P12", "Ford", "Ka", 2019)
	workOrderID := createTestWorkOrder(t, app, "Status test", customerID, vehicleID)

	svcID := createTestWorkshopService(t, app, "Serviço de Teste", 5000, 15)
	addServicesToWorkOrder(t, app, workOrderID, []string{svcID})

	// Advance through approval
	transitionWorkOrder(t, app, workOrderID, "AGUARDANDO_APROVACAO")
	transitionWorkOrder(t, app, workOrderID, "APROVADO")

	// Try to add more services — should fail
	t.Run("NaoPodeAdicionarServicosAposAprovacao", func(t *testing.T) {
		svc2ID := createTestWorkshopService(t, app, "Outro Serviço", 3000, 10)
		resp, err := flowPostJSON(app,
			fmt.Sprintf("/work-orders/%s/services", workOrderID),
			[]map[string]any{{"service_id": svc2ID}},
		)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode,
			"should not allow adding services after approval")
	})
}

// =============================================================================
// Test: Public work order lookup validates customer document.
// =============================================================================

func TestIntegration_Flow_PublicLookupValidatesDocument(t *testing.T) {
	app, _ := setupFlowApp(t)

	customerID := createTestCustomer(t, app, "Fernanda Rocha", "fernanda@example.com", "01079054189", "CPF")
	vehicleID := createTestVehicle(t, app, customerID, "QRS8T34", "Renault", "Kwid", 2023)
	workOrderID := createTestWorkOrder(t, app, "Public lookup test", customerID, vehicleID)
	_ = workOrderID

	// Get the work order code
	resp, err := flowGet(app, "/work-orders/"+workOrderID)
	require.NoError(t, err)
	woBody := flowReadBody(t, resp)
	code := woBody["code"].(string)

	// Correct document
	t.Run("DocumentoCorreto", func(t *testing.T) {
		resp, err := flowGet(app, fmt.Sprintf("/public/work-orders/%s?document=01079054189", code))
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	// Wrong document
	t.Run("DocumentoErrado", func(t *testing.T) {
		resp, err := flowGet(app, fmt.Sprintf("/public/work-orders/%s?document=00000000000", code))
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	// Missing document
	t.Run("DocumentoAusente", func(t *testing.T) {
		resp, err := flowGet(app, fmt.Sprintf("/public/work-orders/%s", code))
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// =============================================================================
// Test: Remove service and supply from work order
// =============================================================================

func TestIntegration_Flow_RemoveServiceAndSupply(t *testing.T) {
	app, _ := setupFlowApp(t)

	customerID := createTestCustomer(t, app, "Lucia Santos", "lucia@example.com", "13471876421", "CPF")
	vehicleID := createTestVehicle(t, app, customerID, "UVW0X56", "Hyundai", "HB20", 2021)
	workOrderID := createTestWorkOrder(t, app, "Remove test", customerID, vehicleID)

	svcID := createTestWorkshopService(t, app, "Lavagem Completa", 5000, 60)
	wosIDs := addServicesToWorkOrder(t, app, workOrderID, []string{svcID})
	wosID := wosIDs[0]

	// Add a supply
	supplyID := createTestSupply(t, app, "Shampoo Automotivo", "INSUMO", 1500, 50, 5)
	resp, err := flowPostJSON(app,
		fmt.Sprintf("/work-orders/%s/services/%s/supplies", workOrderID, wosID),
		[]map[string]any{{"supply_id": supplyID, "quantity": 2}},
	)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)
	addedSupplies := flowReadBodyArray(t, resp)
	require.Len(t, addedSupplies, 1)
	wosSupplyID := addedSupplies[0]["id"].(string)

	// Remove the supply
	t.Run("RemoveSupply", func(t *testing.T) {
		resp, err := flowDelete(app, fmt.Sprintf("/work-orders/%s/services/%s/supplies/%s", workOrderID, wosID, wosSupplyID))
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
	})

	// Remove the service
	t.Run("RemoveService", func(t *testing.T) {
		resp, err := flowDelete(app, fmt.Sprintf("/work-orders/%s/services/%s", workOrderID, wosID))
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
	})
}

// =============================================================================
// Test: Idempotent approval — approving an already-approved service is a no-op.
// =============================================================================

func TestIntegration_Flow_IdempotentApproval(t *testing.T) {
	app, _ := setupFlowApp(t)

	customerID := createTestCustomer(t, app, "Marcos Vieira", "marcos@example.com", "09746114760", "CPF")
	vehicleID := createTestVehicle(t, app, customerID, "YZA1B23", "Nissan", "Kicks", 2022)
	workOrderID := createTestWorkOrder(t, app, "Idempotent test", customerID, vehicleID)

	svcID := createTestWorkshopService(t, app, "Polimento", 8000, 90)
	wosIDs := addServicesToWorkOrder(t, app, workOrderID, []string{svcID})

	transitionWorkOrder(t, app, workOrderID, "AGUARDANDO_APROVACAO")

	// Approve twice — should not error
	resp, err := flowGet(app, fmt.Sprintf("/public/approvals/services/%s/approve", wosIDs[0]))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	resp, err = flowGet(app, fmt.Sprintf("/public/approvals/services/%s/approve", wosIDs[0]))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode, "second approval should be idempotent")
}

// =============================================================================
// Test: Duplicate creations — entities with unique constraints must return 409.
// =============================================================================

func TestIntegration_Flow_DuplicateCustomerDocument(t *testing.T) {
	app, _ := setupFlowApp(t)

	createTestCustomer(t, app, "Primeiro Cliente", "primeiro@example.com", "12345678909", "CPF")

	t.Run("MesmoDocumentoRetorna409", func(t *testing.T) {
		resp, err := flowPostJSON(app, "/customers", map[string]any{
			"name":          "Outro Cliente",
			"email":         "outro@example.com",
			"document":      "12345678909",
			"document_type": "CPF",
		})
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
	})
}

func TestIntegration_Flow_DuplicateVehiclePlate(t *testing.T) {
	app, _ := setupFlowApp(t)

	customerID := createTestCustomer(t, app, "Cliente Veículo", "cv@example.com", "12345678909", "CPF")
	createTestVehicle(t, app, customerID, "ABC1D23", "Fiat", "Uno", 2020)

	t.Run("MesmaPlacaRetorna409", func(t *testing.T) {
		resp, err := flowPostJSON(app, "/vehicles", map[string]any{
			"license_plate": "ABC1D23",
			"customer_id":   customerID,
			"brand":         "VW",
			"model":         "Gol",
			"year":          2021,
		})
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
	})
}

func TestIntegration_Flow_DuplicateWorkshopServiceTitle(t *testing.T) {
	app, _ := setupFlowApp(t)

	createTestWorkshopService(t, app, "Troca de Óleo Único", 15000, 30)

	t.Run("MesmoTituloRetorna409", func(t *testing.T) {
		resp, err := flowPostJSON(app, "/services", map[string]any{
			"title":                  "Troca de Óleo Único",
			"price_cents":            20000,
			"estimated_time_minutes": 45,
		})
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
	})
}

func TestIntegration_Flow_DuplicateSupplyOnService(t *testing.T) {
	app, _ := setupFlowApp(t)

	customerID := createTestCustomer(t, app, "Dup Supply", "ds@example.com", "12345678909", "CPF")
	vehicleID := createTestVehicle(t, app, customerID, "DUP1A23", "Fiat", "Mobi", 2022)
	workOrderID := createTestWorkOrder(t, app, "Dup supply test", customerID, vehicleID)

	svcID := createTestWorkshopService(t, app, "Serviço Dup Supply", 5000, 30)
	wosIDs := addServicesToWorkOrder(t, app, workOrderID, []string{svcID})
	supplyID := createTestSupply(t, app, "Peça Única", "PECA", 1000, 10, 2)

	// First add — OK
	resp, err := flowPostJSON(app,
		fmt.Sprintf("/work-orders/%s/services/%s/supplies", workOrderID, wosIDs[0]),
		[]map[string]any{{"supply_id": supplyID, "quantity": 1}},
	)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)

	// Second add of same supply — should return 409
	t.Run("MesmoInsumoRetorna409", func(t *testing.T) {
		resp, err := flowPostJSON(app,
			fmt.Sprintf("/work-orders/%s/services/%s/supplies", workOrderID, wosIDs[0]),
			[]map[string]any{{"supply_id": supplyID, "quantity": 2}},
		)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
	})
}

// =============================================================================
// Test: Validation errors — invalid input data must return 400.
// =============================================================================

func TestIntegration_Flow_CustomerValidationErrors(t *testing.T) {
	app, _ := setupFlowApp(t)

	t.Run("NomeFaltando", func(t *testing.T) {
		resp, err := flowPostJSON(app, "/customers", map[string]any{
			"email":         "sem-nome@example.com",
			"document":      "12345678909",
			"document_type": "CPF",
		})
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("EmailInvalido", func(t *testing.T) {
		resp, err := flowPostJSON(app, "/customers", map[string]any{
			"name":          "Test",
			"email":         "nao-eh-email",
			"document":      "12345678909",
			"document_type": "CPF",
		})
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("CPFInvalido", func(t *testing.T) {
		resp, err := flowPostJSON(app, "/customers", map[string]any{
			"name":          "Test",
			"email":         "test@example.com",
			"document":      "00000000000",
			"document_type": "CPF",
		})
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("TipoDocumentoInvalido", func(t *testing.T) {
		resp, err := flowPostJSON(app, "/customers", map[string]any{
			"name":          "Test",
			"email":         "test@example.com",
			"document":      "12345678909",
			"document_type": "RG",
		})
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestIntegration_Flow_VehicleValidationErrors(t *testing.T) {
	app, _ := setupFlowApp(t)

	customerID := createTestCustomer(t, app, "Val Vehicle", "vv@example.com", "12345678909", "CPF")

	t.Run("PlacaInvalida", func(t *testing.T) {
		resp, err := flowPostJSON(app, "/vehicles", map[string]any{
			"license_plate": "INVALIDA",
			"customer_id":   customerID,
			"brand":         "Fiat",
			"model":         "Uno",
			"year":          2020,
		})
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("AnoInvalido", func(t *testing.T) {
		resp, err := flowPostJSON(app, "/vehicles", map[string]any{
			"license_plate": "XYZ1A23",
			"customer_id":   customerID,
			"brand":         "Fiat",
			"model":         "Uno",
			"year":          1800,
		})
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("SemPlaca", func(t *testing.T) {
		resp, err := flowPostJSON(app, "/vehicles", map[string]any{
			"customer_id": customerID,
			"brand":       "Fiat",
			"model":       "Uno",
			"year":        2020,
		})
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestIntegration_Flow_WorkshopServiceValidationErrors(t *testing.T) {
	app, _ := setupFlowApp(t)

	t.Run("SemTitulo", func(t *testing.T) {
		resp, err := flowPostJSON(app, "/services", map[string]any{
			"price_cents":            5000,
			"estimated_time_minutes": 30,
		})
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PrecoZero", func(t *testing.T) {
		resp, err := flowPostJSON(app, "/services", map[string]any{
			"title":                  "Serviço Grátis",
			"price_cents":            0,
			"estimated_time_minutes": 30,
		})
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("TempoZero", func(t *testing.T) {
		resp, err := flowPostJSON(app, "/services", map[string]any{
			"title":                  "Serviço Instantâneo",
			"price_cents":            5000,
			"estimated_time_minutes": 0,
		})
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// =============================================================================
// Test: Referential integrity — work order with non-existent or mismatched refs.
// =============================================================================

func TestIntegration_Flow_WorkOrderReferentialErrors(t *testing.T) {
	app, _ := setupFlowApp(t)

	customerID := createTestCustomer(t, app, "Ref Test", "ref@example.com", "12345678909", "CPF")
	vehicleID := createTestVehicle(t, app, customerID, "REF1A23", "Fiat", "Uno", 2020)

	t.Run("ClienteInexistente", func(t *testing.T) {
		resp, err := flowPostJSON(app, "/work-orders", map[string]any{
			"title":       "OS com cliente fake",
			"customer_id": "00000000-0000-0000-0000-000000000000",
			"vehicle_id":  vehicleID,
		})
		require.NoError(t, err)
		assert.NotEqual(t, fiber.StatusCreated, resp.StatusCode)
	})

	t.Run("VeiculoInexistente", func(t *testing.T) {
		resp, err := flowPostJSON(app, "/work-orders", map[string]any{
			"title":       "OS com veículo fake",
			"customer_id": customerID,
			"vehicle_id":  "00000000-0000-0000-0000-000000000000",
		})
		require.NoError(t, err)
		assert.NotEqual(t, fiber.StatusCreated, resp.StatusCode)
	})

	t.Run("VeiculoNaoPertenceAoCliente", func(t *testing.T) {
		otherCustomerID := createTestCustomer(t, app, "Outro Dono", "outro@example.com", "98765432100", "CPF")
		resp, err := flowPostJSON(app, "/work-orders", map[string]any{
			"title":       "OS com veículo de outro cliente",
			"customer_id": otherCustomerID,
			"vehicle_id":  vehicleID,
		})
		require.NoError(t, err)
		assert.NotEqual(t, fiber.StatusCreated, resp.StatusCode,
			"should not allow creating work order with vehicle belonging to another customer")
	})

	t.Run("SemTitulo", func(t *testing.T) {
		resp, err := flowPostJSON(app, "/work-orders", map[string]any{
			"customer_id": customerID,
			"vehicle_id":  vehicleID,
		})
		require.NoError(t, err)
		assert.NotEqual(t, fiber.StatusCreated, resp.StatusCode)
	})
}

// =============================================================================
// Test: Adding inactive service to work order must fail.
// =============================================================================

func TestIntegration_Flow_CannotAddInactiveService(t *testing.T) {
	app, _ := setupFlowApp(t)

	customerID := createTestCustomer(t, app, "Inactive Svc", "inactive@example.com", "52998224725", "CPF")
	vehicleID := createTestVehicle(t, app, customerID, "INA1B23", "Honda", "Fit", 2021)
	workOrderID := createTestWorkOrder(t, app, "Inactive service test", customerID, vehicleID)

	// Create service then deactivate it
	svcID := createTestWorkshopService(t, app, "Serviço Desativado", 7000, 40)
	resp, err := flowPutJSON(app, "/services/"+svcID, map[string]any{"active": false})
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	// Try to add inactive service to work order — should fail
	resp, err = flowPostJSON(app,
		fmt.Sprintf("/work-orders/%s/services", workOrderID),
		[]map[string]any{{"service_id": svcID}},
	)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode,
		"should not allow adding inactive service to work order")
}

// =============================================================================
// Test: Cannot start service when supplies exceed stock.
// Reproduces: add service with 1000 units of a supply that has stock=10,
// approve, transition to EM_EXECUCAO, then try to start → must fail 422.
// =============================================================================

func TestIntegration_Flow_CannotStartServiceWithoutStock(t *testing.T) {
	app, _ := setupFlowApp(t)

	customerID := createTestCustomer(t, app, "Stock Test", "stock@example.com", "12345678909", "CPF")
	vehicleID := createTestVehicle(t, app, customerID, "STK2B34", "Fiat", "Argo", 2023)
	workOrderID := createTestWorkOrder(t, app, "Stock block test", customerID, vehicleID)

	svcID := createTestWorkshopService(t, app, "Servico Sem Estoque", 5000, 30)
	wosIDs := addServicesToWorkOrder(t, app, workOrderID, []string{svcID})

	// Supply with stock=10, but we request quantity=1000
	supplyID := createTestSupply(t, app, "Peca Escassa", "PECA", 100, 10, 2)
	resp, err := flowPostJSON(app,
		fmt.Sprintf("/work-orders/%s/services/%s/supplies", workOrderID, wosIDs[0]),
		[]map[string]any{{"supply_id": supplyID, "quantity": 1000}},
	)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)

	// Advance: EM_DIAGNOSTICO → AGUARDANDO_APROVACAO → approve → APROVADO → EM_EXECUCAO
	transitionWorkOrder(t, app, workOrderID, "AGUARDANDO_APROVACAO")

	resp, err = flowGet(app, fmt.Sprintf("/public/approvals/work-orders/%s/approve-all", workOrderID))
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	transitionWorkOrder(t, app, workOrderID, "EM_EXECUCAO")

	// Try to start the service — must be blocked because stock=10 < quantity=1000
	t.Run("IniciarBloqueadoPorEstoque", func(t *testing.T) {
		resp, err := flowPutJSON(app,
			fmt.Sprintf("/work-orders/%s/services/%s/start", workOrderID, wosIDs[0]),
			map[string]any{},
		)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode,
			"should block starting service when supply stock is insufficient")

		body := flowReadBody(t, resp)
		assert.Contains(t, body["error"], "insufficient stock")
	})

	// After increasing stock, start should succeed
	t.Run("IniciarLiberadoAposReporEstoque", func(t *testing.T) {
		// Update supply stock to 1000
		resp, err := flowPutJSON(app, "/supplies/"+supplyID, map[string]any{
			"title":          "Peca Escassa",
			"type":           "PECA",
			"price_cents":    100,
			"stock_quantity": 1000,
			"minimum_stock":  2,
		})
		require.NoError(t, err)
		require.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Now start should work
		resp, err = flowPutJSON(app,
			fmt.Sprintf("/work-orders/%s/services/%s/start", workOrderID, wosIDs[0]),
			map[string]any{},
		)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode,
			"should allow starting service after stock is replenished")
	})
}

// =============================================================================
// Test: Not found errors — GET/PUT/DELETE on non-existent resources return 404.
// =============================================================================

func TestIntegration_Flow_NotFoundErrors(t *testing.T) {
	app, _ := setupFlowApp(t)

	fakeID := "00000000-0000-0000-0000-000000000001"

	t.Run("CustomerNotFound", func(t *testing.T) {
		resp, err := flowGet(app, "/customers/"+fakeID)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("VehicleNotFound", func(t *testing.T) {
		resp, err := flowGet(app, "/vehicles/"+fakeID)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("WorkOrderNotFound", func(t *testing.T) {
		resp, err := flowGet(app, "/work-orders/"+fakeID)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("WorkshopServiceNotFound", func(t *testing.T) {
		resp, err := flowGet(app, "/services/"+fakeID)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("SupplyNotFound", func(t *testing.T) {
		resp, err := flowGet(app, "/supplies/"+fakeID)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("PublicWorkOrderNotFound", func(t *testing.T) {
		resp, err := flowGet(app, "/public/work-orders/OS-FAKE-CODE?document=12345678909")
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("ApproveNonExistentService", func(t *testing.T) {
		resp, err := flowGet(app, "/public/approvals/services/"+fakeID+"/approve")
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

// =============================================================================
// Test: Ownership validation — service/supply must belong to the work order.
// =============================================================================

func TestIntegration_Flow_OwnershipValidation(t *testing.T) {
	app, _ := setupFlowApp(t)

	customerID := createTestCustomer(t, app, "Owner Test", "owner@example.com", "12345678909", "CPF")
	vehicleID := createTestVehicle(t, app, customerID, "OWN1A23", "Fiat", "Toro", 2023)

	wo1ID := createTestWorkOrder(t, app, "WO 1", customerID, vehicleID)
	wo2ID := createTestWorkOrder(t, app, "WO 2", customerID, vehicleID)

	svcID := createTestWorkshopService(t, app, "Serviço Ownership", 5000, 30)
	wos1IDs := addServicesToWorkOrder(t, app, wo1ID, []string{svcID})

	t.Run("RemoverServicoDeOutraOS", func(t *testing.T) {
		// Try to remove wo1's service via wo2's route
		resp, err := flowDelete(app, fmt.Sprintf("/work-orders/%s/services/%s", wo2ID, wos1IDs[0]))
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode,
			"should not allow removing service that belongs to another work order")
	})
}

// =============================================================================
// Test: Budget generation — transitioning to AGUARDANDO_APROVACAO triggers
// budget calculation, sets quote_sent_at, calculates total, and sends email.
// Cobre os passos do fluxo DDD: "Gera orçamento com tempo estimado",
// "Envia notificação para o cliente", "Atualiza OS para aguardando aprovação".
// =============================================================================

func TestIntegration_Flow_BudgetGenerationAndNotification(t *testing.T) {
	app, _, mockEmail := setupFlowAppWithBudget(t)

	customerID := createTestCustomer(t, app, "Cliente Orçamento", "orcamento@example.com", "12345678909", "CPF")
	vehicleID := createTestVehicle(t, app, customerID, "ORC1A23", "Fiat", "Strada", 2022)
	workOrderID := createTestWorkOrder(t, app, "Orçamento test", customerID, vehicleID)

	// Add two services: 15000 + 12000 = 27000 cents
	svc1ID := createTestWorkshopService(t, app, "Serviço Orçamento A", 15000, 30)
	svc2ID := createTestWorkshopService(t, app, "Serviço Orçamento B", 12000, 60)
	wosIDs := addServicesToWorkOrder(t, app, workOrderID, []string{svc1ID, svc2ID})

	// Add supplies to service 1: 3500*1 + 4500*4 = 21500 cents
	supply1ID := createTestSupply(t, app, "Filtro Orçamento", "PECA", 3500, 10, 2)
	supply2ID := createTestSupply(t, app, "Óleo Orçamento", "INSUMO", 4500, 20, 5)
	resp, err := flowPostJSON(app,
		fmt.Sprintf("/work-orders/%s/services/%s/supplies", workOrderID, wosIDs[0]),
		[]map[string]any{
			{"supply_id": supply1ID, "quantity": 1},
			{"supply_id": supply2ID, "quantity": 4},
		},
	)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)

	// Transition to AGUARDANDO_APROVACAO — triggers budget generation
	t.Run("TransicaoGeraOrcamento", func(t *testing.T) {
		resp, err := flowPutJSON(app, "/work-orders/"+workOrderID, map[string]any{
			"status": "AGUARDANDO_APROVACAO",
		})
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	// Verify quote_sent_at was set
	t.Run("QuoteSentAtDefinido", func(t *testing.T) {
		resp, err := flowGet(app, "/work-orders/"+workOrderID)
		require.NoError(t, err)
		body := flowReadBody(t, resp)
		assert.Equal(t, "AGUARDANDO_APROVACAO", body["status"])
		assert.NotNil(t, body["quote_sent_at"], "quote_sent_at should be set after budget generation")
	})

	// Verify total was calculated: services(15000+12000) + supplies(3500+18000) = 48500
	t.Run("TotalCalculadoCorretamente", func(t *testing.T) {
		resp, err := flowGet(app, "/work-orders/"+workOrderID)
		require.NoError(t, err)
		body := flowReadBody(t, resp)
		assert.Equal(t, float64(48500), body["total_estimated_price_cents"],
			"total should be sum of services + supplies: 15000+12000+3500+(4500*4)=48500")
	})

	// Verify email was sent to customer
	t.Run("EmailEnviadoParaCliente", func(t *testing.T) {
		msgs := mockEmail.Messages()
		require.GreaterOrEqual(t, len(msgs), 2, "should send diagnosis and budget emails")

		msg, ok := findLatestEmailBySubjectContains(msgs, "Orçamento")
		require.True(t, ok)
		assert.Contains(t, msg.To, "orcamento@example.com", "email should be sent to customer")
		assert.Contains(t, msg.Subject, "Orçamento", "subject should mention orçamento")
		assert.True(t, msg.HTML, "email should be HTML")
	})

	// Verify email body contains approval links
	t.Run("EmailContemLinksDeAprovacao", func(t *testing.T) {
		msgs := mockEmail.Messages()
		msg, ok := findLatestEmailBySubjectContains(msgs, "Orçamento")
		require.True(t, ok)

		body := msg.Body
		assert.Contains(t, body, "Cliente Orçamento", "email should contain customer name")
		assert.Contains(t, body, "approve-all", "email should contain approve-all link")
		assert.Contains(t, body, "reject-all", "email should contain reject-all link")

		// Should contain individual service approve/reject links
		for _, wosID := range wosIDs {
			assert.Contains(t, body, fmt.Sprintf("/services/%s/approve", wosID),
				"email should contain individual approve link for each service")
			assert.Contains(t, body, fmt.Sprintf("/services/%s/reject", wosID),
				"email should contain individual reject link for each service")
		}
	})
}

func TestIntegration_Flow_AdjustmentWhileWaitingApprovalResendsBudget(t *testing.T) {
	app, _, mockEmail := setupFlowAppWithBudget(t)

	customerID := createTestCustomer(t, app, "Cliente Ajuste", "ajuste@example.com", "12345678909", "CPF")
	vehicleID := createTestVehicle(t, app, customerID, "AJU1A23", "Fiat", "Toro", 2024)
	workOrderID := createTestWorkOrder(t, app, "Ajuste orçamento", customerID, vehicleID)

	svc1ID := createTestWorkshopService(t, app, "Serviço Ajuste A", 10000, 30)
	addServicesToWorkOrder(t, app, workOrderID, []string{svc1ID})

	transitionWorkOrder(t, app, workOrderID, "AGUARDANDO_APROVACAO")
	_, ok := findLatestEmailBySubjectContains(mockEmail.Messages(), "Orçamento")
	require.True(t, ok)

	svc2ID := createTestWorkshopService(t, app, "Serviço Ajuste B", 7000, 45)
	resp, err := flowPostJSON(app,
		fmt.Sprintf("/work-orders/%s/services", workOrderID),
		[]map[string]any{{"service_id": svc2ID}},
	)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)

	msgs := mockEmail.Messages()
	budgetEmails := 0
	for _, msg := range msgs {
		if strings.Contains(msg.Subject, "Orçamento") {
			budgetEmails++
		}
	}
	require.Equal(t, 2, budgetEmails, "adjusting services while waiting approval should send a new budget")
	latest, ok := findLatestEmailBySubjectContains(msgs, "Orçamento")
	require.True(t, ok)
	assert.Contains(t, latest.Body, "Serviço Ajuste B")
}

// =============================================================================
// Test: Supply shortage detection — when supplies needed exceed stock_quantity,
// the budget adds 2 extra days to estimated time for the affected services.
// Cobre os passos do fluxo DDD: "Calcula atraso do serviço porque nem todas as
// peças estão em estoque", "Verifica se todas as peças estão disponíveis".
// =============================================================================

func TestIntegration_Flow_SupplyShortageDelay(t *testing.T) {
	app, _, mockEmail := setupFlowAppWithBudget(t)

	customerID := createTestCustomer(t, app, "Cliente Shortage", "shortage@example.com", "12345678909", "CPF")
	vehicleID := createTestVehicle(t, app, customerID, "SHR1A23", "VW", "Polo", 2023)
	workOrderID := createTestWorkOrder(t, app, "Shortage test", customerID, vehicleID)

	// Service with 30min estimated time
	svcID := createTestWorkshopService(t, app, "Serviço com Falta", 10000, 30)
	wosIDs := addServicesToWorkOrder(t, app, workOrderID, []string{svcID})

	// Create a supply with stock_quantity=2 but request quantity=5 → shortage
	supplyID := createTestSupply(t, app, "Peça Rara", "PECA", 5000, 2, 1)
	resp, err := flowPostJSON(app,
		fmt.Sprintf("/work-orders/%s/services/%s/supplies", workOrderID, wosIDs[0]),
		[]map[string]any{{"supply_id": supplyID, "quantity": 5}},
	)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)

	// Transition to AGUARDANDO_APROVACAO — triggers budget with shortage
	resp, err = flowPutJSON(app, "/work-orders/"+workOrderID, map[string]any{
		"status": "AGUARDANDO_APROVACAO",
	})
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	// Verify the email body contains extended time (30min + 2 days = "2 dias")
	t.Run("EmailContemAtrasoDeEstoque", func(t *testing.T) {
		msgs := mockEmail.Messages()
		msg, ok := findLatestEmailBySubjectContains(msgs, "Orçamento")
		require.True(t, ok)
		body := msg.Body
		assert.Contains(t, body, "2 dias",
			"email should show extended estimated time when supply has shortage (+2 days)")
	})
}

// =============================================================================
// Test: No shortage — when all supplies are in stock, budget shows only the
// normal estimated time without extra delay.
// =============================================================================

func TestIntegration_Flow_NoShortageNormalTime(t *testing.T) {
	app, _, mockEmail := setupFlowAppWithBudget(t)

	customerID := createTestCustomer(t, app, "Cliente Estoque OK", "estoque@example.com", "12345678909", "CPF")
	vehicleID := createTestVehicle(t, app, customerID, "STK1A23", "Toyota", "Yaris", 2024)
	workOrderID := createTestWorkOrder(t, app, "No shortage test", customerID, vehicleID)

	// Service with 90min estimated time
	svcID := createTestWorkshopService(t, app, "Serviço Estoque OK", 8000, 90)
	wosIDs := addServicesToWorkOrder(t, app, workOrderID, []string{svcID})
	_ = wosIDs

	// Supply with stock=50, quantity=2 → no shortage
	supplyID := createTestSupply(t, app, "Peça Abundante", "PECA", 2000, 50, 5)
	resp, err := flowPostJSON(app,
		fmt.Sprintf("/work-orders/%s/services/%s/supplies", workOrderID, wosIDs[0]),
		[]map[string]any{{"supply_id": supplyID, "quantity": 2}},
	)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)

	resp, err = flowPutJSON(app, "/work-orders/"+workOrderID, map[string]any{
		"status": "AGUARDANDO_APROVACAO",
	})
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	t.Run("EmailSemAtraso", func(t *testing.T) {
		msgs := mockEmail.Messages()
		msg, ok := findLatestEmailBySubjectContains(msgs, "Orçamento")
		require.True(t, ok)
		body := msg.Body
		// 90 min = "1 hora e 30 min", should NOT contain "dias"
		assert.Contains(t, body, "1 hora",
			"email should show normal estimated time without delay")
		assert.NotContains(t, body, "dias",
			"email should NOT contain extra days when stock is sufficient")
	})
}

// =============================================================================
// Test: Full flow with budget — happy path including budget generation, email
// notification, and approval through the public links from the email.
// Cobre o fluxo DDD completo: orçamento → email → aprovação via link → entrega.
// =============================================================================

func TestIntegration_Flow_HappyPathWithBudgetAndEmail(t *testing.T) {
	app, _, mockEmail := setupFlowAppWithBudget(t)

	customerID := createTestCustomer(t, app, "João Completo", "joao.completo@example.com", "12345678909", "CPF")
	vehicleID := createTestVehicle(t, app, customerID, "FUL1A23", "Honda", "City", 2023)
	workOrderID := createTestWorkOrder(t, app, "Fluxo completo com email", customerID, vehicleID)

	svcID := createTestWorkshopService(t, app, "Revisão Completa Email", 20000, 60)
	wosIDs := addServicesToWorkOrder(t, app, workOrderID, []string{svcID})

	// Transition to AGUARDANDO_APROVACAO → sends email
	transitionWorkOrder(t, app, workOrderID, "AGUARDANDO_APROVACAO")

	// Extract the approve-all link from the email body
	t.Run("ClienteAprovaViaLinkDoEmail", func(t *testing.T) {
		msgs := mockEmail.Messages()
		msg, ok := findLatestEmailBySubjectContains(msgs, "Orçamento")
		require.True(t, ok)
		body := msg.Body

		// Find the approve-all URL in the email
		approveAllSuffix := fmt.Sprintf("/public/approvals/work-orders/%s/approve-all", workOrderID)
		assert.True(t, strings.Contains(body, approveAllSuffix),
			"email should contain approve-all link")

		// Use the link to approve — same as the client would
		resp, err := flowGet(app, approveAllSuffix)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	// After approval via email link, work order should be APROVADO
	t.Run("OSAprovadaAposCliqueLinkEmail", func(t *testing.T) {
		resp, err := flowGet(app, "/work-orders/"+workOrderID)
		require.NoError(t, err)
		body := flowReadBody(t, resp)
		assert.Equal(t, "APROVADO", body["status"])
		assert.NotNil(t, body["approved_at"])
		assert.NotNil(t, body["quote_sent_at"])
	})

	// Continue the flow: EM_EXECUCAO → start/finalize services → auto-FINALIZADA → ENTREGUE
	t.Run("FluxoContinuaAteEntrega", func(t *testing.T) {
		transitionWorkOrder(t, app, workOrderID, "EM_EXECUCAO")
		startAndFinalizeService(t, app, workOrderID, wosIDs[0])

		resp, err := flowGet(app, "/work-orders/"+workOrderID)
		require.NoError(t, err)
		body := flowReadBody(t, resp)
		assert.Equal(t, "FINALIZADA", body["status"])

		transitionWorkOrder(t, app, workOrderID, "ENTREGUE")
	})
}

func TestIntegration_Flow_StatusNotificationsForAllTransitions(t *testing.T) {
	app, _, mockEmail := setupFlowAppWithBudget(t)

	customerID := createTestCustomer(t, app, "Cliente Status", "status@example.com", "12345678909", "CPF")
	vehicleID := createTestVehicle(t, app, customerID, "STS1A23", "Ford", "Ka", 2021)
	workOrderID := createTestWorkOrder(t, app, "Notificações de status", customerID, vehicleID)

	svcID := createTestWorkshopService(t, app, "Serviço Status", 9000, 45)
	wosIDs := addServicesToWorkOrder(t, app, workOrderID, []string{svcID})

	assertStatusEmail(t, mockEmail, "Em diagnóstico")

	transitionWorkOrder(t, app, workOrderID, "AGUARDANDO_APROVACAO")
	budgetMsg, ok := findLatestEmailBySubjectContains(mockEmail.Messages(), "Orçamento")
	require.True(t, ok)
	assert.Contains(t, budgetMsg.Body, "Em diagnóstico")
	assert.Contains(t, budgetMsg.Body, "Aguardando aprovação")

	approveAllSuffix := fmt.Sprintf("/public/approvals/work-orders/%s/approve-all", workOrderID)
	resp, err := flowGet(app, approveAllSuffix)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	assertStatusEmail(t, mockEmail, "Aprovada")

	transitionWorkOrder(t, app, workOrderID, "EM_EXECUCAO")
	assertStatusEmail(t, mockEmail, "Em execução")

	startAndFinalizeService(t, app, workOrderID, wosIDs[0])
	assertStatusEmail(t, mockEmail, "Finalizada")

	transitionWorkOrder(t, app, workOrderID, "ENTREGUE")
	assertStatusEmail(t, mockEmail, "Entregue")
}

func TestIntegration_Flow_IdempotentTransitionDoesNotDuplicateEmail(t *testing.T) {
	app, _, mockEmail := setupFlowAppWithBudget(t)

	customerID := createTestCustomer(t, app, "Cliente Idempotente", "idempotente@example.com", "12345678909", "CPF")
	vehicleID := createTestVehicle(t, app, customerID, "IDP1A23", "Chevrolet", "Onix", 2020)
	workOrderID := createTestWorkOrder(t, app, "Idempotência", customerID, vehicleID)

	svcID := createTestWorkshopService(t, app, "Serviço Idempotente", 5000, 20)
	addServicesToWorkOrder(t, app, workOrderID, []string{svcID})

	before := len(mockEmail.Messages())
	transitionWorkOrder(t, app, workOrderID, "EM_DIAGNOSTICO")
	transitionWorkOrder(t, app, workOrderID, "EM_DIAGNOSTICO")
	after := len(mockEmail.Messages())

	assert.Equal(t, before, after, "idempotent transition should not send duplicate email")
}

func TestIntegration_Flow_EmailFailureDoesNotRollbackTransition(t *testing.T) {
	app, _, mockEmail := setupFlowAppWithBudget(t)

	customerID := createTestCustomer(t, app, "Cliente Falha Email", "falha@example.com", "12345678909", "CPF")
	vehicleID := createTestVehicle(t, app, customerID, "FAL1A23", "Renault", "Kwid", 2019)
	workOrderID := createTestWorkOrder(t, app, "Falha email", customerID, vehicleID)
	svcID := createTestWorkshopService(t, app, "Serviço Falha", 4000, 15)
	addServicesToWorkOrder(t, app, workOrderID, []string{svcID})

	mockEmail.failNext = true
	resp, err := flowPutJSON(app, "/work-orders/"+workOrderID, map[string]any{"status": "AGUARDANDO_APROVACAO"})
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	woResp, err := flowGet(app, "/work-orders/"+workOrderID)
	require.NoError(t, err)
	body := flowReadBody(t, woResp)
	assert.Equal(t, "AGUARDANDO_APROVACAO", body["status"])
	assert.Nil(t, body["quote_sent_at"], "quote_sent_at should not be set when email fails")
}

func assertStatusEmail(t *testing.T, mockEmail *flowMockEmail, statusLabel string) {
	t.Helper()
	msg, ok := findLatestEmailBySubjectContains(mockEmail.Messages(), statusLabel)
	require.True(t, ok, "expected status email for %s", statusLabel)
	assert.Contains(t, msg.Body, "Status anterior:")
	assert.Contains(t, msg.Body, "Novo status:")
}

// =============================================================================
// Helper functions to reduce boilerplate in tests
// =============================================================================

func createTestCustomer(t *testing.T, app *fiber.App, name, email, document, docType string) string {
	t.Helper()
	resp, err := flowPostJSON(app, "/customers", map[string]any{
		"name":          name,
		"email":         email,
		"document":      document,
		"document_type": docType,
	})
	require.NoError(t, err)
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)
	body := flowReadBody(t, resp)
	return body["id"].(string)
}

func createTestVehicle(t *testing.T, app *fiber.App, customerID, plate, brand, model string, year int) string {
	t.Helper()
	resp, err := flowPostJSON(app, "/vehicles", map[string]any{
		"license_plate": plate,
		"customer_id":   customerID,
		"brand":         brand,
		"model":         model,
		"year":          year,
	})
	require.NoError(t, err)
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)
	body := flowReadBody(t, resp)
	return body["id"].(string)
}

func createTestWorkOrder(t *testing.T, app *fiber.App, title, customerID, vehicleID string) string {
	t.Helper()
	resp, err := flowPostJSON(app, "/work-orders", map[string]any{
		"title":       title,
		"customer_id": customerID,
		"vehicle_id":  vehicleID,
	})
	require.NoError(t, err)
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)
	body := flowReadBody(t, resp)
	return body["id"].(string)
}

func createTestWorkshopService(t *testing.T, app *fiber.App, title string, priceCents, estMinutes int) string {
	t.Helper()
	resp, err := flowPostJSON(app, "/services", map[string]any{
		"title":                  title,
		"price_cents":            priceCents,
		"estimated_time_minutes": estMinutes,
	})
	require.NoError(t, err)
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)
	body := flowReadBody(t, resp)
	return body["id"].(string)
}

func createTestSupply(t *testing.T, app *fiber.App, title, supplyType string, priceCents, stock, minStock int) string {
	t.Helper()
	resp, err := flowPostJSON(app, "/supplies", map[string]any{
		"title":          title,
		"type":           supplyType,
		"price_cents":    priceCents,
		"stock_quantity": stock,
		"minimum_stock":  minStock,
	})
	require.NoError(t, err)
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)
	body := flowReadBody(t, resp)
	return body["id"].(string)
}

func addServicesToWorkOrder(t *testing.T, app *fiber.App, workOrderID string, serviceIDs []string) []string {
	t.Helper()
	items := make([]map[string]any, len(serviceIDs))
	for i, id := range serviceIDs {
		items[i] = map[string]any{"service_id": id}
	}

	resp, err := flowPostJSON(app, fmt.Sprintf("/work-orders/%s/services", workOrderID), items)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)

	body := flowReadBodyArray(t, resp)
	ids := make([]string, len(body))
	for i, item := range body {
		ids[i] = item["id"].(string)
	}
	return ids
}

func startAndFinalizeService(t *testing.T, app *fiber.App, workOrderID, wosID string) {
	t.Helper()
	resp, err := flowPutJSON(app, fmt.Sprintf("/work-orders/%s/services/%s/start", workOrderID, wosID), map[string]any{})
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode, "failed to start service %s", wosID)

	resp, err = flowPutJSON(app, fmt.Sprintf("/work-orders/%s/services/%s/finalize", workOrderID, wosID), map[string]any{})
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode, "failed to finalize service %s", wosID)
}

func transitionWorkOrder(t *testing.T, app *fiber.App, workOrderID, status string) {
	t.Helper()
	resp, err := flowPutJSON(app, "/work-orders/"+workOrderID, map[string]any{
		"status": status,
	})
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode,
		"failed to transition work order %s to %s", workOrderID, status)
}

func insertWorkOrderRow(t *testing.T, pool *pgxpool.Pool, code, status string, receivedAt time.Time, customerID, vehicleID, openedBy string) string {
	t.Helper()
	id := uuid.New()
	now := time.Now()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO work_orders (
			id, code, title, description, customer_id, vehicle_id, opened_by_user_id,
			assigned_technician_id, status, total_estimated_price_cents, received_at,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		id, code, "OS "+code, nil,
		uuid.MustParse(customerID), uuid.MustParse(vehicleID), uuid.MustParse(openedBy),
		nil, status, 0, receivedAt, now, now,
	)
	require.NoError(t, err)
	return id.String()
}

func listWorkOrderIDs(t *testing.T, app *fiber.App, query string) (ids []string, statuses []string, total int, totalPages int) {
	t.Helper()
	resp, err := flowGet(app, "/work-orders"+query)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	body := flowReadBody(t, resp)
	total = int(body["total"].(float64))
	totalPages = int(body["total_pages"].(float64))

	data, _ := body["data"].([]any)
	for _, item := range data {
		wo := item.(map[string]any)
		ids = append(ids, wo["id"].(string))
		statuses = append(statuses, wo["status"].(string))
	}
	return ids, statuses, total, totalPages
}

func TestIntegration_Flow_OperationalListingRules(t *testing.T) {
	app, pool := setupFlowApp(t)
	openedBy := seedTestUser(t, pool)

	customerA := createTestCustomer(t, app, "Listagem A", "listagem.a@example.com", "12345678909", "CPF")
	vehicleA := createTestVehicle(t, app, customerA, "LST1A23", "Fiat", "Uno", 2020)

	customerB := createTestCustomer(t, app, "Listagem B", "listagem.b@example.com", "52998224725", "CPF")
	vehicleB := createTestVehicle(t, app, customerB, "LST1B23", "Honda", "Fit", 2021)

	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	h := func(n int) time.Time { return base.Add(time.Duration(n) * time.Hour) }

	e2 := insertWorkOrderRow(t, pool, "OS-LST-E2", "EM_EXECUCAO", h(2), customerA, vehicleA, openedBy)
	e1 := insertWorkOrderRow(t, pool, "OS-LST-E1", "EM_EXECUCAO", h(4), customerA, vehicleA, openedBy)
	a1 := insertWorkOrderRow(t, pool, "OS-LST-A1", "AGUARDANDO_APROVACAO", h(1), customerA, vehicleA, openedBy)
	d1 := insertWorkOrderRow(t, pool, "OS-LST-D1", "EM_DIAGNOSTICO", h(5), customerA, vehicleA, openedBy)
	r2 := insertWorkOrderRow(t, pool, "OS-LST-R2", "RECEBIDA", h(1), customerA, vehicleA, openedBy)
	r1 := insertWorkOrderRow(t, pool, "OS-LST-R1", "RECEBIDA", h(3), customerA, vehicleA, openedBy)
	ap1 := insertWorkOrderRow(t, pool, "OS-LST-AP1", "APROVADO", h(0), customerA, vehicleA, openedBy)
	cancA := insertWorkOrderRow(t, pool, "OS-LST-C1", "CANCELADA", h(1), customerA, vehicleA, openedBy)
	insertWorkOrderRow(t, pool, "OS-LST-F1", "FINALIZADA", h(1), customerA, vehicleA, openedBy)
	insertWorkOrderRow(t, pool, "OS-LST-EN1", "ENTREGUE", h(1), customerA, vehicleA, openedBy)

	t.Run("DefaultListing_ExcludesTerminalAndCancelled_OrdersByPriorityThenReceivedAt", func(t *testing.T) {
		ids, statuses, total, _ := listWorkOrderIDs(t, app, "?customerId="+customerA+"&limit=50")

		assert.Equal(t, 7, total, "total deve considerar as exclusões")
		assert.NotContains(t, statuses, "FINALIZADA")
		assert.NotContains(t, statuses, "ENTREGUE")
		assert.NotContains(t, statuses, "CANCELADA")

		assert.Equal(t, []string{e2, e1, a1, d1, r2, r1, ap1}, ids,
			"ordem esperada: EM_EXECUCAO(received asc) > AGUARDANDO_APROVACAO > EM_DIAGNOSTICO > RECEBIDA(received asc) > APROVADO")
	})

	t.Run("StatusFilterCannotBypassExclusion_Finished", func(t *testing.T) {
		ids, _, total, totalPages := listWorkOrderIDs(t, app, "?customerId="+customerA+"&status=FINALIZADA")
		assert.Equal(t, 0, total, "FINALIZADA nunca aparece, mesmo com filtro explícito")
		assert.Equal(t, 0, totalPages)
		assert.Empty(t, ids)
	})

	t.Run("StatusFilterCannotBypassExclusion_Delivered", func(t *testing.T) {
		ids, _, total, _ := listWorkOrderIDs(t, app, "?customerId="+customerA+"&status=ENTREGUE")
		assert.Equal(t, 0, total, "ENTREGUE nunca aparece, mesmo com filtro explícito")
		assert.Empty(t, ids)
	})

	t.Run("CancelledIsHiddenByDefaultButReachableViaExplicitFilter", func(t *testing.T) {
		ids, statuses, total, _ := listWorkOrderIDs(t, app, "?customerId="+customerA+"&status=CANCELADA")
		assert.Equal(t, 1, total)
		assert.Equal(t, []string{cancA}, ids)
		assert.Equal(t, []string{"CANCELADA"}, statuses)
	})

	t.Run("ExplicitStatusFilter_ReturnsOnlyThatStatus_OrderedByReceivedAt", func(t *testing.T) {
		ids, statuses, total, _ := listWorkOrderIDs(t, app, "?customerId="+customerA+"&status=EM_EXECUCAO")
		assert.Equal(t, 2, total)
		assert.Equal(t, []string{e2, e1}, ids, "received_at ASC dentro do status filtrado")
		assert.Equal(t, []string{"EM_EXECUCAO", "EM_EXECUCAO"}, statuses)
	})

	t.Run("ApprovedRemainsListable", func(t *testing.T) {
		ids, _, total, _ := listWorkOrderIDs(t, app, "?customerId="+customerA+"&status=APROVADO")
		assert.Equal(t, 1, total)
		assert.Equal(t, []string{ap1}, ids)
	})

	t.Run("PaginationConsidersExclusions", func(t *testing.T) {
		idsP1, _, total, totalPages := listWorkOrderIDs(t, app, "?customerId="+customerA+"&limit=3&page=1")
		assert.Equal(t, 7, total)
		assert.Equal(t, 3, totalPages)
		assert.Equal(t, []string{e2, e1, a1}, idsP1)

		idsP3, _, _, _ := listWorkOrderIDs(t, app, "?customerId="+customerA+"&limit=3&page=3")
		assert.Equal(t, []string{ap1}, idsP3, "última página traz apenas a OS de menor prioridade")
	})

	t.Run("DateFilterOnReceivedAt_ComposesWithExclusion", func(t *testing.T) {
		bMid := insertWorkOrderRow(t, pool, "OS-LST-BMID", "RECEBIDA", time.Date(2026, 1, 15, 9, 0, 0, 0, time.UTC), customerB, vehicleB, openedBy)
		insertWorkOrderRow(t, pool, "OS-LST-BOLD", "RECEBIDA", time.Date(2025, 12, 1, 9, 0, 0, 0, time.UTC), customerB, vehicleB, openedBy)
		insertWorkOrderRow(t, pool, "OS-LST-BNEW", "RECEBIDA", time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC), customerB, vehicleB, openedBy)
		insertWorkOrderRow(t, pool, "OS-LST-BCAN", "CANCELADA", time.Date(2026, 1, 16, 9, 0, 0, 0, time.UTC), customerB, vehicleB, openedBy)
		insertWorkOrderRow(t, pool, "OS-LST-BFIN", "FINALIZADA", time.Date(2026, 1, 17, 9, 0, 0, 0, time.UTC), customerB, vehicleB, openedBy)

		ids, statuses, total, _ := listWorkOrderIDs(t, app, "?customerId="+customerB+"&from=2026-01-01&to=2026-01-31&limit=50")
		assert.Equal(t, 1, total, "apenas a OS dentro do intervalo e não excluída")
		assert.Equal(t, []string{bMid}, ids)
		assert.Equal(t, []string{"RECEBIDA"}, statuses)
	})
}
