package domain

import (
	"time"

	"github.com/google/uuid"
)

type WorkOrderService struct {
	ID                                  uuid.UUID                      `json:"id"`
	WorkOrderID                         uuid.UUID                      `json:"work_order_id"`
	ServiceID                           uuid.UUID                      `json:"service_id"`
	ServiceTitleSnapshot                string                         `json:"service_title_snapshot"`
	ServiceDescriptionSnapshot          *string                        `json:"service_description_snapshot,omitempty"`
	ServicePriceCentsSnapshot           int                            `json:"service_price_cents_snapshot"`
	ServiceEstimatedTimeMinutesSnapshot int                            `json:"service_estimated_time_minutes_snapshot"`
	ApprovalStatus                      WorkOrderServiceApprovalStatus `json:"approval_status"`
	Status                              WorkOrderServiceStatus         `json:"status"`
	Supplies                            []WorkOrderServiceSupply       `json:"supplies"`
	StartedAt                           *time.Time                     `json:"started_at,omitempty"`
	FinishedAt                          *time.Time                     `json:"finished_at,omitempty"`
	CreatedAt                           time.Time                      `json:"created_at"`
	UpdatedAt                           time.Time                      `json:"updated_at"`
}
