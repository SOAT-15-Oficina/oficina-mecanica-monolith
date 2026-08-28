package domain

import (
	"time"

	"github.com/google/uuid"
)

type SupplyType string

type Supply struct {
	ID            uuid.UUID  `json:"id"`
	Title         string     `json:"title"`
	Type          SupplyType `json:"type"`
	PriceCents    int        `json:"price_cents"`
	StockQuantity int        `json:"stock_quantity"`
	MinimumStock  int        `json:"minimum_stock"`
	Active        bool       `json:"active"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
