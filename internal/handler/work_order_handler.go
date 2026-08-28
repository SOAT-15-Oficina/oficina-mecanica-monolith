package handler

import (
	"errors"
	"strconv"
	"time"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/auth"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type WorkOrderHandler struct {
	svc         service.WorkOrderService
	creationSvc service.WorkOrderCreationService
	statusSvc   service.WorkOrderStatusService
	userSvc     service.UserService
}

func NewWorkOrderHandler(svc service.WorkOrderService, creationSvc service.WorkOrderCreationService, statusSvc service.WorkOrderStatusService, userSvc service.UserService) *WorkOrderHandler {
	return &WorkOrderHandler{svc: svc, creationSvc: creationSvc, statusSvc: statusSvc, userSvc: userSvc}
}

type createWorkOrderRequest struct {
	Title                string     `json:"title"`
	Description          *string    `json:"description,omitempty"`
	CustomerID           uuid.UUID  `json:"customer_id"`
	VehicleID            uuid.UUID  `json:"vehicle_id"`
	AssignedTechnicianID *uuid.UUID `json:"assigned_technician_id,omitempty"`
}

type updateWorkOrderRequest struct {
	Title                string                 `json:"title"`
	Description          *string                `json:"description,omitempty"`
	Status               domain.WorkOrderStatus `json:"status"`
	AssignedTechnicianID *uuid.UUID             `json:"assigned_technician_id,omitempty"`
}

type addServiceRequest struct {
	ServiceID            uuid.UUID `json:"service_id"`
	EstimatedTimeMinutes *int      `json:"estimated_time_minutes,omitempty"`
}

type addSupplyRequest struct {
	SupplyID uuid.UUID `json:"supply_id"`
	Quantity int       `json:"quantity"`
}

func (h *WorkOrderHandler) Create(c fiber.Ctx) error {
	var req createWorkOrderRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	claims, ok := c.Locals("token").(*auth.AppClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing token claims"})
	}

	user, err := h.userSvc.GetByUsername(c.Context(), claims.User)
	if err != nil {
		return internalServerError(c)
	}

	workOrder := domain.WorkOrder{
		Title:                req.Title,
		Description:          req.Description,
		CustomerID:           req.CustomerID,
		VehicleID:            req.VehicleID,
		OpenedByUserID:       user.ID,
		AssignedTechnicianID: req.AssignedTechnicianID,
	}

	result, err := h.svc.Create(c.Context(), &workOrder)
	if err != nil {
		if handled, resp := mapErrorResponse(c, err, "work order not found"); handled {
			return resp
		}
		return internalServerError(c)
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

func (h *WorkOrderHandler) GetAll(c fiber.Ctx) error {
	filters := application.WorkOrderListFilters{
		Page:  1,
		Limit: 10,
	}

	if page := c.Query("page"); page != "" {
		p, err := strconv.Atoi(page)
		if err != nil || p <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "page must be a positive integer"})
		}
		filters.Page = p
	}
	if limit := c.Query("limit"); limit != "" {
		l, err := strconv.Atoi(limit)
		if err != nil || l <= 0 || l > 100 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "limit must be an integer between 1 and 100"})
		}
		filters.Limit = l
	}

	if status := c.Query("status"); status != "" {
		if !domain.IsValidWorkOrderStatus(domain.WorkOrderStatus(status)) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid status"})
		}
		filters.Status = status
	}

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

	vehicleID, err := queryWithAlias(c, "vehicle_id", "vehicleId")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if vehicleID != "" {
		id, err := uuid.Parse(vehicleID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "vehicle_id must be a valid UUID"})
		}
		filters.VehicleID = id
	}

	if from := c.Query("from"); from != "" {
		t, err := time.Parse("2006-01-02", from)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "from must be in format YYYY-MM-DD"})
		}
		filters.FromDate = &t
	}

	if to := c.Query("to"); to != "" {
		t, err := time.Parse("2006-01-02", to)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "to must be in format YYYY-MM-DD"})
		}
		t = t.Add(24*time.Hour - 1*time.Nanosecond)
		filters.ToDate = &t
	}
	if filters.FromDate != nil && filters.ToDate != nil && filters.FromDate.After(*filters.ToDate) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "from must be before or equal to to"})
	}

	result, err := h.svc.GetAllWithFilters(c.Context(), filters)
	if err != nil {
		return internalServerError(c)
	}

	if result == nil || result.Data == nil {
		result = &application.WorkOrderListResponse{
			Data:       []domain.WorkOrder{},
			Total:      0,
			Page:       filters.Page,
			Limit:      filters.Limit,
			TotalPages: 0,
		}
	}

	if result.Data == nil {
		result.Data = []domain.WorkOrder{}
	}
	return c.JSON(result)
}

func (h *WorkOrderHandler) GetByID(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	workOrder, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		if handled, resp := mapErrorResponse(c, err, "work order not found"); handled {
			return resp
		}
		return internalServerError(c)
	}

	return c.JSON(workOrder)
}

