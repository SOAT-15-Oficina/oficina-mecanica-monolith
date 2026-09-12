package service

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/observability"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func captureServiceLogs(t *testing.T) (context.Context, *bytes.Buffer) {
	t.Helper()
	t.Setenv("DD_ENV", "prod")

	var buf bytes.Buffer
	return observability.WithLogger(context.Background(), observability.New(&buf)), &buf
}

func lineWithEvent(t *testing.T, buf *bytes.Buffer, event string) map[string]any {
	t.Helper()

	for _, raw := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if raw == "" {
			continue
		}

		var line map[string]any
		require.NoError(t, json.Unmarshal([]byte(raw), &line))
		if line[observability.KeyEvent] == event {
			return line
		}
	}

	t.Fatalf("nenhuma linha com @event:%s em:\n%s", event, buf.String())
	return nil
}

func TestTransitionTo_EmitsStatusChangedWithTimeSpentInThePreviousStatus(t *testing.T) {
	ctx, buf := captureServiceLogs(t)

	woRepo := new(mockWorkOrderRepo)
	svc := NewWorkOrderStatusService(woRepo, nil)

	woID := uuid.New()
	startedAt := time.Now().Add(-90 * time.Minute)
	wo := &domain.WorkOrder{
		ID:        woID,
		Code:      "OS-20260910-A1B2",
		Status:    domain.WorkOrderStatusInProgress,
		StartedAt: &startedAt,
		UpdatedAt: startedAt,
	}
	updated := &domain.WorkOrder{ID: woID, Code: wo.Code, Status: domain.WorkOrderStatusFinished}

	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	woRepo.On("TransitionStatus", ctx, mock.AnythingOfType("application.WorkOrderStatusTransitionInput")).Return(updated, true, nil)

	_, err := svc.TransitionTo(ctx, woID, domain.WorkOrderStatusFinished)
	require.NoError(t, err)

	line := lineWithEvent(t, buf, observability.EventWorkOrderStatusChanged)
	assert.Equal(t, woID.String(), line[observability.KeyWorkOrderID])
	assert.Equal(t, "OS-20260910-A1B2", line[observability.KeyWorkOrderCode])
	assert.Equal(t, string(domain.WorkOrderStatusInProgress), line[observability.KeyFrom])
	assert.Equal(t, string(domain.WorkOrderStatusFinished), line[observability.KeyTo])

	assert.InDelta(t, 90*60*1000, line[observability.KeyDurationMS], 5000)
}

func TestTransitionTo_EmitsTransitionRejected(t *testing.T) {
	ctx, buf := captureServiceLogs(t)

	woRepo := new(mockWorkOrderRepo)
	svc := NewWorkOrderStatusService(woRepo, nil)

	woID := uuid.New()
	woRepo.On("FindByID", ctx, woID).Return(&domain.WorkOrder{
		ID:     woID,
		Code:   "OS-20260910-C3D4",
		Status: domain.WorkOrderStatusDelivered,
	}, nil)

	_, err := svc.TransitionTo(ctx, woID, domain.WorkOrderStatusReceived)
	require.ErrorIs(t, err, ErrInvalidStatusTransition)

	line := lineWithEvent(t, buf, observability.EventWorkOrderTransitionRejected)
	assert.Equal(t, woID.String(), line[observability.KeyWorkOrderID])
	assert.Equal(t, string(domain.WorkOrderStatusDelivered), line[observability.KeyFrom])
	assert.Equal(t, string(domain.WorkOrderStatusReceived), line[observability.KeyTo])
	assert.Equal(t, "WARN", line["level"])
}

func TestStatusEnteredAt_UsesTheStatusColumnAndFallsBackToUpdatedAt(t *testing.T) {
	receivedAt := time.Now().Add(-4 * time.Hour)
	quoteSentAt := time.Now().Add(-3 * time.Hour)
	updatedAt := time.Now().Add(-1 * time.Hour)

	wo := &domain.WorkOrder{
		ReceivedAt:  receivedAt,
		QuoteSentAt: &quoteSentAt,
		UpdatedAt:   updatedAt,
	}

	assert.Equal(t, receivedAt, statusEnteredAt(wo, domain.WorkOrderStatusReceived))
	assert.Equal(t, quoteSentAt, statusEnteredAt(wo, domain.WorkOrderStatusWaitingApproval))
	assert.Equal(t, updatedAt, statusEnteredAt(wo, domain.WorkOrderStatusInDiagnosis))

	assert.Equal(t, updatedAt, statusEnteredAt(wo, domain.WorkOrderStatusApproved))
}

func TestDurationMillis_NeverGoesNegative(t *testing.T) {
	now := time.Now()

	assert.Equal(t, float64(0), durationMillis(now, now.Add(-time.Hour)))
	assert.Equal(t, float64(0), durationMillis(time.Time{}, now))
	assert.InDelta(t, 1000, durationMillis(now, now.Add(time.Second)), 1)
}
