package application

import (
	"context"
	"errors"
	"time"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/google/uuid"
)

var ErrNotFound = errors.New("not found")

// ValidationError represents input that is syntactically valid but violates
// the API contract. Handlers translate it to HTTP 400 without treating it as
// an internal failure.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

func NewValidationError(message string) error {
	return &ValidationError{Message: message}
}

type WorkOrderListFilters struct {
	Status     string    `query:"status"`
	CustomerID uuid.UUID `query:"customer_id"`
	VehicleID  uuid.UUID `query:"vehicle_id"`
	FromDate   *time.Time
	ToDate     *time.Time
	Page       int
	Limit      int
}

type WorkOrderListResponse struct {
	Data       []domain.WorkOrder `json:"data"`
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
	TotalPages int                `json:"total_pages"`
}

type WorkOrderRepository interface {
	Create(ctx context.Context, workOrder *domain.WorkOrder) (*domain.WorkOrder, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.WorkOrder, error)
	FindByCode(ctx context.Context, code string) (*domain.WorkOrder, error)
	FindAll(ctx context.Context) ([]domain.WorkOrder, error)
	FindAllWithFilters(ctx context.Context, filters WorkOrderListFilters) (*WorkOrderListResponse, error)
	Update(ctx context.Context, workOrder *domain.WorkOrder) (*domain.WorkOrder, error)
	TransitionStatus(ctx context.Context, input WorkOrderStatusTransitionInput) (*domain.WorkOrder, bool, error)
}

type WorkOrderStatusTransitionInput struct {
	WorkOrderID uuid.UUID
	FromStatus  domain.WorkOrderStatus
	ToStatus    domain.WorkOrderStatus
	Now         time.Time
}

