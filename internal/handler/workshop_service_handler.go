package handler

import (
	"errors"
	"math"
	"strconv"
	"time"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type WorkshopServiceHandler struct {
	svc service.WorkshopServiceManager
}

type workshopServiceRequest struct {
	Title                *string `json:"title"`
	Description          *string `json:"description"`
	PriceCents           *int    `json:"price_cents"`
	EstimatedTimeMinutes *int    `json:"estimated_time_minutes"`
	Active               *bool   `json:"active"`
}

type workshopServiceResponse struct {
	ID                   uuid.UUID `json:"id"`
	Title                string    `json:"title"`
	Description          string    `json:"description"`
	PriceCents           int       `json:"price_cents"`
	EstimatedTimeMinutes int       `json:"estimated_time_minutes"`
	Active               bool      `json:"active"`
	CreatedAt            string    `json:"created_at"`
	UpdatedAt            string    `json:"updated_at"`
}

type workshopServiceListResponse struct {
	Data       []workshopServiceResponse `json:"data"`
	Page       int                       `json:"page"`
	Limit      int                       `json:"limit"`
	Total      int                       `json:"total"`
	TotalPages int                       `json:"total_pages"`
}

type avgExecutionTimeResponse struct {
	ServiceID            uuid.UUID `json:"service_id"`
	Title                string    `json:"title"`
	EstimatedTimeMinutes int       `json:"estimated_time_minutes"`
	AvgRealTimeMinutes   float64   `json:"avg_real_time_minutes"`
	ExecutionCount       int       `json:"execution_count"`
	DifferenceMinutes    float64   `json:"difference_minutes"`
}

func NewWorkshopServiceHandler(svc service.WorkshopServiceManager) *WorkshopServiceHandler {
	return &WorkshopServiceHandler{svc: svc}
}

func (h *WorkshopServiceHandler) RegisterRoutes(app *fiber.App) {
	group := app.Group("/services")
	group.Post("/", h.Create)
	group.Get("/avg-execution-time", h.GetAvgExecutionTime)
	group.Get("/", h.GetAll)
	group.Get("/:id", h.GetByID)
	group.Put("/:id", h.Update)
	group.Delete("/:id", h.Delete)
}

func (h *WorkshopServiceHandler) Create(c fiber.Ctx) error {
	var req workshopServiceRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	ws, err := req.toCreateDomain()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	result, err := h.svc.Create(c.Context(), ws)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(toResponse(result))
}

func (h *WorkshopServiceHandler) GetAll(c fiber.Ctx) error {
	filters, err := parseListFilters(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	items, total, err := h.svc.List(c.Context(), filters)
	if err != nil {
		return internalServerError(c)
	}

	responseItems := make([]workshopServiceResponse, 0, len(items))
	for _, item := range items {
		responseItems = append(responseItems, toResponse(&item))
	}

	return c.JSON(workshopServiceListResponse{
		Data:       responseItems,
		Page:       filters.Page,
		Limit:      filters.Limit,
		Total:      total,
		TotalPages: (total + filters.Limit - 1) / filters.Limit,
	})
}

func (h *WorkshopServiceHandler) GetByID(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	ws, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.JSON(toResponse(ws))
}

func (h *WorkshopServiceHandler) Update(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var req workshopServiceRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	input, err := req.toUpdateInput()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	result, err := h.svc.Update(c.Context(), id, input)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.JSON(toResponse(result))
}

func (h *WorkshopServiceHandler) Delete(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	result, err := h.svc.Delete(c.Context(), id)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	if result.Deactivated {
		return c.JSON(fiber.Map{
			"message": "service has existing work order links and was deactivated",
			"service": toResponse(result.DeactivatedResource),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *WorkshopServiceHandler) GetAvgExecutionTime(c fiber.Ctx) error {
	filters, err := parseAvgExecutionTimeFilters(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	results, err := h.svc.GetAvgExecutionTime(c.Context(), filters)
	if err != nil {
		return internalServerError(c)
	}

	items := make([]avgExecutionTimeResponse, 0, len(results))
	for _, r := range results {
		items = append(items, avgExecutionTimeResponse{
			ServiceID:            r.ServiceID,
			Title:                r.Title,
			EstimatedTimeMinutes: r.EstimatedTimeMinutes,
			AvgRealTimeMinutes:   math.Round(r.AvgRealTimeMinutes*100) / 100,
			ExecutionCount:       r.ExecutionCount,
			DifferenceMinutes:    math.Round((r.AvgRealTimeMinutes-float64(r.EstimatedTimeMinutes))*100) / 100,
		})
	}

	return c.JSON(fiber.Map{"data": items})
}

func parseAvgExecutionTimeFilters(c fiber.Ctx) (domain.AvgExecutionTimeFilters, error) {
	var filters domain.AvgExecutionTimeFilters

	if v := c.Query("from"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return filters, errors.New("from must be in format YYYY-MM-DD")
		}
		filters.From = &t
	}

	if v := c.Query("to"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return filters, errors.New("to must be in format YYYY-MM-DD")
		}
		endOfDay := t.Add(24*time.Hour - time.Nanosecond)
		filters.To = &endOfDay
	}

	v, err := queryWithAlias(c, "technician_id", "technicianId")
	if err != nil {
		return filters, err
	}
	if v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return filters, errors.New("technician_id must be a valid UUID")
		}
		filters.TechnicianID = &id
	}
	if filters.From != nil && filters.To != nil && filters.From.After(*filters.To) {
		return filters, errors.New("from must be before or equal to to")
	}

	return filters, nil
}

func (h *WorkshopServiceHandler) handleServiceError(c fiber.Ctx, err error) error {
	if handled, resp := dbErrResponse(c, err, "service not found"); handled {
		return resp
	}
	switch {
	case errors.Is(err, service.ErrWorkshopServiceTitleAlreadyExists):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, domain.ErrWorkshopServiceTitleRequired),
		errors.Is(err, domain.ErrWorkshopServiceTitleLength),
		errors.Is(err, domain.ErrWorkshopServiceDescriptionLength),
		errors.Is(err, domain.ErrWorkshopServicePriceMustBePositive),
		errors.Is(err, domain.ErrWorkshopServiceDurationMustBePositive):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	default:
		return internalServerError(c)
	}
}

func parseListFilters(c fiber.Ctx) (domain.WorkshopServiceListFilters, error) {
	page := 1
	limit := 10

	if v := c.Query("page"); v != "" {
		val, err := strconv.Atoi(v)
		if err != nil || val <= 0 {
			return domain.WorkshopServiceListFilters{}, errors.New("page must be a positive integer")
		}
		page = val
	}

	if v := c.Query("limit"); v != "" {
		val, err := strconv.Atoi(v)
		if err != nil || val <= 0 || val > 100 {
			return domain.WorkshopServiceListFilters{}, errors.New("limit must be an integer between 1 and 100")
		}
		limit = val
	}

	var active *bool
	if v := c.Query("active"); v != "" {
		val, err := strconv.ParseBool(v)
		if err != nil {
			return domain.WorkshopServiceListFilters{}, errors.New("active must be true or false")
		}
		active = &val
	}

	title := c.Query("title")

	return domain.WorkshopServiceListFilters{
		Active: active,
		Title:  title,
		Page:   page,
		Limit:  limit,
	}, nil
}

func (r workshopServiceRequest) toCreateDomain() (*domain.WorkshopService, error) {
	if r.Title == nil || r.PriceCents == nil || r.EstimatedTimeMinutes == nil {
		return nil, errors.New("title, price_cents and estimated_time_minutes are required")
	}

	return &domain.WorkshopService{
		Title:                *r.Title,
		Description:          ptrString(r.Description),
		PriceCents:           *r.PriceCents,
		EstimatedTimeMinutes: *r.EstimatedTimeMinutes,
	}, nil
}

func (r workshopServiceRequest) toUpdateInput() (service.WorkshopServiceUpdateInput, error) {
	input := service.WorkshopServiceUpdateInput{}

	if r.Title != nil {
		input.Title = r.Title
	}
	if r.Description != nil {
		input.Description = r.Description
	}
	if r.PriceCents != nil {
		input.PriceCents = r.PriceCents
	}
	if r.EstimatedTimeMinutes != nil {
		input.EstimatedTimeMinutes = r.EstimatedTimeMinutes
	}
	if r.Active != nil {
		input.Active = r.Active
	}

	if input.Title == nil && input.Description == nil && input.PriceCents == nil && input.EstimatedTimeMinutes == nil && input.Active == nil {
		return service.WorkshopServiceUpdateInput{}, errors.New("at least one field must be provided")
	}

	return input, nil
}

func toResponse(item *domain.WorkshopService) workshopServiceResponse {
	return workshopServiceResponse{
		ID:                   item.ID,
		Title:                item.Title,
		Description:          item.Description,
		PriceCents:           item.PriceCents,
		EstimatedTimeMinutes: item.EstimatedTimeMinutes,
		Active:               item.Active,
		CreatedAt:            item.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:            item.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

func ptrString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
