package domain

import (
	"time"

	"github.com/google/uuid"
)

type WorkOrderServiceSupply struct {
	ID                       uuid.UUID `json:"id"`
	WorkOrderServiceID       uuid.UUID `json:"work_order_service_id"`
	SupplyID                 uuid.UUID `json:"supply_id"`
	SupplyTitleSnapshot      string    `json:"supply_title_snapshot"`
	SupplyPriceCentsSnapshot int       `json:"supply_price_cents_snapshot"`
	SupplyQuantity           int       `json:"supply_quantity"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}
