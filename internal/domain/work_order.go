package domain

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

type WorkOrderStatus string

const (
	WorkOrderStatusReceived        WorkOrderStatus = "RECEBIDA"
	WorkOrderStatusInDiagnosis     WorkOrderStatus = "EM_DIAGNOSTICO"
	WorkOrderStatusWaitingApproval WorkOrderStatus = "AGUARDANDO_APROVACAO"
	WorkOrderStatusApproved        WorkOrderStatus = "APROVADO"
	WorkOrderStatusInProgress      WorkOrderStatus = "EM_EXECUCAO"
	WorkOrderStatusFinished        WorkOrderStatus = "FINALIZADA"
	WorkOrderStatusDelivered       WorkOrderStatus = "ENTREGUE"
	WorkOrderStatusCanceled        WorkOrderStatus = "CANCELADA"
)

type WorkOrder struct {
	ID                       uuid.UUID            `json:"id"`
	Code                     string               `json:"code"`
	Title                    string               `json:"title"`
	Description              *string              `json:"description,omitempty"`
	CustomerID               uuid.UUID            `json:"-"`
	VehicleID                uuid.UUID            `json:"-"`
	OpenedByUserID           uuid.UUID            `json:"opened_by_user_id"`
	AssignedTechnicianID     *uuid.UUID           `json:"assigned_technician_id,omitempty"`
	Status                   WorkOrderStatus      `json:"status"`
	TotalEstimatedPriceCents int                  `json:"total_estimated_price_cents"`
	ReceivedAt               time.Time            `json:"received_at"`
	QuoteSentAt              *time.Time           `json:"quote_sent_at,omitempty"`
	ApprovedAt               *time.Time           `json:"approved_at,omitempty"`
	StartedAt                *time.Time           `json:"started_at,omitempty"`
	FinishedAt               *time.Time           `json:"finished_at,omitempty"`
	DeliveredAt              *time.Time           `json:"delivered_at,omitempty"`
	CreatedAt                time.Time            `json:"created_at"`
	UpdatedAt                time.Time            `json:"updated_at"`
	Customer                 *WorkOrderCustomer   `json:"customer"`
	Vehicle                  *WorkOrderVehicle    `json:"vehicle"`
	AssignedTechnician       *WorkOrderTechnician `json:"assigned_technician"`
	Services                 []WorkOrderService   `json:"services"`
}

type WorkOrderCustomer struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Document string    `json:"document"`
}

type WorkOrderVehicle struct {
	ID           uuid.UUID `json:"id"`
	LicensePlate string    `json:"license_plate"`
	Brand        string    `json:"brand"`
	Model        string    `json:"model"`
	Year         int       `json:"year"`
}

type WorkOrderTechnician struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Role     UserRole  `json:"role"`
}

func IsValidWorkOrderStatus(status WorkOrderStatus) bool {
	switch status {
	case WorkOrderStatusReceived,
		WorkOrderStatusInDiagnosis,
		WorkOrderStatusWaitingApproval,
		WorkOrderStatusApproved,
		WorkOrderStatusInProgress,
		WorkOrderStatusFinished,
		WorkOrderStatusDelivered,
		WorkOrderStatusCanceled:
		return true
	default:
		return false
	}
}

var WorkOrderListingAlwaysExcludedStatuses = []WorkOrderStatus{
	WorkOrderStatusFinished,
	WorkOrderStatusDelivered,
}

var WorkOrderListingDefaultHiddenStatuses = []WorkOrderStatus{
	WorkOrderStatusCanceled,
}

var WorkOrderListingStatusPriorityOrder = []WorkOrderStatus{
	WorkOrderStatusInProgress,
	WorkOrderStatusWaitingApproval,
	WorkOrderStatusInDiagnosis,
	WorkOrderStatusReceived,
}

const WorkOrderStatusDefaultSortPriority = 99

var workOrderStatusSortPriority = func() map[WorkOrderStatus]int {
	m := make(map[WorkOrderStatus]int, len(WorkOrderListingStatusPriorityOrder))
	for i, s := range WorkOrderListingStatusPriorityOrder {
		m[s] = i + 1
	}
	return m
}()

func WorkOrderStatusSortPriorityOf(status WorkOrderStatus) int {
	if p, ok := workOrderStatusSortPriority[status]; ok {
		return p
	}
	return WorkOrderStatusDefaultSortPriority
}

func IsAlwaysExcludedFromListing(status WorkOrderStatus) bool {
	return containsStatus(WorkOrderListingAlwaysExcludedStatuses, status)
}

func IsHiddenFromDefaultListing(status WorkOrderStatus) bool {
	return containsStatus(WorkOrderListingDefaultHiddenStatuses, status)
}

func containsStatus(list []WorkOrderStatus, status WorkOrderStatus) bool {
	return slices.Contains(list, status)
}

func WorkOrderStatusLabel(status WorkOrderStatus) string {
	switch status {
	case WorkOrderStatusReceived:
		return "Recebida"
	case WorkOrderStatusInDiagnosis:
		return "Em diagnóstico"
	case WorkOrderStatusWaitingApproval:
		return "Aguardando aprovação"
	case WorkOrderStatusApproved:
		return "Aprovada"
	case WorkOrderStatusInProgress:
		return "Em execução"
	case WorkOrderStatusFinished:
		return "Finalizada"
	case WorkOrderStatusDelivered:
		return "Entregue"
	case WorkOrderStatusCanceled:
		return "Cancelada"
	default:
		return string(status)
	}
}
