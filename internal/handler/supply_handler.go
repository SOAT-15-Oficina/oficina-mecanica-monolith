package handler

import (
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type SupplyHandler struct {
	svc service.SupplyService
}

func NewSupplyHandler(svc service.SupplyService) *SupplyHandler {
	return &SupplyHandler{svc: svc}
}

func (h *SupplyHandler) Create(c fiber.Ctx) error {
	var supply domain.Supply
	if err := c.Bind().JSON(&supply); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	result, err := h.svc.Create(c.Context(), &supply)
	if err != nil {
		if handled, resp := dbErrResponse(c, err, "supply not found"); handled {
			return resp
		}
		return internalServerError(c)
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

func (h *SupplyHandler) GetAll(c fiber.Ctx) error {
	supplies, err := h.svc.GetAll(c.Context())
	if err != nil {
		return internalServerError(c)
	}

	if supplies == nil {
		supplies = []domain.Supply{}
	}

	return c.JSON(fiber.Map{"data": supplies})
}

func (h *SupplyHandler) GetByID(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	supply, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		if handled, resp := dbErrResponse(c, err, "supply not found"); handled {
			return resp
		}
		return internalServerError(c)
	}

	return c.JSON(supply)
}

func (h *SupplyHandler) Update(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var supply domain.Supply
	if err := c.Bind().JSON(&supply); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	supply.ID = id

	result, err := h.svc.Update(c.Context(), &supply)
	if err != nil {
		if handled, resp := dbErrResponse(c, err, "supply not found"); handled {
			return resp
		}
		return internalServerError(c)
	}

	return c.JSON(result)
}

func (h *SupplyHandler) Delete(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	if err := h.svc.Delete(c.Context(), id); err != nil {
		if handled, resp := dbErrResponse(c, err, "supply not found"); handled {
			return resp
		}
		return internalServerError(c)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *SupplyHandler) PendingPurchases(c fiber.Ctx) error {
	alerts, err := h.svc.PendingPurchases(c.Context())
	if err != nil {
		return internalServerError(c)
	}
	return c.JSON(fiber.Map{"data": alerts})
}