type VehicleRepository interface {
	Create(ctx context.Context, vehicle *domain.Vehicle) (*domain.Vehicle, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Vehicle, error)
	FindAll(ctx context.Context) ([]domain.Vehicle, error)
	FindAllWithFilters(ctx context.Context, filters domain.VehicleListFilters) ([]domain.Vehicle, error)
	Update(ctx context.Context, vehicle *domain.Vehicle) (*domain.Vehicle, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type CustomerRepository interface {
	Create(ctx context.Context, customer *domain.Customer) (*domain.Customer, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error)
	FindAll(ctx context.Context) ([]domain.Customer, error)
	FindAllWithFilters(ctx context.Context, filters domain.CustomerListFilters) ([]domain.Customer, error)
	Update(ctx context.Context, customer *domain.Customer) (*domain.Customer, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type WorkshopServiceRepository interface {
	Create(ctx context.Context, ws *domain.WorkshopService) (*domain.WorkshopService, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.WorkshopService, error)
	List(ctx context.Context, filters domain.WorkshopServiceListFilters) ([]domain.WorkshopService, int, error)
	Update(ctx context.Context, ws *domain.WorkshopService) (*domain.WorkshopService, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Deactivate(ctx context.Context, id uuid.UUID) (*domain.WorkshopService, error)
	ExistsByTitle(ctx context.Context, title string, excludeID *uuid.UUID) (bool, error)
	HasWorkOrderLinks(ctx context.Context, id uuid.UUID) (bool, error)
	GetAvgExecutionTime(ctx context.Context, filters domain.AvgExecutionTimeFilters) ([]domain.AvgExecutionTimeResult, error)
	SubtractSuppliesFromStock(ctx context.Context, serviceID uuid.UUID) error
}

type SupplyRepository interface {
	Create(ctx context.Context, supply *domain.Supply) (*domain.Supply, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Supply, error)
	FindAll(ctx context.Context) ([]domain.Supply, error)
	Update(ctx context.Context, supply *domain.Supply) (*domain.Supply, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DecrementStockForService(ctx context.Context, workOrderServiceID uuid.UUID) error
}

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) (*domain.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	FindByUsername(ctx context.Context, username string) (*domain.User, error)
	FindAll(ctx context.Context) ([]domain.User, error)
	Update(ctx context.Context, user *domain.User) (*domain.User, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type WorkOrderServiceRepository interface {
	Create(ctx context.Context, wos *domain.WorkOrderService) (*domain.WorkOrderService, error)
	CreateBatch(ctx context.Context, items []*domain.WorkOrderService) ([]*domain.WorkOrderService, error)
	CreateSupply(ctx context.Context, supply *domain.WorkOrderServiceSupply) (*domain.WorkOrderServiceSupply, error)
	CreateSupplyBatch(ctx context.Context, items []*domain.WorkOrderServiceSupply) ([]*domain.WorkOrderServiceSupply, error)
	DeleteSupplyForWorkOrderService(ctx context.Context, workOrderServiceID, supplyID uuid.UUID) error
	DeleteSuppliesByWorkOrderServiceID(ctx context.Context, workOrderServiceID uuid.UUID) error
	DeleteByID(ctx context.Context, id uuid.UUID) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.WorkOrderService, error)
	FindByWorkOrderID(ctx context.Context, workOrderID uuid.UUID) ([]domain.WorkOrderService, error)
	FindSupplyShortagesByWorkOrderID(ctx context.Context, workOrderID uuid.UUID) (map[uuid.UUID]bool, error)
	UpdateApprovalStatus(ctx context.Context, id uuid.UUID, status domain.WorkOrderServiceApprovalStatus) error
	UpdateApprovalStatusByWorkOrderID(ctx context.Context, workOrderID uuid.UUID, status domain.WorkOrderServiceApprovalStatus) error
	CalculateTotalForWorkOrder(ctx context.Context, workOrderID uuid.UUID) (int, error)
	CalculateApprovedTotalForWorkOrder(ctx context.Context, workOrderID uuid.UUID) (int, error)
	MarkAsStartedByWorkOrderID(ctx context.Context, workOrderID uuid.UUID, startedAt time.Time) error
	MarkAsFinishedByWorkOrderID(ctx context.Context, workOrderID uuid.UUID, finishedAt time.Time) error
	MarkServiceAsFinished(ctx context.Context, id uuid.UUID, finishedAt time.Time) error
	MarkServiceAsStarted(ctx context.Context, id uuid.UUID, startedAt time.Time) error
	HasSupplyShortagesForService(ctx context.Context, workOrderServiceID uuid.UUID) (bool, error)
	FindApprovedServicesWithShortages(ctx context.Context) ([]SupplyShortageAlert, error)
}

type SupplyShortageAlert struct {
	WorkOrderCode  string    `json:"work_order_code"`
	WorkOrderTitle string    `json:"work_order_title"`
	ServiceTitle   string    `json:"service_title"`
	SupplyTitle    string    `json:"supply_title"`
	SupplyID       uuid.UUID `json:"supply_id"`
	Required       int       `json:"required"`
	InStock        int       `json:"in_stock"`
}

type BudgetNotification struct {
	CustomerName        string
	CustomerEmail       string
	WorkOrderID         uuid.UUID
	WorkOrderCode       string
	PreviousStatusLabel string
	NewStatusLabel      string
	Amount              string
	Services            []BudgetNotificationService
	ApproveAllLink      string
	RejectAllLink       string
	BudgetLink          string
}

type BudgetNotificationService struct {
	Title       string
	Amount      string
	Estimated   string
	ApproveLink string
	RejectLink  string
}

type BudgetNotificationSender interface {
	SendBudget(ctx context.Context, notification BudgetNotification) error
}

type StatusChangeNotification struct {
	CustomerEmail       string
	CustomerName        string
	WorkOrderCode       string
	PreviousStatusLabel string
	NewStatusLabel      string
	Message             string
}

type StatusChangeSender interface {
	SendStatusChange(ctx context.Context, notification StatusChangeNotification) error
}

type PurchaseAlertNotification struct {
	To             string
	WorkOrderCode  string
	WorkOrderTitle string
	Items          []PurchaseAlertNotificationItem
}

type PurchaseAlertNotificationItem struct {
	ServiceTitle string
	SupplyTitle  string
	Required     int
	InStock      int
	ToBuy        int
}

type PurchaseAlertSender interface {
	SendPurchaseAlert(ctx context.Context, notification PurchaseAlertNotification) error
}
