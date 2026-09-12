package service

import (
	"time"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
)

func statusEnteredAt(wo *domain.WorkOrder, status domain.WorkOrderStatus) time.Time {
	switch status {
	case domain.WorkOrderStatusReceived:
		return wo.ReceivedAt
	case domain.WorkOrderStatusWaitingApproval:
		return valueOr(wo.QuoteSentAt, wo.UpdatedAt)
	case domain.WorkOrderStatusApproved:
		return valueOr(wo.ApprovedAt, wo.UpdatedAt)
	case domain.WorkOrderStatusInProgress:
		return valueOr(wo.StartedAt, wo.UpdatedAt)
	case domain.WorkOrderStatusFinished:
		return valueOr(wo.FinishedAt, wo.UpdatedAt)
	case domain.WorkOrderStatusDelivered:
		return valueOr(wo.DeliveredAt, wo.UpdatedAt)
	default:
		return wo.UpdatedAt
	}
}

func durationMillis(from, to time.Time) float64 {
	if from.IsZero() || to.Before(from) {
		return 0
	}
	return float64(to.Sub(from).Microseconds()) / 1000
}

func valueOr(value *time.Time, fallback time.Time) time.Time {
	if value == nil || value.IsZero() {
		return fallback
	}
	return *value
}
