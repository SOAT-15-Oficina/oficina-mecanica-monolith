package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/google/uuid"
)

var ErrInvalidStatusTransition = errors.New("invalid status transition")

var allowedTransitions = map[domain.WorkOrderStatus][]domain.WorkOrderStatus{
	domain.WorkOrderStatusReceived:        {domain.WorkOrderStatusInDiagnosis, domain.WorkOrderStatusCanceled},
	domain.WorkOrderStatusInDiagnosis:     {domain.WorkOrderStatusWaitingApproval, domain.WorkOrderStatusCanceled},
	domain.WorkOrderStatusWaitingApproval: {domain.WorkOrderStatusApproved, domain.WorkOrderStatusCanceled},
	domain.WorkOrderStatusApproved:        {domain.WorkOrderStatusInProgress},
	domain.WorkOrderStatusInProgress:      {domain.WorkOrderStatusFinished},
	domain.WorkOrderStatusFinished:        {domain.WorkOrderStatusDelivered},
	domain.WorkOrderStatusDelivered:       {},
	domain.WorkOrderStatusCanceled:        {},
}

type WorkOrderStatusService interface {
	TransitionTo(ctx context.Context, workOrderID uuid.UUID, newStatus domain.WorkOrderStatus) (*domain.WorkOrder, error)
	IsValidTransition(from, to domain.WorkOrderStatus) bool
}

type workOrderStatusService struct {
	woRepo   application.WorkOrderRepository
	notifier WorkOrderStatusNotifier
}

func NewWorkOrderStatusService(
	woRepo application.WorkOrderRepository,
	notifier WorkOrderStatusNotifier,
) WorkOrderStatusService {
	return &workOrderStatusService{
		woRepo:   woRepo,
		notifier: notifier,
	}
}

func (s *workOrderStatusService) IsValidTransition(from, to domain.WorkOrderStatus) bool {
	allowed, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	for _, status := range allowed {
		if status == to {
			return true
		}
	}
	return false
}

func (s *workOrderStatusService) TransitionTo(ctx context.Context, workOrderID uuid.UUID, newStatus domain.WorkOrderStatus) (*domain.WorkOrder, error) {
	wo, err := s.woRepo.FindByID(ctx, workOrderID)
	if err != nil {
		return nil, fmt.Errorf("transition: find work order: %w", err)
	}

	previousStatus := wo.Status
	if previousStatus == newStatus {
		return wo, nil
	}

	if !s.IsValidTransition(previousStatus, newStatus) {
		return nil, fmt.Errorf("%w: %s -> %s", ErrInvalidStatusTransition, previousStatus, newStatus)
	}

	updated, transitioned, err := s.woRepo.TransitionStatus(ctx, application.WorkOrderStatusTransitionInput{
		WorkOrderID: workOrderID,
		FromStatus:  previousStatus,
		ToStatus:    newStatus,
		Now:         time.Now(),
	})
	if err != nil {
		return nil, fmt.Errorf("transition: update work order: %w", err)
	}

	if transitioned && s.notifier != nil {
		s.notifier.NotifyTransition(ctx, updated, previousStatus)
	}

	return updated, nil
}
