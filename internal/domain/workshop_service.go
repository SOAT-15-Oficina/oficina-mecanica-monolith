package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrWorkshopServiceTitleRequired          = errors.New("title is required")
	ErrWorkshopServiceTitleLength             = errors.New("title must be between 3 and 150 characters")
	ErrWorkshopServiceDescriptionLength       = errors.New("description must be at most 500 characters")
	ErrWorkshopServicePriceMustBePositive     = errors.New("price must be greater than zero")
	ErrWorkshopServiceDurationMustBePositive  = errors.New("estimated time must be greater than zero")
)

type WorkshopService struct {
	ID                   uuid.UUID             `json:"id"`
	Title                string                `json:"title"`
	Description          string                `json:"description"`
	PriceCents           int                   `json:"price_cents"`
	EstimatedTimeMinutes int                   `json:"estimated_time_minutes"`
	Active               bool                  `json:"active"`
	CreatedAt            time.Time             `json:"created_at"`
	UpdatedAt            time.Time             `json:"updated_at"`
}

type WorkshopServiceListFilters struct {
	Active *bool
	Title  string
	Page   int
	Limit  int
}

func (s *WorkshopService) Normalize() {
	s.Title = strings.TrimSpace(s.Title)
	s.Description = strings.TrimSpace(s.Description)
}

func (s *WorkshopService) Validate() error {
	s.Normalize()

	if s.Title == "" {
		return ErrWorkshopServiceTitleRequired
	}
	if len(s.Title) < 3 || len(s.Title) > 150 {
		return ErrWorkshopServiceTitleLength
	}
	if len(s.Description) > 500 {
		return ErrWorkshopServiceDescriptionLength
	}
	if s.PriceCents <= 0 {
		return ErrWorkshopServicePriceMustBePositive
	}
	if s.EstimatedTimeMinutes <= 0 {
		return ErrWorkshopServiceDurationMustBePositive
	}

	return nil
}

type AvgExecutionTimeFilters struct {
	From         *time.Time
	To           *time.Time
	TechnicianID *uuid.UUID
}

type AvgExecutionTimeResult struct {
	ServiceID            uuid.UUID `json:"service_id"`
	Title                string    `json:"title"`
	EstimatedTimeMinutes int       `json:"estimated_time_minutes"`
	AvgRealTimeMinutes   float64   `json:"avg_real_time_minutes"`
	ExecutionCount       int       `json:"execution_count"`
}

func (s *WorkshopService) Deactivate() {
	s.Active = false
	s.UpdatedAt = time.Now().UTC()
}
