package handler

import (
	"errors"
	"regexp"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

var onlyDigits = regexp.MustCompile(`\D`)

type CustomerHandler struct {
	svc service.CustomerService
}

func NewCustomerHandler(svc service.CustomerService) *CustomerHandler {
	return &CustomerHandler{svc: svc}
}

func (h *CustomerHandler) Create(c fiber.Ctx) error {
	var customer domain.Customer
	if err := c.Bind().JSON(&customer); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	result, err := h.svc.Create(c.Context(), &customer)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

func (h *CustomerHandler) GetAll(c fiber.Ctx) error {
	filters := domain.CustomerListFilters{}

	if doc := c.Query("document"); doc != "" {
		normalized := onlyDigits.ReplaceAllString(doc, "")
		filters.Document = normalized
	}

	customers, err := h.svc.GetAllWithFilters(c.Context(), filters)
	if err != nil {
		return internalServerError(c)
	}

	if customers == nil {
		customers = []domain.Customer{}
	}

	return c.JSON(fiber.Map{"data": customers})
}

func (h *CustomerHandler) GetByID(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	customer, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		if handled, resp := dbErrResponse(c, err, "customer not found"); handled {
			return resp
		}
		return internalServerError(c)
	}

	return c.JSON(customer)
}

func (h *CustomerHandler) Update(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var customer domain.Customer
	if err := c.Bind().JSON(&customer); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	customer.ID = id

	result, err := h.svc.Update(c.Context(), &customer)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.JSON(result)
}

func (h *CustomerHandler) handleServiceError(c fiber.Ctx, err error) error {
	if handled, resp := dbErrResponse(c, err, "customer not found"); handled {
		return resp
	}
	switch {
	case errors.Is(err, domain.ErrCustomerNameRequired),
		errors.Is(err, domain.ErrCustomerEmailRequired),
		errors.Is(err, domain.ErrCustomerInvalidEmailFormat),
		errors.Is(err, domain.ErrCustomerInvalidDocumentType),
		errors.Is(err, domain.ErrCustomerInvalidCPFFormat),
		errors.Is(err, domain.ErrCustomerInvalidCPFChecksum),
		errors.Is(err, domain.ErrCustomerInvalidCNPJFormat),
		errors.Is(err, domain.ErrCustomerInvalidCNPJChecksum):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	default:
		return internalServerError(c)
	}
}

func (h *CustomerHandler) Delete(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	if err := h.svc.Delete(c.Context(), id); err != nil {
		if handled, resp := mapErrorResponse(c, err, "customer not found"); handled {
			return resp
		}
		return internalServerError(c)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
