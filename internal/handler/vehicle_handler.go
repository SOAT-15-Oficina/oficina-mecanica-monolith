package handler

import (
	"errors"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type VehicleHandler struct {
	svc service.VehicleService
}

func NewVehicleHandler(svc service.VehicleService) *VehicleHandler {
	return &VehicleHandler{svc: svc}
}

func (h *VehicleHandler) Create(c fiber.Ctx) error {
	var vehicle domain.Vehicle
	if err := c.Bind().JSON(&vehicle); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	result, err := h.svc.Create(c.Context(), &vehicle)
	if err != nil {
		var valErr *domain.VehicleValidationError
		if errors.As(err, &valErr) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if handled, resp := dbErrResponse(c, err, "vehicle not found"); handled {
			return resp
		}
		return internalServerError(c)
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

func (h *VehicleHandler) GetAll(c fiber.Ctx) error {
	filters := domain.VehicleListFilters{}

	customerID, err := queryWithAlias(c, "customer_id", "customerId")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if customerID != "" {
		id, err := uuid.Parse(customerID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "customer_id must be a valid UUID"})
		}
		filters.CustomerID = id
	}

	vehicles, err := h.svc.GetAllWithFilters(c.Context(), filters)
	if err != nil {
		return internalServerError(c)
	}

	if vehicles == nil {
		vehicles = []domain.Vehicle{}
	}

	return c.JSON(fiber.Map{"data": vehicles})
}

func (h *VehicleHandler) GetByID(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	vehicle, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		if handled, resp := dbErrResponse(c, err, "vehicle not found"); handled {
			return resp
		}
		return internalServerError(c)
	}

	return c.JSON(vehicle)
}

func (h *VehicleHandler) Update(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var vehicle domain.Vehicle
	if err := c.Bind().JSON(&vehicle); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	vehicle.ID = id

	result, err := h.svc.Update(c.Context(), &vehicle)
	if err != nil {
		var valErr *domain.VehicleValidationError
		if errors.As(err, &valErr) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if handled, resp := dbErrResponse(c, err, "vehicle not found"); handled {
			return resp
		}
		return internalServerError(c)
	}

	return c.JSON(result)
}

func (h *VehicleHandler) Delete(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	if err := h.svc.Delete(c.Context(), id); err != nil {
		if handled, resp := dbErrResponse(c, err, "vehicle not found"); handled {
			return resp
		}
		return internalServerError(c)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
