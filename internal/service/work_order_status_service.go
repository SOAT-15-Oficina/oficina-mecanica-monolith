package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/observability"
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
	logger := observability.FromContext(ctx)

	wo, err := s.woRepo.FindByID(ctx, workOrderID)
	if err != nil {
		return nil, fmt.Errorf("transition: find work order: %w", err)
	}

	previousStatus := wo.Status
	if previousStatus == newStatus {
		return wo, nil
	}

	// UM DOS DOIS GATILHOS DO ALERTA EXIGIDO PELA FASE.
	//
	// `datadog_monitor.work_order_processing_failure` dispara com
	// `@event:work_order.transition_rejected` OU com qualquer ERROR que carregue
	// `@work_order_id`. E por isso que a transicao recusada e um evento nomeado,
	// e nao um `log.Printf` com o texto do erro: a query do alerta pergunta pelo
	// nome, e nome nao muda quando alguem reescreve a mensagem.
	if !s.IsValidTransition(previousStatus, newStatus) {
		logger.LogAttrs(ctx, slog.LevelWarn, "invalid work order status transition",
			observability.Event(observability.EventWorkOrderTransitionRejected),
			slog.String(observability.KeyWorkOrderID, workOrderID.String()),
			slog.String(observability.KeyWorkOrderCode, wo.Code),
			slog.String(observability.KeyFrom, string(previousStatus)),
			slog.String(observability.KeyTo, string(newStatus)))

		return nil, fmt.Errorf("%w: %s -> %s", ErrInvalidStatusTransition, previousStatus, newStatus)
	}

	now := time.Now()

	updated, transitioned, err := s.woRepo.TransitionStatus(ctx, application.WorkOrderStatusTransitionInput{
		WorkOrderID: workOrderID,
		FromStatus:  previousStatus,
		ToStatus:    newStatus,
		Now:         now,
	})
	if err != nil {
		logger.LogAttrs(ctx, slog.LevelError, "work order status transition failed",
			observability.Integration(observability.IntegrationRDS),
			slog.String(observability.KeyWorkOrderID, workOrderID.String()),
			slog.String(observability.KeyFrom, string(previousStatus)),
			slog.String(observability.KeyTo, string(newStatus)),
			observability.Err(err))

		return nil, fmt.Errorf("transition: update work order: %w", err)
	}

	if transitioned {
		// `duration_ms` aqui e o tempo passado no status ANTERIOR -- nao a
		// duracao desta chamada. Agrupado por `@to`, e exatamente o painel de
		// "tempo medio por status" que a fase pede.
		logger.LogAttrs(ctx, slog.LevelInfo, "work order status changed",
			observability.Event(observability.EventWorkOrderStatusChanged),
			slog.String(observability.KeyWorkOrderID, workOrderID.String()),
			slog.String(observability.KeyWorkOrderCode, updated.Code),
			slog.String(observability.KeyFrom, string(previousStatus)),
			slog.String(observability.KeyTo, string(newStatus)),
			slog.Float64(observability.KeyDurationMS, durationMillis(statusEnteredAt(wo, previousStatus), now)))

		if s.notifier != nil {
			s.notifier.NotifyTransition(ctx, updated, previousStatus)
		}
	}

	return updated, nil
}
