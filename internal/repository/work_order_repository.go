package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkOrderRepository interface {
	Create(ctx context.Context, workOrder *domain.WorkOrder) (*domain.WorkOrder, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.WorkOrder, error)
	FindByCode(ctx context.Context, code string) (*domain.WorkOrder, error)
	FindAll(ctx context.Context) ([]domain.WorkOrder, error)
	FindAllWithFilters(ctx context.Context, filters application.WorkOrderListFilters) (*application.WorkOrderListResponse, error)
	Update(ctx context.Context, workOrder *domain.WorkOrder) (*domain.WorkOrder, error)
	TransitionStatus(ctx context.Context, input application.WorkOrderStatusTransitionInput) (*domain.WorkOrder, bool, error)
}

type WorkOrderStatusTransitionInput = application.WorkOrderStatusTransitionInput

type workOrderRepository struct {
	db *pgxpool.Pool
}

func NewWorkOrderRepository(db *pgxpool.Pool) application.WorkOrderRepository {
	return &workOrderRepository{db: db}
}

func (r *workOrderRepository) Create(ctx context.Context, wo *domain.WorkOrder) (*domain.WorkOrder, error) {
	query := `
		INSERT INTO work_orders (
			id, code, title, description, customer_id, vehicle_id, opened_by_user_id, 
			assigned_technician_id, status, total_estimated_price_cents, received_at, 
			quote_sent_at, approved_at, started_at, finished_at, delivered_at, 
			created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		)
		RETURNING 
			id, code, title, description, customer_id, vehicle_id, opened_by_user_id, 
			assigned_technician_id, status, total_estimated_price_cents, received_at, 
			quote_sent_at, approved_at, started_at, finished_at, delivered_at, 
			created_at, updated_at`

	if wo.ID == uuid.Nil {
		wo.ID = uuid.New()
	}
	now := time.Now()
	if wo.CreatedAt.IsZero() {
		wo.CreatedAt = now
	}
	if wo.UpdatedAt.IsZero() {
		wo.UpdatedAt = now
	}

	var result domain.WorkOrder
	err := r.db.QueryRow(ctx, query,
		wo.ID, wo.Code, wo.Title, wo.Description, wo.CustomerID, wo.VehicleID, wo.OpenedByUserID,
		wo.AssignedTechnicianID, wo.Status, wo.TotalEstimatedPriceCents, wo.ReceivedAt,
		wo.QuoteSentAt, wo.ApprovedAt, wo.StartedAt, wo.FinishedAt, wo.DeliveredAt,
		wo.CreatedAt, wo.UpdatedAt).
		Scan(
			&result.ID, &result.Code, &result.Title, &result.Description, &result.CustomerID, &result.VehicleID, &result.OpenedByUserID,
			&result.AssignedTechnicianID, &result.Status, &result.TotalEstimatedPriceCents, &result.ReceivedAt,
			&result.QuoteSentAt, &result.ApprovedAt, &result.StartedAt, &result.FinishedAt, &result.DeliveredAt,
			&result.CreatedAt, &result.UpdatedAt,
		)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, application.ErrNotFound
		}
		return nil, err
	}
	return &result, nil
}