func (h *WorkOrderHandler) Update(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var req updateWorkOrderRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if req.Status != "" {
		if !domain.IsValidWorkOrderStatus(req.Status) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid status"})
		}
		_, err := h.statusSvc.TransitionTo(c.Context(), id, req.Status)
		if err != nil {
			if handled, resp := mapErrorResponse(c, err, "work order not found"); handled {
				return resp
			}
			return internalServerError(c)
		}
	}

	workOrder := domain.WorkOrder{
		ID:                   id,
		Title:                req.Title,
		Description:          req.Description,
		AssignedTechnicianID: req.AssignedTechnicianID,
	}

	workOrder.Status = ""

	result, err := h.svc.Update(c.Context(), &workOrder)
	if err != nil {
		if handled, resp := mapErrorResponse(c, err, "work order not found"); handled {
			return resp
		}
		return internalServerError(c)
	}

	return c.JSON(result)
}

func (h *WorkOrderHandler) AddServices(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid work order id"})
	}

	var reqs []addServiceRequest
	if err := c.Bind().JSON(&reqs); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if len(reqs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "at least one service is required"})
	}

	items := make([]service.AddWorkOrderServiceInput, len(reqs))
	for i, r := range reqs {
		if r.ServiceID == uuid.Nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "service_id is required"})
		}
		if r.EstimatedTimeMinutes != nil && *r.EstimatedTimeMinutes <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "estimated_time_minutes must be greater than zero"})
		}
		items[i] = service.AddWorkOrderServiceInput{
			ServiceID:            r.ServiceID,
			EstimatedTimeMinutes: r.EstimatedTimeMinutes,
		}
	}

	result, err := h.creationSvc.AddServices(c.Context(), id, items)
	if err != nil {
		if handled, resp := mapErrorResponse(c, err, "resource not found"); handled {
			return resp
		}
		return internalServerError(c)
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

func (h *WorkOrderHandler) RemoveService(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid work order id"})
	}

	wosID, err := uuid.Parse(c.Params("wosId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid work order service id"})
	}

	if err := h.creationSvc.RemoveService(c.Context(), id, wosID); err != nil {
		if handled, resp := mapErrorResponse(c, err, "resource not found"); handled {
			return resp
		}
		return internalServerError(c)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *WorkOrderHandler) AddSupplies(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid work order id"})
	}

	wosID, err := uuid.Parse(c.Params("wosId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid work order service id"})
	}

	var reqs []addSupplyRequest
	if err := c.Bind().JSON(&reqs); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if len(reqs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "at least one supply is required"})
	}

	items := make([]service.AddWorkOrderSupplyInput, len(reqs))
	for i, r := range reqs {
		if r.SupplyID == uuid.Nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "supply_id is required"})
		}
		if r.Quantity <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "quantity must be greater than zero"})
		}
		items[i] = service.AddWorkOrderSupplyInput{
			SupplyID: r.SupplyID,
			Quantity: r.Quantity,
		}
	}

	result, err := h.creationSvc.AddSupplies(c.Context(), id, wosID, items)
	if err != nil {
		if errors.Is(err, service.ErrWorkOrderServiceOwnership) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrWorkOrderInvalidStatusForItems) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
		}
		if handled, resp := mapErrorResponse(c, err, "resource not found"); handled {
			return resp
		}
		return internalServerError(c)
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

func (h *WorkOrderHandler) StartService(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid work order id"})
	}
	wosID, err := uuid.Parse(c.Params("wosId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid work order service id"})
	}
	if err := h.creationSvc.StartService(c.Context(), id, wosID); err != nil {
		if errors.Is(err, service.ErrWorkOrderServiceOwnership) ||
			errors.Is(err, service.ErrWorkOrderNotInProgress) ||
			errors.Is(err, service.ErrServiceNotPending) ||
			errors.Is(err, service.ErrServiceNotApproved) ||
			errors.Is(err, service.ErrInsufficientStock) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
		}
		if handled, resp := mapErrorResponse(c, err, "resource not found"); handled {
			return resp
		}
		return internalServerError(c)
	}
	return c.JSON(fiber.Map{"message": "Servico iniciado com sucesso"})
}

func (h *WorkOrderHandler) FinalizeService(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid work order id"})
	}
	wosID, err := uuid.Parse(c.Params("wosId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid work order service id"})
	}
	if err := h.creationSvc.FinalizeService(c.Context(), id, wosID); err != nil {
		if errors.Is(err, service.ErrWorkOrderServiceOwnership) ||
			errors.Is(err, service.ErrWorkOrderNotInProgress) ||
			errors.Is(err, service.ErrServiceNotInProgress) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
		}
		if handled, resp := mapErrorResponse(c, err, "resource not found"); handled {
			return resp
		}
		return internalServerError(c)
	}
	return c.JSON(fiber.Map{"message": "Servico finalizado com sucesso"})
}

func (h *WorkOrderHandler) RemoveSupplyFromService(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid work order id"})
	}

	wosID, err := uuid.Parse(c.Params("wosId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid work order service id"})
	}

	supplyID, err := uuid.Parse(c.Params("supplyId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid supply id"})
	}

	if err := h.creationSvc.RemoveSupplyFromService(c.Context(), id, wosID, supplyID); err != nil {
		if handled, resp := mapErrorResponse(c, err, "resource not found"); handled {
			return resp
		}
		return internalServerError(c)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
