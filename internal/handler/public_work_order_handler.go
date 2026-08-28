package handler

import (
	"errors"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/service"
	"github.com/gofiber/fiber/v3"
)

type PublicWorkOrderHandler struct {
	svc service.PublicWorkOrderService
}

func NewPublicWorkOrderHandler(svc service.PublicWorkOrderService) *PublicWorkOrderHandler {
	return &PublicWorkOrderHandler{svc: svc}
}

func (h *PublicWorkOrderHandler) GetByCode(c fiber.Ctx) error {
	code := c.Params("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "code is required"})
	}

	document := c.Query("document")
	if document == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "document query parameter is required"})
	}

	view, err := h.svc.GetPublicStatus(c.Context(), code, document)
	if err != nil {
		if errors.Is(err, service.ErrWorkOrderNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "work order not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}

	return c.JSON(view)
}
