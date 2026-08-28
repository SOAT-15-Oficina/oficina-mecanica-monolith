package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkOrderServiceRepository = application.WorkOrderServiceRepository

type SupplyShortageAlert = application.SupplyShortageAlert

type workOrderServiceRepository struct {
	db *pgxpool.Pool
}

func NewWorkOrderServiceRepository(db *pgxpool.Pool) application.WorkOrderServiceRepository {
	return &workOrderServiceRepository{db: db}
}

func (r *workOrderServiceRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.WorkOrderService, error) {
	query := `
		SELECT id, work_order_id, service_id, service_title_snapshot, service_description_snapshot,
			service_price_cents_snapshot, service_estimated_time_minutes_snapshot,
			approval_status, status, started_at, finished_at, created_at, updated_at
		FROM work_order_services WHERE id = $1`

	var wos domain.WorkOrderService
	err := r.db.QueryRow(ctx, query, id).Scan(
		&wos.ID, &wos.WorkOrderID, &wos.ServiceID, &wos.ServiceTitleSnapshot, &wos.ServiceDescriptionSnapshot,
		&wos.ServicePriceCentsSnapshot, &wos.ServiceEstimatedTimeMinutesSnapshot,
		&wos.ApprovalStatus, &wos.Status, &wos.StartedAt, &wos.FinishedAt, &wos.CreatedAt, &wos.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, application.ErrNotFound
		}
		return nil, err
	}
	return &wos, nil
}

func (r *workOrderServiceRepository) FindByWorkOrderID(ctx context.Context, workOrderID uuid.UUID) ([]domain.WorkOrderService, error) {
	query := `
		SELECT id, work_order_id, service_id, service_title_snapshot, service_description_snapshot,
			service_price_cents_snapshot, service_estimated_time_minutes_snapshot,
			approval_status, status, started_at, finished_at, created_at, updated_at
		FROM work_order_services WHERE work_order_id = $1`

	rows, err := r.db.Query(ctx, query, workOrderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []domain.WorkOrderService
	for rows.Next() {
		var wos domain.WorkOrderService
		if err := rows.Scan(
			&wos.ID, &wos.WorkOrderID, &wos.ServiceID, &wos.ServiceTitleSnapshot, &wos.ServiceDescriptionSnapshot,
			&wos.ServicePriceCentsSnapshot, &wos.ServiceEstimatedTimeMinutesSnapshot,
			&wos.ApprovalStatus, &wos.Status, &wos.StartedAt, &wos.FinishedAt, &wos.CreatedAt, &wos.UpdatedAt,
		); err != nil {
			return nil, err
		}
		services = append(services, wos)
	}
	return services, nil
}

func (r *workOrderServiceRepository) UpdateApprovalStatus(ctx context.Context, id uuid.UUID, status domain.WorkOrderServiceApprovalStatus) error {
	query := `UPDATE work_order_services SET approval_status = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.Exec(ctx, query, status, time.Now(), id)
	return err
}

func (r *workOrderServiceRepository) UpdateApprovalStatusByWorkOrderID(ctx context.Context, workOrderID uuid.UUID, status domain.WorkOrderServiceApprovalStatus) error {
	query := `UPDATE work_order_services SET approval_status = $1, updated_at = $2 WHERE work_order_id = $3 AND approval_status = $4`
	_, err := r.db.Exec(ctx, query, status, time.Now(), workOrderID, domain.WorkOrderServiceApprovalPending)
	return err
}

func (r *workOrderServiceRepository) CalculateTotalForWorkOrder(ctx context.Context, workOrderID uuid.UUID) (int, error) {
	query := `
		SELECT COALESCE(SUM(wos.service_price_cents_snapshot), 0) +
			COALESCE((
				SELECT SUM(woss.supply_price_cents_snapshot * woss.supply_quantity)
				FROM work_order_service_supplies woss
				JOIN work_order_services wos2 ON wos2.id = woss.work_order_service_id
				WHERE wos2.work_order_id = $1
			), 0)
		FROM work_order_services wos
		WHERE wos.work_order_id = $1`

	var total int
	err := r.db.QueryRow(ctx, query, workOrderID).Scan(&total)
	return total, err
}

func (r *workOrderServiceRepository) CalculateApprovedTotalForWorkOrder(ctx context.Context, workOrderID uuid.UUID) (int, error) {
	query := `
		SELECT COALESCE(SUM(wos.service_price_cents_snapshot), 0) +
			COALESCE((
				SELECT SUM(woss.supply_price_cents_snapshot * woss.supply_quantity)
				FROM work_order_service_supplies woss
				JOIN work_order_services wos2 ON wos2.id = woss.work_order_service_id
				WHERE wos2.work_order_id = $1 AND wos2.approval_status = 'APROVADO'
			), 0)
		FROM work_order_services wos
		WHERE wos.work_order_id = $1 AND wos.approval_status = 'APROVADO'`

	var total int
	err := r.db.QueryRow(ctx, query, workOrderID).Scan(&total)
	return total, err
}

func (r *workOrderServiceRepository) DeleteSuppliesByWorkOrderServiceID(ctx context.Context, workOrderServiceID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM work_order_service_supplies WHERE work_order_service_id = $1`,
		workOrderServiceID,
	)
	return err
}

