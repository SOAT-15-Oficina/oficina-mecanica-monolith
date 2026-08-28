package routes

import (
	"context"
	"time"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/config"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/handler"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/repository"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/routes/middlewares"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/service"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/packages/email"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
)

type databasePinger interface {
	Ping(context.Context) error
}

func RegisterRoutes(app *fiber.App, db *pgxpool.Pool, cfg *config.Config, emailProv email.Provider) {
	if db == nil {
		registerHealthRoutes(app, nil)
	} else {
		registerHealthRoutes(app, db)
	}

	registerSwagger(app)
	registerUsers(app, db, cfg.JWT.SecretKey)
	registerCustomer(app, db, cfg.JWT.SecretKey)
	registerVehicle(app, db, cfg.JWT.SecretKey)
	registerSupply(app, db, cfg.JWT.SecretKey)
	registerWorkOrderServicePublic(app, db, emailProv, cfg.Server.BaseURL)
	registerPublicWorkOrder(app, db)
	registerWorkOrder(app, db, cfg.JWT.SecretKey, emailProv, cfg.Server.BaseURL)
	registerWorkshopService(app, db, cfg.JWT.SecretKey)
}

func registerHealthRoutes(app *fiber.App, db databasePinger) {
	app.Get("/ping", func(c fiber.Ctx) error {
		return c.SendString("Pong")
	})

	app.Get("/ready", func(c fiber.Ctx) error {
		if db == nil {
			return c.SendStatus(fiber.StatusServiceUnavailable)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			return c.SendStatus(fiber.StatusServiceUnavailable)
		}

		return c.SendStatus(fiber.StatusOK)
	})
}

// POST /auth/register e POST /auth/login sao servidos pela Lambda no
// oficina-mecanica-serverless, roteados pelo API Gateway. O monolito apenas
// valida o token emitido la e mantem a manutencao administrativa de usuarios.
func registerUsers(app *fiber.App, db *pgxpool.Pool, jwtSecretKey string) {
	userRepo := repository.NewUserRepository(db)
	userSvc := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userSvc)

	users := app.Group("/users", middlewares.Auth(jwtSecretKey), middlewares.RequireRoles(middlewares.RoleAdmin))
	users.Get("/", userHandler.GetAll)
	users.Get("/:id", userHandler.GetByID)
	users.Put("/:id", userHandler.Update)
	users.Delete("/:id", userHandler.Delete)
}

func registerCustomer(app *fiber.App, db *pgxpool.Pool, jwtSecretKey string) {
	customerRepo := repository.NewCustomerRepository(db)
	customerSvc := service.NewCustomerService(customerRepo)
	customerHandler := handler.NewCustomerHandler(customerSvc)

	group := app.Group("/customers", middlewares.Auth(jwtSecretKey), middlewares.RequireRoles(middlewares.RoleAdmin, middlewares.RoleEmployee))
	group.Post("/", customerHandler.Create)
	group.Get("/", customerHandler.GetAll)
	group.Get("/:id", customerHandler.GetByID)
	group.Put("/:id", customerHandler.Update)
	group.Delete("/:id", customerHandler.Delete)

}

func registerVehicle(app *fiber.App, db *pgxpool.Pool, jwtSecretKey string) {
	vehicleRepo := repository.NewVehicleRepository(db)
	vehicleSvc := service.NewVehicleService(vehicleRepo)
	vehicleHandler := handler.NewVehicleHandler(vehicleSvc)

	group := app.Group("/vehicles", middlewares.Auth(jwtSecretKey), middlewares.RequireRoles(middlewares.RoleAdmin, middlewares.RoleEmployee))
	group.Post("/", vehicleHandler.Create)
	group.Get("/", vehicleHandler.GetAll)
	group.Get("/:id", vehicleHandler.GetByID)
	group.Put("/:id", vehicleHandler.Update)
	group.Delete("/:id", vehicleHandler.Delete)
}

func registerWorkshopService(app *fiber.App, db *pgxpool.Pool, jwtSecretKey string) {
	wsRepo := repository.NewWorkshopServiceRepository(db)
	wsSvc := service.NewWorkshopServiceManager(wsRepo)
	wsHandler := handler.NewWorkshopServiceHandler(wsSvc)

	group := app.Group("/services", middlewares.Auth(jwtSecretKey), middlewares.RequireRoles(middlewares.RoleAdmin, middlewares.RoleEmployee))
	group.Post("/", wsHandler.Create)
	group.Get("/avg-execution-time", wsHandler.GetAvgExecutionTime)
	group.Get("/", wsHandler.GetAll)
	group.Get("/:id", wsHandler.GetByID)
	group.Put("/:id", wsHandler.Update)
	group.Delete("/:id", wsHandler.Delete)
}