func (r *workOrderRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.WorkOrder, error) {
	query := `
		SELECT 
			wo.id, wo.code, wo.title, wo.description, wo.customer_id, wo.vehicle_id, wo.opened_by_user_id, 
			wo.assigned_technician_id, wo.status, wo.total_estimated_price_cents, wo.received_at, 
			wo.quote_sent_at, wo.approved_at, wo.started_at, wo.finished_at, wo.delivered_at, 
			wo.created_at, wo.updated_at,
			c.id, c.name, c.document,
			v.id, v.license_plate, v.brand, v.model, v.year,
			t.id, t.username, t.role
		FROM work_orders wo
		JOIN customers c ON wo.customer_id = c.id
		JOIN vehicles v ON wo.vehicle_id = v.id
		LEFT JOIN users t ON wo.assigned_technician_id = t.id
		WHERE wo.id = $1`

	var result domain.WorkOrder
	var customer domain.WorkOrderCustomer
	var vehicle domain.WorkOrderVehicle
	var customerID, vehicleID uuid.UUID
	var technicianID *uuid.UUID
	var technicianUsername *string
	var technicianRole *domain.UserRole

	err := r.db.QueryRow(ctx, query, id).
		Scan(
			&result.ID, &result.Code, &result.Title, &result.Description, &customerID, &vehicleID, &result.OpenedByUserID,
			&result.AssignedTechnicianID, &result.Status, &result.TotalEstimatedPriceCents, &result.ReceivedAt,
			&result.QuoteSentAt, &result.ApprovedAt, &result.StartedAt, &result.FinishedAt, &result.DeliveredAt,
			&result.CreatedAt, &result.UpdatedAt,
			&customer.ID, &customer.Name, &customer.Document,
			&vehicle.ID, &vehicle.LicensePlate, &vehicle.Brand, &vehicle.Model, &vehicle.Year,
			&technicianID, &technicianUsername, &technicianRole,
		)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, application.ErrNotFound
		}
		return nil, err
	}

	result.CustomerID = customerID
	result.VehicleID = vehicleID
	result.Customer = &customer
	result.Vehicle = &vehicle
	if technicianID != nil && technicianUsername != nil && technicianRole != nil {
		result.AssignedTechnician = &domain.WorkOrderTechnician{
			ID: *technicianID, Username: *technicianUsername, Role: *technicianRole,
		}
	}

	services, err := r.fetchServicesForWorkOrder(ctx, result.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, application.ErrNotFound
		}
		return nil, err
	}
	result.Services = services

	return &result, nil
}

func (r *workOrderRepository) FindByCode(ctx context.Context, code string) (*domain.WorkOrder, error) {
	query := `
		SELECT
			id, code, title, description, customer_id, vehicle_id, opened_by_user_id,
			assigned_technician_id, status, total_estimated_price_cents, received_at,
			quote_sent_at, approved_at, started_at, finished_at, delivered_at,
			created_at, updated_at
		FROM work_orders WHERE code = $1`

	var result domain.WorkOrder
	err := r.db.QueryRow(ctx, query, code).
		Scan(
			&result.ID, &result.Code, &result.Title, &result.Description, &result.CustomerID, &result.VehicleID, &result.OpenedByUserID,
			&result.AssignedTechnicianID, &result.Status, &result.TotalEstimatedPriceCents, &result.ReceivedAt,
			&result.QuoteSentAt, &result.ApprovedAt, &result.StartedAt, &result.FinishedAt, &result.DeliveredAt,
			&result.CreatedAt, &result.UpdatedAt,
		)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, application.ErrNotFound
		}
		return nil, err
	}
	return &result, nil
}

