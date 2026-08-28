package handler

import (
	"errors"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	pgErrUniqueViolation     = "23505"
	pgErrForeignKeyViolation = "23503"
	pgErrNotNullViolation    = "23502"
	pgErrCheckViolation      = "23514"
)

var uniqueConstraintMessages = map[string]string{
	"users_username_key":                            "username already taken",
	"customers_document_key":                        "document already registered",
	"vehicles_license_plate_key":                    "license plate already registered",
	"supplies_code_key":                             "supply code already registered",
	"idx_services_title_unique":                     "service title already exists",
	"idx_work_order_service_supplies_wos_supply_id": "supply already added to this service",
}

var foreignKeyMessages = map[string]string{
	"fk_vehicles_customer":      "customer not found",
	"fk_work_orders_customer":   "customer not found",
	"fk_work_orders_vehicle":    "vehicle not found",
	"fk_work_orders_opened_by":  "user not found",
	"fk_work_orders_technician": "user not found",
	"fk_wos_work_order":         "work order not found",
	"fk_wos_service":            "service not found",
	"fk_wosi_work_order_s":      "work order service not found",
	"fk_wosi_supply":            "insumo vinculado a uma OS e não pode ser removido",
	"fk_wosh_work_order":        "work order service not found",
}

func dbErrResponse(c fiber.Ctx, err error, notFoundMsg string) (bool, error) {
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, application.ErrNotFound) {
		return true, c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": notFoundMsg})
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgErrUniqueViolation:
			msg := uniqueConstraintMessages[pgErr.ConstraintName]
			if msg == "" {
				msg = "resource already exists"
			}
			return true, c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": msg})
		case pgErrForeignKeyViolation:
			msg := foreignKeyMessages[pgErr.ConstraintName]
			if msg == "" {
				msg = "resource conflicts with an existing relationship"
			}
			return true, c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": msg})
		case pgErrNotNullViolation, pgErrCheckViolation:
			return true, c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid resource data"})
		}
	}

	return false, nil
}

func internalServerError(c fiber.Ctx) error {
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
}