func registerSupply(app *fiber.App, db *pgxpool.Pool, jwtSecretKey string) {
	supplyRepo := repository.NewSupplyRepository(db)
	wosRepo := repository.NewWorkOrderServiceRepository(db)
	supplySvc := service.NewSupplyService(supplyRepo, wosRepo)
	supplyHandler := handler.NewSupplyHandler(supplySvc)

	group := app.Group("/supplies", middlewares.Auth(jwtSecretKey), middlewares.RequireRoles(middlewares.RoleAdmin, middlewares.RoleEmployee))
	group.Get("/pending-purchases", supplyHandler.PendingPurchases)
	group.Post("/", supplyHandler.Create)
	group.Get("/", supplyHandler.GetAll)
	group.Get("/:id", supplyHandler.GetByID)
	group.Put("/:id", supplyHandler.Update)
	group.Delete("/:id", supplyHandler.Delete)
}

func registerWorkOrder(app *fiber.App, db *pgxpool.Pool, jwtSecretKey string, emailProv email.Provider, baseURL string) {
	workOrderRepo := repository.NewWorkOrderRepository(db)
	wosRepo := repository.NewWorkOrderServiceRepository(db)
	customerRepo := repository.NewCustomerRepository(db)
	vehicleRepo := repository.NewVehicleRepository(db)
	wsRepo := repository.NewWorkshopServiceRepository(db)
	supplyRepo := repository.NewSupplyRepository(db)
	var budgetNotifier application.BudgetNotificationSender
	if emailProv != nil {
		budgetNotifier = email.NewWorkOrderNotificationSender(emailProv)
	}
	budgetSvc := service.NewBudgetService(workOrderRepo, wosRepo, customerRepo, budgetNotifier, baseURL)
	statusSvc := service.NewWorkOrderStatusServiceWithNotifications(workOrderRepo, wosRepo, customerRepo, emailProv, baseURL)
	workOrderSvc := service.NewWorkOrderService(workOrderRepo, vehicleRepo)
	creationSvc := service.NewWorkOrderCreationService(workOrderRepo, wosRepo, wsRepo, supplyRepo, statusSvc, service.WithBudgetRefresh(budgetSvc))
	userRepo := repository.NewUserRepository(db)
	userSvc := service.NewUserService(userRepo)
	workOrderHandler := handler.NewWorkOrderHandler(workOrderSvc, creationSvc, statusSvc, userSvc)

	group := app.Group("/work-orders", middlewares.Auth(jwtSecretKey), middlewares.RequireRoles(middlewares.RoleAdmin, middlewares.RoleEmployee))
	group.Post("/", workOrderHandler.Create)
	group.Get("/", workOrderHandler.GetAll)
	group.Get("/:id", workOrderHandler.GetByID)
	group.Put("/:id", workOrderHandler.Update)
	group.Post("/:id/services", workOrderHandler.AddServices)
	group.Delete("/:id/services/:wosId", workOrderHandler.RemoveService)
	group.Put("/:id/services/:wosId/start", workOrderHandler.StartService)
	group.Put("/:id/services/:wosId/finalize", workOrderHandler.FinalizeService)
	group.Post("/:id/services/:wosId/supplies", workOrderHandler.AddSupplies)
	group.Delete("/:id/services/:wosId/supplies/:supplyId", workOrderHandler.RemoveSupplyFromService)
}

func registerPublicWorkOrder(app *fiber.App, db *pgxpool.Pool) {
	woRepo := repository.NewWorkOrderRepository(db)
	customerRepo := repository.NewCustomerRepository(db)
	wosRepo := repository.NewWorkOrderServiceRepository(db)
	publicSvc := service.NewPublicWorkOrderService(woRepo, customerRepo, wosRepo)
	publicHandler := handler.NewPublicWorkOrderHandler(publicSvc)

	public := app.Group("/public/work-orders")
	public.Get("/:code", publicHandler.GetByCode)
}

func registerWorkOrderServicePublic(app *fiber.App, db *pgxpool.Pool, emailProv email.Provider, baseURL string) {
	wosRepo := repository.NewWorkOrderServiceRepository(db)
	woRepo := repository.NewWorkOrderRepository(db)
	customerRepo := repository.NewCustomerRepository(db)
	notificationSender := email.NewWorkOrderNotificationSender(emailProv)
	statusSvc := service.NewWorkOrderStatusServiceWithNotifications(woRepo, wosRepo, customerRepo, emailProv, baseURL)
	itemSvc := service.NewWorkOrderItemService(wosRepo, woRepo, statusSvc,
		service.WithPurchaseAlert(notificationSender, "compras@oficina.com"))
	wosHandler := handler.NewWorkOrderServiceHandler(itemSvc)

	approval := app.Group("/public/approvals")
	approval.Get("/services/:workOrderServiceId/approve", wosHandler.Approve)
	approval.Get("/services/:workOrderServiceId/reject", wosHandler.Reject)
	approval.Get("/work-orders/:workOrderId/approve-all", wosHandler.ApproveAll)
	approval.Get("/work-orders/:workOrderId/reject-all", wosHandler.RejectAll)
}