func (r *workOrderRepository) FindAll(ctx context.Context) ([]domain.WorkOrder, error) {
	query := `
		SELECT 
			id, code, title, description, customer_id, vehicle_id, opened_by_user_id, 
			assigned_technician_id, status, total_estimated_price_cents, received_at, 
			quote_sent_at, approved_at, started_at, finished_at, delivered_at, 
			created_at, updated_at
		FROM work_orders
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workOrders []domain.WorkOrder
	for rows.Next() {
		var wo domain.WorkOrder
		if err := rows.Scan(
			&wo.ID, &wo.Code, &wo.Title, &wo.Description, &wo.CustomerID, &wo.VehicleID, &wo.OpenedByUserID,
			&wo.AssignedTechnicianID, &wo.Status, &wo.TotalEstimatedPriceCents, &wo.ReceivedAt,
			&wo.QuoteSentAt, &wo.ApprovedAt, &wo.StartedAt, &wo.FinishedAt, &wo.DeliveredAt,
			&wo.CreatedAt, &wo.UpdatedAt,
		); err != nil {
			return nil, err
		}
		workOrders = append(workOrders, wo)
	}
	return workOrders, nil
}

func (r *workOrderRepository) Update(ctx context.Context, wo *domain.WorkOrder) (*domain.WorkOrder, error) {
	query := `
		UPDATE work_orders
		SET 
			code = $1, title = $2, description = $3, customer_id = $4, vehicle_id = $5, 
			opened_by_user_id = $6, assigned_technician_id = $7, status = $8, 
			total_estimated_price_cents = $9, received_at = $10, quote_sent_at = $11, 
			approved_at = $12, started_at = $13, finished_at = $14, delivered_at = $15, 
			updated_at = $16
		WHERE id = $17
		RETURNING 
			id, code, title, description, customer_id, vehicle_id, opened_by_user_id, 
			assigned_technician_id, status, total_estimated_price_cents, received_at, 
			quote_sent_at, approved_at, started_at, finished_at, delivered_at, 
			created_at, updated_at`

	wo.UpdatedAt = time.Now()

	var result domain.WorkOrder
	err := r.db.QueryRow(ctx, query,
		wo.Code, wo.Title, wo.Description, wo.CustomerID, wo.VehicleID, wo.OpenedByUserID,
		wo.AssignedTechnicianID, wo.Status, wo.TotalEstimatedPriceCents, wo.ReceivedAt,
		wo.QuoteSentAt, wo.ApprovedAt, wo.StartedAt, wo.FinishedAt, wo.DeliveredAt,
		wo.UpdatedAt, wo.ID).
		Scan(
			&result.ID, &result.Code, &result.Title, &result.Description, &result.CustomerID, &result.VehicleID, &result.OpenedByUserID,
			&result.AssignedTechnicianID, &result.Status, &result.TotalEstimatedPriceCents, &result.ReceivedAt,
			&result.QuoteSentAt, &result.ApprovedAt, &result.StartedAt, &result.FinishedAt, &result.DeliveredAt,
			&result.CreatedAt, &result.UpdatedAt,
		)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, application.ErrNotFound
		}
		return nil, err
	}
	return &result, nil
}

func workOrderStatusPriorityCase() string {
	var b strings.Builder
	b.WriteString("CASE wo.status")
	for i, status := range domain.WorkOrderListingStatusPriorityOrder {
		fmt.Fprintf(&b, " WHEN '%s' THEN %d", status, i+1)
	}
	fmt.Fprintf(&b, " ELSE %d END", domain.WorkOrderStatusDefaultSortPriority)
	return b.String()
}

func (r *workOrderRepository) TransitionStatus(ctx context.Context, input WorkOrderStatusTransitionInput) (*domain.WorkOrder, bool, error) {
	var approvedAt, startedAt, finishedAt, deliveredAt *time.Time

	switch input.ToStatus {
	case domain.WorkOrderStatusApproved:
		approvedAt = &input.Now
	case domain.WorkOrderStatusInProgress:
		startedAt = &input.Now
	case domain.WorkOrderStatusFinished:
		finishedAt = &input.Now
	case domain.WorkOrderStatusDelivered:
		deliveredAt = &input.Now
	}

	query := `
		UPDATE work_orders
		SET
			status = $1,
			approved_at = COALESCE($2, approved_at),
			started_at = COALESCE($3, started_at),
			finished_at = COALESCE($4, finished_at),
			delivered_at = COALESCE($5, delivered_at),
			updated_at = $6
		WHERE id = $7 AND status = $8
		RETURNING
			id, code, title, description, customer_id, vehicle_id, opened_by_user_id,
			assigned_technician_id, status, total_estimated_price_cents, received_at,
			quote_sent_at, approved_at, started_at, finished_at, delivered_at,
			created_at, updated_at`

	var result domain.WorkOrder
	err := r.db.QueryRow(ctx, query,
		input.ToStatus,
		approvedAt,
		startedAt,
		finishedAt,
		deliveredAt,
		input.Now,
		input.WorkOrderID,
		input.FromStatus,
	).Scan(
		&result.ID, &result.Code, &result.Title, &result.Description, &result.CustomerID, &result.VehicleID, &result.OpenedByUserID,
		&result.AssignedTechnicianID, &result.Status, &result.TotalEstimatedPriceCents, &result.ReceivedAt,
		&result.QuoteSentAt, &result.ApprovedAt, &result.StartedAt, &result.FinishedAt, &result.DeliveredAt,
		&result.CreatedAt, &result.UpdatedAt,
	)
	if err == nil {
		return &result, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, err
	}

	current, findErr := r.FindByID(ctx, input.WorkOrderID)
	if findErr != nil {
		return nil, false, findErr
	}
	if current.Status == input.ToStatus {
		return current, false, nil
	}

	return nil, false, fmt.Errorf("%w: %s -> %s", ErrInvalidWorkOrderStatusTransition, current.Status, input.ToStatus)
}

func (r *workOrderRepository) FindAllWithFilters(ctx context.Context, filters application.WorkOrderListFilters) (*application.WorkOrderListResponse, error) {
	whereConditions := []string{}
	args := []interface{}{}
	argIndex := 1

	for _, excluded := range domain.WorkOrderListingAlwaysExcludedStatuses {
		whereConditions = append(whereConditions, fmt.Sprintf("wo.status <> $%d", argIndex))
		args = append(args, string(excluded))
		argIndex++
	}

	if filters.Status != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("wo.status = $%d", argIndex))
		args = append(args, filters.Status)
		argIndex++
	} else {
		for _, hidden := range domain.WorkOrderListingDefaultHiddenStatuses {
			whereConditions = append(whereConditions, fmt.Sprintf("wo.status <> $%d", argIndex))
			args = append(args, string(hidden))
			argIndex++
		}
	}

	if filters.CustomerID != uuid.Nil {
		whereConditions = append(whereConditions, fmt.Sprintf("wo.customer_id = $%d", argIndex))
		args = append(args, filters.CustomerID)
		argIndex++
	}

	if filters.VehicleID != uuid.Nil {
		whereConditions = append(whereConditions, fmt.Sprintf("wo.vehicle_id = $%d", argIndex))
		args = append(args, filters.VehicleID)
		argIndex++
	}

	if filters.FromDate != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("wo.received_at >= $%d", argIndex))
		args = append(args, filters.FromDate)
		argIndex++
	}

	if filters.ToDate != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("wo.received_at <= $%d", argIndex))
		args = append(args, filters.ToDate)
		argIndex++
	}

	whereClause := "1 = 1"
	if len(whereConditions) > 0 {
		whereClause = strings.Join(whereConditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM work_orders wo WHERE %s", whereClause)
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	offset := (filters.Page - 1) * filters.Limit
	args = append(args, filters.Limit, offset)

	query := fmt.Sprintf(`
		SELECT
			wo.id, wo.code, wo.title, wo.description, wo.customer_id, wo.vehicle_id, wo.opened_by_user_id,
			wo.assigned_technician_id, wo.status, wo.total_estimated_price_cents, wo.received_at,
			wo.quote_sent_at, wo.approved_at, wo.started_at, wo.finished_at, wo.delivered_at,
			wo.created_at, wo.updated_at,
			c.id, c.name, c.document,
			v.id, v.license_plate, v.brand, v.model, v.year,
			t.id, t.username, t.role
		FROM work_orders wo
		JOIN customers c ON wo.customer_id = c.id
		JOIN vehicles v ON wo.vehicle_id = v.id
		LEFT JOIN users t ON wo.assigned_technician_id = t.id
		WHERE %s
		ORDER BY %s,
			wo.received_at ASC
		LIMIT $%d OFFSET $%d`, whereClause, workOrderStatusPriorityCase(), len(args)-1, len(args))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workOrders []domain.WorkOrder
	for rows.Next() {
		var wo domain.WorkOrder
		var customer domain.WorkOrderCustomer
		var vehicle domain.WorkOrderVehicle
		var customerID, vehicleID uuid.UUID
		var technicianID *uuid.UUID
		var technicianUsername *string
		var technicianRole *domain.UserRole

		if err := rows.Scan(
			&wo.ID, &wo.Code, &wo.Title, &wo.Description, &customerID, &vehicleID, &wo.OpenedByUserID,
			&wo.AssignedTechnicianID, &wo.Status, &wo.TotalEstimatedPriceCents, &wo.ReceivedAt,
			&wo.QuoteSentAt, &wo.ApprovedAt, &wo.StartedAt, &wo.FinishedAt, &wo.DeliveredAt,
			&wo.CreatedAt, &wo.UpdatedAt,
			&customer.ID, &customer.Name, &customer.Document,
			&vehicle.ID, &vehicle.LicensePlate, &vehicle.Brand, &vehicle.Model, &vehicle.Year,
			&technicianID, &technicianUsername, &technicianRole,
		); err != nil {
			return nil, err
		}

		wo.Customer = &customer
		wo.Vehicle = &vehicle
		if technicianID != nil && technicianUsername != nil && technicianRole != nil {
			wo.AssignedTechnician = &domain.WorkOrderTechnician{
				ID: *technicianID, Username: *technicianUsername, Role: *technicianRole,
			}
		}
		wo.CustomerID = customerID
		wo.VehicleID = vehicleID

		workOrders = append(workOrders, wo)
	}

	totalPages := (total + filters.Limit - 1) / filters.Limit

	return &application.WorkOrderListResponse{
		Data:       workOrders,
		Total:      total,
		Page:       filters.Page,
		Limit:      filters.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *workOrderRepository) fetchServicesForWorkOrder(ctx context.Context, workOrderID uuid.UUID) ([]domain.WorkOrderService, error) {
	query := `SELECT id, work_order_id, service_id, service_title_snapshot, service_description_snapshot, service_price_cents_snapshot, service_estimated_time_minutes_snapshot, approval_status, status, started_at, finished_at, created_at, updated_at FROM work_order_services WHERE work_order_id = $1`
	rows, err := r.db.Query(ctx, query, workOrderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	services := make([]domain.WorkOrderService, 0)
	for rows.Next() {
		var svc domain.WorkOrderService
		if err := rows.Scan(
			&svc.ID, &svc.WorkOrderID, &svc.ServiceID,
			&svc.ServiceTitleSnapshot, &svc.ServiceDescriptionSnapshot,
			&svc.ServicePriceCentsSnapshot, &svc.ServiceEstimatedTimeMinutesSnapshot,
			&svc.ApprovalStatus, &svc.Status, &svc.StartedAt, &svc.FinishedAt, &svc.CreatedAt, &svc.UpdatedAt,
		); err != nil {
			return nil, err
		}

		supplies, err := r.fetchSuppliesForService(ctx, svc.ID)
		if err != nil {
			return nil, err
		}
		svc.Supplies = supplies
		services = append(services, svc)
	}
	return services, nil
}

func (r *workOrderRepository) fetchSuppliesForService(ctx context.Context, serviceID uuid.UUID) ([]domain.WorkOrderServiceSupply, error) {
	query := `SELECT id, work_order_service_id, supply_id, supply_title_snapshot, supply_price_cents_snapshot, supply_quantity, created_at, updated_at FROM work_order_service_supplies WHERE work_order_service_id = $1`
	rows, err := r.db.Query(ctx, query, serviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	supplies := make([]domain.WorkOrderServiceSupply, 0)
	for rows.Next() {
		var sup domain.WorkOrderServiceSupply
		if err := rows.Scan(
			&sup.ID, &sup.WorkOrderServiceID, &sup.SupplyID,
			&sup.SupplyTitleSnapshot, &sup.SupplyPriceCentsSnapshot,
			&sup.SupplyQuantity, &sup.CreatedAt, &sup.UpdatedAt,
		); err != nil {
			return nil, err
		}
		supplies = append(supplies, sup)
	}
	return supplies, nil
}
