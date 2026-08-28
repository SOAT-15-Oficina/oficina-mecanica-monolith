package handler

import (
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type UpdateUserRequest struct {
	Username string          `json:"username"`
	Role     domain.UserRole `json:"role"`
}

type UserHandler struct {
	svc service.UserService
}

func NewUserHandler(svc service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) GetAll(c fiber.Ctx) error {
	users, err := h.svc.GetAll(c.Context())
	if err != nil {
		return internalServerError(c)
	}
	if users == nil {
		users = []domain.User{}
	}
	return c.JSON(fiber.Map{"data": users})
}

func (h *UserHandler) GetByID(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	user, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		if handled, resp := mapErrorResponse(c, err, "user not found"); handled {
			return resp
		}
		return internalServerError(c)
	}
	return c.JSON(user)
}

func (h *UserHandler) Update(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var body UpdateUserRequest
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	user, err := h.svc.Update(c.Context(), id, body.Username, body.Role)
	if err != nil {
		if handled, resp := mapErrorResponse(c, err, "user not found"); handled {
			return resp
		}
		return internalServerError(c)
	}
	return c.JSON(user)
}

func (h *UserHandler) Delete(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	if err := h.svc.Delete(c.Context(), id); err != nil {
		if handled, resp := mapErrorResponse(c, err, "user not found"); handled {
			return resp
		}
		return internalServerError(c)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