func (r *workOrderServiceRepository) DeleteSupplyForWorkOrderService(ctx context.Context, workOrderServiceID, supplyID uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM work_order_service_supplies WHERE work_order_service_id = $1 AND id = $2`,
		workOrderServiceID, supplyID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return nil
}

func (r *workOrderServiceRepository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM work_order_services WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return nil
}

func (r *workOrderServiceRepository) Create(ctx context.Context, wos *domain.WorkOrderService) (*domain.WorkOrderService, error) {
	items, err := r.CreateBatch(ctx, []*domain.WorkOrderService{wos})
	if err != nil {
		return nil, err
	}
	return items[0], nil
}

func (r *workOrderServiceRepository) CreateBatch(ctx context.Context, items []*domain.WorkOrderService) ([]*domain.WorkOrderService, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("create batch: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO work_order_services (
			id, work_order_id, service_id,
			service_title_snapshot, service_description_snapshot,
			service_price_cents_snapshot, service_estimated_time_minutes_snapshot,
			approval_status, status, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id, work_order_id, service_id,
			service_title_snapshot, service_description_snapshot,
			service_price_cents_snapshot, service_estimated_time_minutes_snapshot,
			approval_status, status, started_at, finished_at, created_at, updated_at`

	now := time.Now()
	results := make([]*domain.WorkOrderService, 0, len(items))
	for _, item := range items {
		if item.ID == uuid.Nil {
			item.ID = uuid.New()
		}

		var out domain.WorkOrderService
		if err := tx.QueryRow(ctx, query,
			item.ID, item.WorkOrderID, item.ServiceID,
			item.ServiceTitleSnapshot, item.ServiceDescriptionSnapshot,
			item.ServicePriceCentsSnapshot, item.ServiceEstimatedTimeMinutesSnapshot,
			item.ApprovalStatus, item.Status, now, now,
		).Scan(
			&out.ID, &out.WorkOrderID, &out.ServiceID,
			&out.ServiceTitleSnapshot, &out.ServiceDescriptionSnapshot,
			&out.ServicePriceCentsSnapshot, &out.ServiceEstimatedTimeMinutesSnapshot,
			&out.ApprovalStatus, &out.Status, &out.StartedAt, &out.FinishedAt,
			&out.CreatedAt, &out.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("create batch: insert: %w", err)
		}
		results = append(results, &out)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("create batch: commit: %w", err)
	}
	return results, nil
}

func (r *workOrderServiceRepository) CreateSupply(ctx context.Context, supply *domain.WorkOrderServiceSupply) (*domain.WorkOrderServiceSupply, error) {
	items, err := r.CreateSupplyBatch(ctx, []*domain.WorkOrderServiceSupply{supply})
	if err != nil {
		return nil, err
	}
	return items[0], nil
}

func (r *workOrderServiceRepository) MarkAsStartedByWorkOrderID(ctx context.Context, workOrderID uuid.UUID, startedAt time.Time) error {
	query := `
		UPDATE work_order_services
		SET status = $1, started_at = $2, updated_at = $2
		WHERE work_order_id = $3
		  AND approval_status = $4
		  AND started_at IS NULL`
	_, err := r.db.Exec(ctx, query,
		domain.WorkOrderServiceStatusInProgress, startedAt,
		workOrderID, domain.WorkOrderServiceApprovalApproved,
	)
	return err
}

func (r *workOrderServiceRepository) MarkAsFinishedByWorkOrderID(ctx context.Context, workOrderID uuid.UUID, finishedAt time.Time) error {
	query := `
		UPDATE work_order_services
		SET status = $1, finished_at = $2, updated_at = $2
		WHERE work_order_id = $3
		  AND status = $4
		  AND finished_at IS NULL`
	_, err := r.db.Exec(ctx, query,
		domain.WorkOrderServiceStatusFinished, finishedAt,
		workOrderID, domain.WorkOrderServiceStatusInProgress,
	)
	return err
}

