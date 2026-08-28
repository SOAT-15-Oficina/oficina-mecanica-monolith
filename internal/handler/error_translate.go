package handler

import (
	"errors"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/service"
	"github.com/gofiber/fiber/v3"
)

// mapErrorResponse centralizes mapping of domain/service errors to HTTP responses.
// Returns (true, err) when it already wrote a response to the client.
func mapErrorResponse(c fiber.Ctx, err error, notFoundMsg string) (bool, error) {
	var validationErr *application.ValidationError
	if errors.As(err, &validationErr) {
		return true, c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": validationErr.Error()})
	}
	if errors.Is(err, service.ErrWorkOrderInvalidStatusForItems) || errors.Is(err, service.ErrInvalidStatusTransition) {
		return true, c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
	}

	if errors.Is(err, service.ErrWorkshopServiceInactive) {
		return true, c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
	}

	if errors.Is(err, service.ErrWorkOrderServiceOwnership) {
		return true, c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
	}

	if errors.Is(err, service.ErrVehicleNotBelongingToCustomer) {
		return true, c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
	}

	if errors.Is(err, application.ErrNotFound) {
		return true, c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": notFoundMsg})
	}

	// Fallback to DB-specific translator which may inspect pgx errors.
	return dbErrResponse(c, err, notFoundMsg)
}
