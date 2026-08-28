package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func makeWOS(woID uuid.UUID, approval domain.WorkOrderServiceApprovalStatus) domain.WorkOrderService {
	return domain.WorkOrderService{
		ID:             uuid.New(),
		WorkOrderID:    woID,
		ApprovalStatus: approval,
		Status:         domain.WorkOrderServiceStatusPending,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func makeWO(id uuid.UUID, status domain.WorkOrderStatus) *domain.WorkOrder {
	return &domain.WorkOrder{
		ID:        id,
		Status:    status,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestRejectAll_AllRejected_SetsCanceled(t *testing.T) {
	// when every service is rejected, work order must become CANCELADA (not FINALIZADA)
	wosRepo := new(mockWorkOrderServiceRepo)
	woRepo := new(mockWorkOrderRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderItemService(wosRepo, woRepo, statusSvc)
	ctx := context.Background()
	woID := uuid.New()

	services := []domain.WorkOrderService{
		makeWOS(woID, domain.WorkOrderServiceApprovalRejected),
		makeWOS(woID, domain.WorkOrderServiceApprovalRejected),
	}
	wo := makeWO(woID, domain.WorkOrderStatusCanceled)

	wosRepo.On("UpdateApprovalStatusByWorkOrderID", ctx, woID, domain.WorkOrderServiceApprovalRejected).Return(nil)
	wosRepo.On("FindByWorkOrderID", ctx, woID).Return(services, nil)
	statusSvc.On("TransitionTo", ctx, woID, domain.WorkOrderStatusCanceled).Return(wo, nil)
	wosRepo.On("CalculateApprovedTotalForWorkOrder", ctx, woID).Return(0, nil)
	woRepo.On("Update", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)

	err := svc.RejectAllByWorkOrder(ctx, woID)
	assert.NoError(t, err)
	statusSvc.AssertCalled(t, "TransitionTo", ctx, woID, domain.WorkOrderStatusCanceled)
}

func TestRejectService_LastPending_AllRejected_SetsCanceled(t *testing.T) {
	// when the last pending service is rejected and none are approved, WO becomes CANCELADA
	wosRepo := new(mockWorkOrderServiceRepo)
	woRepo := new(mockWorkOrderRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderItemService(wosRepo, woRepo, statusSvc)
	ctx := context.Background()
	woID := uuid.New()
	wosID := uuid.New()

	pending := &domain.WorkOrderService{
		ID:             wosID,
		WorkOrderID:    woID,
		ApprovalStatus: domain.WorkOrderServiceApprovalPending,
	}
	afterReject := []domain.WorkOrderService{
		makeWOS(woID, domain.WorkOrderServiceApprovalRejected),
		{ID: wosID, WorkOrderID: woID, ApprovalStatus: domain.WorkOrderServiceApprovalRejected},
	}
	wo := makeWO(woID, domain.WorkOrderStatusCanceled)

	wosRepo.On("FindByID", ctx, wosID).Return(pending, nil)
	wosRepo.On("UpdateApprovalStatus", ctx, wosID, domain.WorkOrderServiceApprovalRejected).Return(nil)
	wosRepo.On("FindByWorkOrderID", ctx, woID).Return(afterReject, nil)
	statusSvc.On("TransitionTo", ctx, woID, domain.WorkOrderStatusCanceled).Return(wo, nil)
	wosRepo.On("CalculateApprovedTotalForWorkOrder", ctx, woID).Return(0, nil)
	woRepo.On("Update", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)

	err := svc.RejectService(ctx, wosID)
	assert.NoError(t, err)
	statusSvc.AssertCalled(t, "TransitionTo", ctx, woID, domain.WorkOrderStatusCanceled)
}

func TestApproveAll_AllApproved_SetsApproved(t *testing.T) {
	// existing behaviour: when all approved, WO becomes APROVADO
	wosRepo := new(mockWorkOrderServiceRepo)
	woRepo := new(mockWorkOrderRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderItemService(wosRepo, woRepo, statusSvc)
	ctx := context.Background()
	woID := uuid.New()

	services := []domain.WorkOrderService{
		makeWOS(woID, domain.WorkOrderServiceApprovalApproved),
	}
	wo := makeWO(woID, domain.WorkOrderStatusApproved)

	wosRepo.On("UpdateApprovalStatusByWorkOrderID", ctx, woID, domain.WorkOrderServiceApprovalApproved).Return(nil)
	wosRepo.On("FindByWorkOrderID", ctx, woID).Return(services, nil)
	statusSvc.On("TransitionTo", ctx, woID, domain.WorkOrderStatusApproved).Return(wo, nil)
	wosRepo.On("CalculateApprovedTotalForWorkOrder", ctx, woID).Return(10000, nil)
	woRepo.On("Update", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)

	err := svc.ApproveAllByWorkOrder(ctx, woID)
	assert.NoError(t, err)
	statusSvc.AssertCalled(t, "TransitionTo", ctx, woID, domain.WorkOrderStatusApproved)
}

func TestApproveService_Pending_Success(t *testing.T) {
	wosRepo := new(mockWorkOrderServiceRepo)
	woRepo := new(mockWorkOrderRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderItemService(wosRepo, woRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{
		ID:             wosID,
		WorkOrderID:    woID,
		ApprovalStatus: domain.WorkOrderServiceApprovalPending,
	}
	services := []domain.WorkOrderService{
		{ID: wosID, WorkOrderID: woID, ApprovalStatus: domain.WorkOrderServiceApprovalApproved},
	}
	wo := makeWO(woID, domain.WorkOrderStatusApproved)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	wosRepo.On("UpdateApprovalStatus", ctx, wosID, domain.WorkOrderServiceApprovalApproved).Return(nil)
	wosRepo.On("FindByWorkOrderID", ctx, woID).Return(services, nil)
	statusSvc.On("TransitionTo", ctx, woID, domain.WorkOrderStatusApproved).Return(wo, nil)
	wosRepo.On("CalculateApprovedTotalForWorkOrder", ctx, woID).Return(5000, nil)
	woRepo.On("Update", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)

	err := svc.ApproveService(ctx, wosID)
	assert.NoError(t, err)
}

func TestApproveService_AlreadyApproved_Idempotent(t *testing.T) {
	wosRepo := new(mockWorkOrderServiceRepo)
	woRepo := new(mockWorkOrderRepo)
	svc := NewWorkOrderItemService(wosRepo, woRepo, new(mockStatusService))
	ctx := context.Background()

	wosID := uuid.New()
	wos := &domain.WorkOrderService{
		ID:             wosID,
		ApprovalStatus: domain.WorkOrderServiceApprovalApproved,
	}

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)

	err := svc.ApproveService(ctx, wosID)
	assert.NoError(t, err)
	wosRepo.AssertNotCalled(t, "UpdateApprovalStatus")
}

func TestApproveService_FindFails(t *testing.T) {
	wosRepo := new(mockWorkOrderServiceRepo)
	woRepo := new(mockWorkOrderRepo)
	svc := NewWorkOrderItemService(wosRepo, woRepo, new(mockStatusService))
	ctx := context.Background()
	wosID := uuid.New()

	wosRepo.On("FindByID", ctx, wosID).Return(nil, errors.New("db error"))

	err := svc.ApproveService(ctx, wosID)
	assert.Error(t, err)
}

// --- ApproveService error paths ---

func TestApproveService_UpdateStatusFails(t *testing.T) {
	wosRepo := new(mockWorkOrderServiceRepo)
	woRepo := new(mockWorkOrderRepo)
	svc := NewWorkOrderItemService(wosRepo, woRepo, new(mockStatusService))
	ctx := context.Background()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: uuid.New(), ApprovalStatus: domain.WorkOrderServiceApprovalPending}

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	wosRepo.On("UpdateApprovalStatus", ctx, wosID, domain.WorkOrderServiceApprovalApproved).Return(errors.New("db error"))

	err := svc.ApproveService(ctx, wosID)
	assert.Error(t, err)
}

// --- RejectService error paths ---

func TestRejectService_FindFails(t *testing.T) {
	wosRepo := new(mockWorkOrderServiceRepo)
	woRepo := new(mockWorkOrderRepo)
	svc := NewWorkOrderItemService(wosRepo, woRepo, new(mockStatusService))
	ctx := context.Background()
	wosID := uuid.New()

	wosRepo.On("FindByID", ctx, wosID).Return(nil, errors.New("db error"))

	err := svc.RejectService(ctx, wosID)
	assert.Error(t, err)
}

func TestRejectService_AlreadyRejected_Idempotent(t *testing.T) {
	wosRepo := new(mockWorkOrderServiceRepo)
	woRepo := new(mockWorkOrderRepo)
	svc := NewWorkOrderItemService(wosRepo, woRepo, new(mockStatusService))
	ctx := context.Background()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, ApprovalStatus: domain.WorkOrderServiceApprovalRejected}

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)

	err := svc.RejectService(ctx, wosID)
	assert.NoError(t, err)
	wosRepo.AssertNotCalled(t, "UpdateApprovalStatus")
}

func TestRejectService_UpdateStatusFails(t *testing.T) {
	wosRepo := new(mockWorkOrderServiceRepo)
	woRepo := new(mockWorkOrderRepo)
	svc := NewWorkOrderItemService(wosRepo, woRepo, new(mockStatusService))
	ctx := context.Background()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: uuid.New(), ApprovalStatus: domain.WorkOrderServiceApprovalPending}

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	wosRepo.On("UpdateApprovalStatus", ctx, wosID, domain.WorkOrderServiceApprovalRejected).Return(errors.New("db error"))

	err := svc.RejectService(ctx, wosID)
	assert.Error(t, err)
}

// --- ApproveAllByWorkOrder / RejectAllByWorkOrder error paths ---

func TestApproveAllByWorkOrder_UpdateFails(t *testing.T) {
	wosRepo := new(mockWorkOrderServiceRepo)
	woRepo := new(mockWorkOrderRepo)
	svc := NewWorkOrderItemService(wosRepo, woRepo, new(mockStatusService))
	ctx := context.Background()
	woID := uuid.New()

	wosRepo.On("UpdateApprovalStatusByWorkOrderID", ctx, woID, domain.WorkOrderServiceApprovalApproved).Return(errors.New("db error"))

	err := svc.ApproveAllByWorkOrder(ctx, woID)
	assert.Error(t, err)
}

func TestRejectAllByWorkOrder_UpdateFails(t *testing.T) {
	wosRepo := new(mockWorkOrderServiceRepo)
	woRepo := new(mockWorkOrderRepo)
	svc := NewWorkOrderItemService(wosRepo, woRepo, new(mockStatusService))
	ctx := context.Background()
	woID := uuid.New()

	wosRepo.On("UpdateApprovalStatusByWorkOrderID", ctx, woID, domain.WorkOrderServiceApprovalRejected).Return(errors.New("db error"))

	err := svc.RejectAllByWorkOrder(ctx, woID)
	assert.Error(t, err)
}

// --- evaluateWorkOrderCompletion error paths ---

func TestEvaluate_FindServicesFails(t *testing.T) {
	wosRepo := new(mockWorkOrderServiceRepo)
	woRepo := new(mockWorkOrderRepo)
	svc := NewWorkOrderItemService(wosRepo, woRepo, new(mockStatusService))
	ctx := context.Background()
	woID := uuid.New()

	wosRepo.On("UpdateApprovalStatusByWorkOrderID", ctx, woID, domain.WorkOrderServiceApprovalApproved).Return(nil)
	wosRepo.On("FindByWorkOrderID", ctx, woID).Return(nil, errors.New("db error"))

	err := svc.ApproveAllByWorkOrder(ctx, woID)
	assert.Error(t, err)
}

func TestEvaluate_StillPendingServices_ReturnsNil(t *testing.T) {
	// one service pending → evaluateWorkOrderCompletion returns nil early
	wosRepo := new(mockWorkOrderServiceRepo)
	woRepo := new(mockWorkOrderRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderItemService(wosRepo, woRepo, statusSvc)
	ctx := context.Background()
	woID := uuid.New()

	services := []domain.WorkOrderService{
		makeWOS(woID, domain.WorkOrderServiceApprovalApproved),
		makeWOS(woID, domain.WorkOrderServiceApprovalPending), // still pending
	}

	wosRepo.On("UpdateApprovalStatusByWorkOrderID", ctx, woID, domain.WorkOrderServiceApprovalApproved).Return(nil)
	wosRepo.On("FindByWorkOrderID", ctx, woID).Return(services, nil)

	err := svc.ApproveAllByWorkOrder(ctx, woID)
	assert.NoError(t, err)
	statusSvc.AssertNotCalled(t, "TransitionTo")
}

func TestEvaluate_TransitionFails(t *testing.T) {
	wosRepo := new(mockWorkOrderServiceRepo)
	woRepo := new(mockWorkOrderRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderItemService(wosRepo, woRepo, statusSvc)
	ctx := context.Background()
	woID := uuid.New()

	services := []domain.WorkOrderService{
		makeWOS(woID, domain.WorkOrderServiceApprovalApproved),
	}

	wosRepo.On("UpdateApprovalStatusByWorkOrderID", ctx, woID, domain.WorkOrderServiceApprovalApproved).Return(nil)
	wosRepo.On("FindByWorkOrderID", ctx, woID).Return(services, nil)
	statusSvc.On("TransitionTo", ctx, woID, domain.WorkOrderStatusApproved).Return(nil, errors.New("db error"))

	err := svc.ApproveAllByWorkOrder(ctx, woID)
	assert.Error(t, err)
}

func TestEvaluate_CalculateTotalFails(t *testing.T) {
	wosRepo := new(mockWorkOrderServiceRepo)
	woRepo := new(mockWorkOrderRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderItemService(wosRepo, woRepo, statusSvc)
	ctx := context.Background()
	woID := uuid.New()
	wo := makeWO(woID, domain.WorkOrderStatusApproved)

	services := []domain.WorkOrderService{
		makeWOS(woID, domain.WorkOrderServiceApprovalApproved),
	}

	wosRepo.On("UpdateApprovalStatusByWorkOrderID", ctx, woID, domain.WorkOrderServiceApprovalApproved).Return(nil)
	wosRepo.On("FindByWorkOrderID", ctx, woID).Return(services, nil)
	statusSvc.On("TransitionTo", ctx, woID, domain.WorkOrderStatusApproved).Return(wo, nil)
	wosRepo.On("CalculateApprovedTotalForWorkOrder", ctx, woID).Return(0, errors.New("db error"))

	err := svc.ApproveAllByWorkOrder(ctx, woID)
	assert.Error(t, err)
}

func TestEvaluate_UpdateWorkOrderFails(t *testing.T) {
	wosRepo := new(mockWorkOrderServiceRepo)
	woRepo := new(mockWorkOrderRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderItemService(wosRepo, woRepo, statusSvc)
	ctx := context.Background()
	woID := uuid.New()
	wo := makeWO(woID, domain.WorkOrderStatusApproved)

	services := []domain.WorkOrderService{
		makeWOS(woID, domain.WorkOrderServiceApprovalApproved),
	}

	wosRepo.On("UpdateApprovalStatusByWorkOrderID", ctx, woID, domain.WorkOrderServiceApprovalApproved).Return(nil)
	wosRepo.On("FindByWorkOrderID", ctx, woID).Return(services, nil)
	statusSvc.On("TransitionTo", ctx, woID, domain.WorkOrderStatusApproved).Return(wo, nil)
	wosRepo.On("CalculateApprovedTotalForWorkOrder", ctx, woID).Return(5000, nil)
	woRepo.On("Update", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(nil, errors.New("db error"))

	err := svc.ApproveAllByWorkOrder(ctx, woID)
	assert.Error(t, err)
}

// --- WithPurchaseAlert + sendPurchaseAlertIfNeeded ---

type mockPurchaseAlertSender struct {
	mock.Mock
}

func (m *mockPurchaseAlertSender) SendPurchaseAlert(ctx context.Context, notification application.PurchaseAlertNotification) error {
	return m.Called(ctx, notification).Error(0)
}

func TestWithPurchaseAlert_SetsFields(t *testing.T) {
	wosRepo := new(mockWorkOrderServiceRepo)
	woRepo := new(mockWorkOrderRepo)
	statusSvc := new(mockStatusService)
	alerts := new(mockPurchaseAlertSender)

	svc := NewWorkOrderItemService(wosRepo, woRepo, statusSvc, WithPurchaseAlert(alerts, "admin@test.com"))
	assert.NotNil(t, svc)
}

func TestEvaluate_WithPurchaseAlert_SendsEmail(t *testing.T) {
	wosRepo := new(mockWorkOrderServiceRepo)
	woRepo := new(mockWorkOrderRepo)
	statusSvc := new(mockStatusService)
	alertSender := new(mockPurchaseAlertSender)

	svc := NewWorkOrderItemService(wosRepo, woRepo, statusSvc, WithPurchaseAlert(alertSender, "admin@test.com"))
	ctx := context.Background()
	woID := uuid.New()

	services := []domain.WorkOrderService{
		makeWOS(woID, domain.WorkOrderServiceApprovalApproved),
	}
	wo := makeWO(woID, domain.WorkOrderStatusApproved)

	wosRepo.On("UpdateApprovalStatusByWorkOrderID", ctx, woID, domain.WorkOrderServiceApprovalApproved).Return(nil)
	wosRepo.On("FindByWorkOrderID", ctx, woID).Return(services, nil)
	statusSvc.On("TransitionTo", ctx, woID, domain.WorkOrderStatusApproved).Return(wo, nil)
	wosRepo.On("CalculateApprovedTotalForWorkOrder", ctx, woID).Return(10000, nil)
	woRepo.On("Update", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)

	shortages := map[uuid.UUID]bool{uuid.New(): true}
	wosRepo.On("FindSupplyShortagesByWorkOrderID", ctx, woID).Return(shortages, nil)
	alerts := []application.SupplyShortageAlert{
		{ServiceTitle: "Troca de óleo", SupplyTitle: "Filtro", Required: 5, InStock: 2},
	}
	wosRepo.On("FindApprovedServicesWithShortages", ctx).Return(alerts, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	alertSender.On("SendPurchaseAlert", ctx, mock.AnythingOfType("application.PurchaseAlertNotification")).Return(nil)

	err := svc.ApproveAllByWorkOrder(ctx, woID)
	assert.NoError(t, err)
	alertSender.AssertCalled(t, "SendPurchaseAlert", ctx, mock.AnythingOfType("application.PurchaseAlertNotification"))
}

func TestEvaluate_WithPurchaseAlert_NoShortages_NoEmail(t *testing.T) {
	wosRepo := new(mockWorkOrderServiceRepo)
	woRepo := new(mockWorkOrderRepo)
	statusSvc := new(mockStatusService)
	alertSender := new(mockPurchaseAlertSender)

	svc := NewWorkOrderItemService(wosRepo, woRepo, statusSvc, WithPurchaseAlert(alertSender, "admin@test.com"))
	ctx := context.Background()
	woID := uuid.New()

	services := []domain.WorkOrderService{
		makeWOS(woID, domain.WorkOrderServiceApprovalApproved),
	}
	wo := makeWO(woID, domain.WorkOrderStatusApproved)

	wosRepo.On("UpdateApprovalStatusByWorkOrderID", ctx, woID, domain.WorkOrderServiceApprovalApproved).Return(nil)
	wosRepo.On("FindByWorkOrderID", ctx, woID).Return(services, nil)
	statusSvc.On("TransitionTo", ctx, woID, domain.WorkOrderStatusApproved).Return(wo, nil)
	wosRepo.On("CalculateApprovedTotalForWorkOrder", ctx, woID).Return(10000, nil)
	woRepo.On("Update", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)
	wosRepo.On("FindSupplyShortagesByWorkOrderID", ctx, woID).Return(nil, nil)

	err := svc.ApproveAllByWorkOrder(ctx, woID)
	assert.NoError(t, err)
	alertSender.AssertNotCalled(t, "SendPurchaseAlert")
}

func TestEvaluate_WithPurchaseAlert_ShortageError_NoEmail(t *testing.T) {
	wosRepo := new(mockWorkOrderServiceRepo)
	woRepo := new(mockWorkOrderRepo)
	statusSvc := new(mockStatusService)
	alertSender := new(mockPurchaseAlertSender)

	svc := NewWorkOrderItemService(wosRepo, woRepo, statusSvc, WithPurchaseAlert(alertSender, "admin@test.com"))
	ctx := context.Background()
	woID := uuid.New()

	services := []domain.WorkOrderService{
		makeWOS(woID, domain.WorkOrderServiceApprovalApproved),
	}
	wo := makeWO(woID, domain.WorkOrderStatusApproved)

	wosRepo.On("UpdateApprovalStatusByWorkOrderID", ctx, woID, domain.WorkOrderServiceApprovalApproved).Return(nil)
	wosRepo.On("FindByWorkOrderID", ctx, woID).Return(services, nil)
	statusSvc.On("TransitionTo", ctx, woID, domain.WorkOrderStatusApproved).Return(wo, nil)
	wosRepo.On("CalculateApprovedTotalForWorkOrder", ctx, woID).Return(10000, nil)
	woRepo.On("Update", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)
	wosRepo.On("FindSupplyShortagesByWorkOrderID", ctx, woID).Return(nil, errors.New("db error"))

	err := svc.ApproveAllByWorkOrder(ctx, woID)
	assert.NoError(t, err)
	alertSender.AssertNotCalled(t, "SendPurchaseAlert")
}

func TestEvaluate_WithPurchaseAlert_AlertsFetchError_NoEmail(t *testing.T) {
	wosRepo := new(mockWorkOrderServiceRepo)
	woRepo := new(mockWorkOrderRepo)
	statusSvc := new(mockStatusService)
	alertSender := new(mockPurchaseAlertSender)

	svc := NewWorkOrderItemService(wosRepo, woRepo, statusSvc, WithPurchaseAlert(alertSender, "admin@test.com"))
	ctx := context.Background()
	woID := uuid.New()

	services := []domain.WorkOrderService{
		makeWOS(woID, domain.WorkOrderServiceApprovalApproved),
	}
	wo := makeWO(woID, domain.WorkOrderStatusApproved)

	wosRepo.On("UpdateApprovalStatusByWorkOrderID", ctx, woID, domain.WorkOrderServiceApprovalApproved).Return(nil)
	wosRepo.On("FindByWorkOrderID", ctx, woID).Return(services, nil)
	statusSvc.On("TransitionTo", ctx, woID, domain.WorkOrderStatusApproved).Return(wo, nil)
	wosRepo.On("CalculateApprovedTotalForWorkOrder", ctx, woID).Return(10000, nil)
	woRepo.On("Update", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)
	shortages := map[uuid.UUID]bool{uuid.New(): true}
	wosRepo.On("FindSupplyShortagesByWorkOrderID", ctx, woID).Return(shortages, nil)
	wosRepo.On("FindApprovedServicesWithShortages", ctx).Return(nil, errors.New("db error"))

	err := svc.ApproveAllByWorkOrder(ctx, woID)
	assert.NoError(t, err)
	alertSender.AssertNotCalled(t, "SendPurchaseAlert")
}

func TestEvaluate_WithPurchaseAlert_FindWOFails_NoEmail(t *testing.T) {
	wosRepo := new(mockWorkOrderServiceRepo)
	woRepo := new(mockWorkOrderRepo)
	statusSvc := new(mockStatusService)
	alertSender := new(mockPurchaseAlertSender)

	svc := NewWorkOrderItemService(wosRepo, woRepo, statusSvc, WithPurchaseAlert(alertSender, "admin@test.com"))
	ctx := context.Background()
	woID := uuid.New()

	services := []domain.WorkOrderService{
		makeWOS(woID, domain.WorkOrderServiceApprovalApproved),
	}
	wo := makeWO(woID, domain.WorkOrderStatusApproved)

	wosRepo.On("UpdateApprovalStatusByWorkOrderID", ctx, woID, domain.WorkOrderServiceApprovalApproved).Return(nil)
	wosRepo.On("FindByWorkOrderID", ctx, woID).Return(services, nil)
	statusSvc.On("TransitionTo", ctx, woID, domain.WorkOrderStatusApproved).Return(wo, nil)
	wosRepo.On("CalculateApprovedTotalForWorkOrder", ctx, woID).Return(10000, nil)
	woRepo.On("Update", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)
	shortages := map[uuid.UUID]bool{uuid.New(): true}
	wosRepo.On("FindSupplyShortagesByWorkOrderID", ctx, woID).Return(shortages, nil)
	alerts := []application.SupplyShortageAlert{{ServiceTitle: "S", SupplyTitle: "T", Required: 3, InStock: 1}}
	wosRepo.On("FindApprovedServicesWithShortages", ctx).Return(alerts, nil)
	woRepo.On("FindByID", ctx, woID).Return(nil, errors.New("db error"))

	err := svc.ApproveAllByWorkOrder(ctx, woID)
	assert.NoError(t, err)
	alertSender.AssertNotCalled(t, "SendPurchaseAlert")
}
