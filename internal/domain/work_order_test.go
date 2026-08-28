package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAlwaysExcludedFromListing(t *testing.T) {
	tests := []struct {
		name   string
		status WorkOrderStatus
		want   bool
	}{
		{"RECEBIDA is not always excluded", WorkOrderStatusReceived, false},
		{"EM_DIAGNOSTICO is not always excluded", WorkOrderStatusInDiagnosis, false},
		{"AGUARDANDO_APROVACAO is not always excluded", WorkOrderStatusWaitingApproval, false},
		{"APROVADO is not always excluded", WorkOrderStatusApproved, false},
		{"EM_EXECUCAO is not always excluded", WorkOrderStatusInProgress, false},
		{"CANCELADA is not always excluded (only hidden by default)", WorkOrderStatusCanceled, false},
		{"FINALIZADA is always excluded", WorkOrderStatusFinished, true},
		{"ENTREGUE is always excluded", WorkOrderStatusDelivered, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsAlwaysExcludedFromListing(tt.status))
		})
	}
}

func TestIsHiddenFromDefaultListing(t *testing.T) {
	tests := []struct {
		name   string
		status WorkOrderStatus
		want   bool
	}{
		{"CANCELADA is hidden from default listing", WorkOrderStatusCanceled, true},
		{"FINALIZADA is not in the default-hidden set (it is always excluded)", WorkOrderStatusFinished, false},
		{"ENTREGUE is not in the default-hidden set (it is always excluded)", WorkOrderStatusDelivered, false},
		{"RECEBIDA is shown by default", WorkOrderStatusReceived, false},
		{"APROVADO is shown by default", WorkOrderStatusApproved, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsHiddenFromDefaultListing(tt.status))
		})
	}
}

func TestWorkOrderStatusSortPriorityOf(t *testing.T) {
	tests := []struct {
		name   string
		status WorkOrderStatus
		want   int
	}{
		{"EM_EXECUCAO has highest priority (1)", WorkOrderStatusInProgress, 1},
		{"AGUARDANDO_APROVACAO is priority 2", WorkOrderStatusWaitingApproval, 2},
		{"EM_DIAGNOSTICO is priority 3", WorkOrderStatusInDiagnosis, 3},
		{"RECEBIDA is priority 4", WorkOrderStatusReceived, 4},
		{"APROVADO falls to default (not a prioritized status)", WorkOrderStatusApproved, WorkOrderStatusDefaultSortPriority},
		{"CANCELADA falls to default", WorkOrderStatusCanceled, WorkOrderStatusDefaultSortPriority},
		{"FINALIZADA falls to default", WorkOrderStatusFinished, WorkOrderStatusDefaultSortPriority},
		{"ENTREGUE falls to default", WorkOrderStatusDelivered, WorkOrderStatusDefaultSortPriority},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, WorkOrderStatusSortPriorityOf(tt.status))
		})
	}
}

func TestWorkOrderListingExclusionSets(t *testing.T) {
	assert.ElementsMatch(t,
		[]WorkOrderStatus{WorkOrderStatusFinished, WorkOrderStatusDelivered},
		WorkOrderListingAlwaysExcludedStatuses,
	)
	assert.ElementsMatch(t,
		[]WorkOrderStatus{WorkOrderStatusCanceled},
		WorkOrderListingDefaultHiddenStatuses,
	)

	assert.Equal(t,
		[]WorkOrderStatus{
			WorkOrderStatusInProgress,
			WorkOrderStatusWaitingApproval,
			WorkOrderStatusInDiagnosis,
			WorkOrderStatusReceived,
		},
		WorkOrderListingStatusPriorityOrder,
	)
}