func (r *workOrderServiceRepository) MarkServiceAsStarted(ctx context.Context, id uuid.UUID, startedAt time.Time) error {
	query := `
		UPDATE work_order_services
		SET status = $1, started_at = $2, updated_at = $2
		WHERE id = $3
		  AND status = $4
		  AND started_at IS NULL`
	_, err := r.db.Exec(ctx, query,
		domain.WorkOrderServiceStatusInProgress, startedAt,
		id, domain.WorkOrderServiceStatusPending,
	)
	return err
}

func (r *workOrderServiceRepository) MarkServiceAsFinished(ctx context.Context, id uuid.UUID, finishedAt time.Time) error {
	query := `
		UPDATE work_order_services
		SET status = $1, finished_at = $2, updated_at = $2
		WHERE id = $3
		  AND status = $4
		  AND finished_at IS NULL`
	_, err := r.db.Exec(ctx, query,
		domain.WorkOrderServiceStatusFinished, finishedAt,
		id, domain.WorkOrderServiceStatusInProgress,
	)
	return err
}

func (r *workOrderServiceRepository) FindSupplyShortagesByWorkOrderID(ctx context.Context, workOrderID uuid.UUID) (map[uuid.UUID]bool, error) {
	query := `
		SELECT DISTINCT wos.id
		FROM work_order_services wos
		JOIN work_order_service_supplies woss ON woss.work_order_service_id = wos.id
		JOIN supplies s ON s.id = woss.supply_id
		WHERE wos.work_order_id = $1
		  AND woss.supply_quantity > s.stock_quantity`

	rows, err := r.db.Query(ctx, query, workOrderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	shortages := make(map[uuid.UUID]bool)
	for rows.Next() {
		var serviceID uuid.UUID
		if err := rows.Scan(&serviceID); err != nil {
			return nil, err
		}
		shortages[serviceID] = true
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return shortages, nil
}

func (r *workOrderServiceRepository) CreateSupplyBatch(ctx context.Context, items []*domain.WorkOrderServiceSupply) ([]*domain.WorkOrderServiceSupply, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("create supply batch: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO work_order_service_supplies (
			id, work_order_service_id, supply_id,
			supply_title_snapshot, supply_price_cents_snapshot, supply_quantity,
			created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, work_order_service_id, supply_id,
			supply_title_snapshot, supply_price_cents_snapshot, supply_quantity,
			created_at, updated_at`

	now := time.Now()
	results := make([]*domain.WorkOrderServiceSupply, 0, len(items))
	for _, item := range items {
		if item.ID == uuid.Nil {
			item.ID = uuid.New()
		}

		var out domain.WorkOrderServiceSupply
		if err := tx.QueryRow(ctx, query,
			item.ID, item.WorkOrderServiceID, item.SupplyID,
			item.SupplyTitleSnapshot, item.SupplyPriceCentsSnapshot, item.SupplyQuantity,
			now, now,
		).Scan(
			&out.ID, &out.WorkOrderServiceID, &out.SupplyID,
			&out.SupplyTitleSnapshot, &out.SupplyPriceCentsSnapshot, &out.SupplyQuantity,
			&out.CreatedAt, &out.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("create supply batch: insert: %w", err)
		}
		results = append(results, &out)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("create supply batch: commit: %w", err)
	}
	return results, nil
}

func (r *workOrderServiceRepository) HasSupplyShortagesForService(ctx context.Context, workOrderServiceID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM work_order_service_supplies woss
			JOIN supplies s ON s.id = woss.supply_id
			WHERE woss.work_order_service_id = $1
			  AND woss.supply_quantity > s.stock_quantity
		)`
	var has bool
	err := r.db.QueryRow(ctx, query, workOrderServiceID).Scan(&has)
	return has, err
}

func (r *workOrderServiceRepository) FindApprovedServicesWithShortages(ctx context.Context) ([]SupplyShortageAlert, error) {
	query := `
		SELECT wo.code, wo.title, wos.service_title_snapshot, s.title, s.id,
		       woss.supply_quantity, s.stock_quantity
		FROM work_order_services wos
		JOIN work_orders wo ON wo.id = wos.work_order_id
		JOIN work_order_service_supplies woss ON woss.work_order_service_id = wos.id
		JOIN supplies s ON s.id = woss.supply_id
		WHERE wos.approval_status = 'APROVADO'
		  AND wos.status = 'PENDENTE'
		  AND woss.supply_quantity > s.stock_quantity
		ORDER BY wo.code, wos.service_title_snapshot`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []SupplyShortageAlert
	for rows.Next() {
		var a SupplyShortageAlert
		if err := rows.Scan(&a.WorkOrderCode, &a.WorkOrderTitle, &a.ServiceTitle, &a.SupplyTitle, &a.SupplyID, &a.Required, &a.InStock); err != nil {
			return nil, err
		}
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}
