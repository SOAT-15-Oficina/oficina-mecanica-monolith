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

func activeWorkshopService(id uuid.UUID) *domain.WorkshopService {
	return &domain.WorkshopService{
		ID:                   id,
		Title:                "Troca de óleo",
		Description:          "Troca completa",
		PriceCents:           8000,
		EstimatedTimeMinutes: 30,
		Active:               true,
	}
}

func inactiveWorkshopService(id uuid.UUID) *domain.WorkshopService {
	ws := activeWorkshopService(id)
	ws.Active = false
	return ws
}

func activeSupply(id uuid.UUID) *domain.Supply {
	return &domain.Supply{
		ID:         id,
		Title:      "Filtro de óleo",
		PriceCents: 3500,
		Active:     true,
	}
}

func openWO(id uuid.UUID, status domain.WorkOrderStatus) *domain.WorkOrder {
	return &domain.WorkOrder{
		ID:        id,
		Status:    status,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestAddServices_ValidInput_CreatesItems(t *testing.T) {
	// should create work order service records with snapshots from catalog
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wsID := uuid.New()
	wo := openWO(woID, domain.WorkOrderStatusReceived)
	ws := activeWorkshopService(wsID)

	created := []*domain.WorkOrderService{
		{
			ID:                                  uuid.New(),
			WorkOrderID:                         woID,
			ServiceID:                           wsID,
			ServiceTitleSnapshot:                ws.Title,
			ServicePriceCentsSnapshot:           ws.PriceCents,
			ServiceEstimatedTimeMinutesSnapshot: ws.EstimatedTimeMinutes,
			ApprovalStatus:                      domain.WorkOrderServiceApprovalPending,
			Status:                              domain.WorkOrderServiceStatusPending,
		},
	}

	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wsRepo.On("FindByID", ctx, wsID).Return(ws, nil)
	wosRepo.On("CreateBatch", ctx, mock.AnythingOfType("[]*domain.WorkOrderService")).Return(created, nil)
	statusSvc.On("TransitionTo", ctx, woID, domain.WorkOrderStatusInDiagnosis).Return(wo, nil)

	items := []AddWorkOrderServiceInput{{ServiceID: wsID}}
	result, err := svc.AddServices(ctx, woID, items)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, ws.Title, result[0].ServiceTitleSnapshot)
	assert.Equal(t, ws.PriceCents, result[0].ServicePriceCentsSnapshot)
	assert.Equal(t, domain.WorkOrderServiceApprovalPending, result[0].ApprovalStatus)
}

func TestAddServices_InvalidStatus_ReturnsError(t *testing.T) {
	// should reject when work order is already approved
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wo := openWO(woID, domain.WorkOrderStatusApproved)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)

	items := []AddWorkOrderServiceInput{{ServiceID: uuid.New()}}
	result, err := svc.AddServices(ctx, woID, items)

	assert.ErrorIs(t, err, ErrWorkOrderInvalidStatusForItems)
	assert.Nil(t, result)
	wosRepo.AssertNotCalled(t, "CreateBatch")
}

func TestAddServices_WaitingApproval_AllowsAdjustment(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wsID := uuid.New()
	wo := openWO(woID, domain.WorkOrderStatusWaitingApproval)
	ws := activeWorkshopService(wsID)
	created := []*domain.WorkOrderService{{ID: uuid.New(), WorkOrderID: woID, ServiceID: wsID}}

	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wsRepo.On("FindByID", ctx, wsID).Return(ws, nil)
	wosRepo.On("CreateBatch", ctx, mock.AnythingOfType("[]*domain.WorkOrderService")).Return(created, nil)

	result, err := svc.AddServices(ctx, woID, []AddWorkOrderServiceInput{{ServiceID: wsID}})

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	statusSvc.AssertNotCalled(t, "TransitionTo")
}

func TestAddServices_WaitingApproval_RefreshesBudget(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	budgetSvc := new(mockBudgetServiceUseCase)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc, WithBudgetRefresh(budgetSvc))
	ctx := context.Background()

	woID := uuid.New()
	wsID := uuid.New()
	wo := openWO(woID, domain.WorkOrderStatusWaitingApproval)
	ws := activeWorkshopService(wsID)
	created := []*domain.WorkOrderService{{ID: uuid.New(), WorkOrderID: woID, ServiceID: wsID}}

	woRepo.On("FindByID", ctx, woID).Return(wo, nil).Twice()
	wsRepo.On("FindByID", ctx, wsID).Return(ws, nil)
	wosRepo.On("CreateBatch", ctx, mock.AnythingOfType("[]*domain.WorkOrderService")).Return(created, nil)
	budgetSvc.On("GenerateAndSendBudget", ctx, woID, mock.Anything).Return(nil)

	result, err := svc.AddServices(ctx, woID, []AddWorkOrderServiceInput{{ServiceID: wsID}})

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	budgetSvc.AssertCalled(t, "GenerateAndSendBudget", ctx, woID, mock.Anything)
}

func TestAddServices_InactiveService_ReturnsError(t *testing.T) {
	// should reject when a service in the catalog is inactive
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wsID := uuid.New()
	wo := openWO(woID, domain.WorkOrderStatusReceived)
	ws := inactiveWorkshopService(wsID)

	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wsRepo.On("FindByID", ctx, wsID).Return(ws, nil)

	items := []AddWorkOrderServiceInput{{ServiceID: wsID}}
	result, err := svc.AddServices(ctx, woID, items)

	assert.ErrorIs(t, err, ErrWorkshopServiceInactive)
	assert.Nil(t, result)
	wosRepo.AssertNotCalled(t, "CreateBatch")
}

func TestAddServices_OptionalEstimatedTime_UsesCustomValue(t *testing.T) {
	// when estimated_time_minutes is provided, it overrides the catalog value
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wsID := uuid.New()
	wo := openWO(woID, domain.WorkOrderStatusInDiagnosis)
	ws := activeWorkshopService(wsID)
	customTime := 90

	created := []*domain.WorkOrderService{
		{
			ID:                                  uuid.New(),
			WorkOrderID:                         woID,
			ServiceID:                           wsID,
			ServiceEstimatedTimeMinutesSnapshot: customTime,
		},
	}

	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wsRepo.On("FindByID", ctx, wsID).Return(ws, nil)
	wosRepo.On("CreateBatch", ctx, mock.AnythingOfType("[]*domain.WorkOrderService")).Return(created, nil)

	items := []AddWorkOrderServiceInput{{ServiceID: wsID, EstimatedTimeMinutes: &customTime}}
	_, err := svc.AddServices(ctx, woID, items)
	assert.NoError(t, err)

	call := wosRepo.Calls[0]
	submitted := call.Arguments[1].([]*domain.WorkOrderService)
	assert.Equal(t, customTime, submitted[0].ServiceEstimatedTimeMinutesSnapshot)
}

func TestAddSupplies_ValidInput_CreatesItems(t *testing.T) {
	// should create supply records with snapshots
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	supplyID := uuid.New()

	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: woID}
	wo := openWO(woID, domain.WorkOrderStatusInDiagnosis)
	supply := activeSupply(supplyID)

	created := []*domain.WorkOrderServiceSupply{
		{
			ID:                       uuid.New(),
			WorkOrderServiceID:       wosID,
			SupplyID:                 supplyID,
			SupplyTitleSnapshot:      supply.Title,
			SupplyPriceCentsSnapshot: supply.PriceCents,
			SupplyQuantity:           2,
		},
	}

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	supplyRepo.On("FindByID", ctx, supplyID).Return(supply, nil)
	wosRepo.On("CreateSupplyBatch", ctx, mock.AnythingOfType("[]*domain.WorkOrderServiceSupply")).Return(created, nil)

	items := []AddWorkOrderSupplyInput{{SupplyID: supplyID, Quantity: 2}}
	result, err := svc.AddSupplies(ctx, woID, wosID, items)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, supply.Title, result[0].SupplyTitleSnapshot)
	assert.Equal(t, 2, result[0].SupplyQuantity)
}

func TestAddSupplies_WosNotBelongingToWorkOrder_ReturnsError(t *testing.T) {
	// must reject when wosID belongs to a different work order
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	otherWOID := uuid.New()
	wosID := uuid.New()

	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: otherWOID}
	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)

	items := []AddWorkOrderSupplyInput{{SupplyID: uuid.New(), Quantity: 1}}
	result, err := svc.AddSupplies(ctx, woID, wosID, items)

	assert.ErrorIs(t, err, ErrWorkOrderServiceOwnership)
	assert.Nil(t, result)
	wosRepo.AssertNotCalled(t, "CreateSupplyBatch")
}

func TestAddSupplies_WorkOrderApproved_ReturnsInvalidStatusError(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: woID}
	wo := openWO(woID, domain.WorkOrderStatusApproved)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)

	result, err := svc.AddSupplies(ctx, woID, wosID, []AddWorkOrderSupplyInput{{SupplyID: uuid.New(), Quantity: 1}})

	assert.ErrorIs(t, err, ErrWorkOrderInvalidStatusForItems)
	assert.Nil(t, result)
	wosRepo.AssertNotCalled(t, "CreateSupplyBatch")
}

func TestAddServices_WorkOrderRecebida_TransitionsToEmDiagnostico(t *testing.T) {
	// when WO is RECEBIDA and first service is added, status must become EM_DIAGNOSTICO
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wsID := uuid.New()
	wo := openWO(woID, domain.WorkOrderStatusReceived)
	ws := activeWorkshopService(wsID)

	created := []*domain.WorkOrderService{
		{ID: uuid.New(), WorkOrderID: woID, ServiceID: wsID},
	}

	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wsRepo.On("FindByID", ctx, wsID).Return(ws, nil)
	wosRepo.On("CreateBatch", ctx, mock.AnythingOfType("[]*domain.WorkOrderService")).Return(created, nil)
	statusSvc.On("TransitionTo", ctx, woID, domain.WorkOrderStatusInDiagnosis).Return(wo, nil)

	_, err := svc.AddServices(ctx, woID, []AddWorkOrderServiceInput{{ServiceID: wsID}})
	assert.NoError(t, err)
	statusSvc.AssertCalled(t, "TransitionTo", ctx, woID, domain.WorkOrderStatusInDiagnosis)
}

func TestAddServices_WorkOrderEmDiagnostico_DoesNotChangeStatus(t *testing.T) {
	// when WO is already EM_DIAGNOSTICO, status must not change after adding service
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wsID := uuid.New()
	wo := openWO(woID, domain.WorkOrderStatusInDiagnosis)
	ws := activeWorkshopService(wsID)

	created := []*domain.WorkOrderService{
		{ID: uuid.New(), WorkOrderID: woID, ServiceID: wsID},
	}

	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wsRepo.On("FindByID", ctx, wsID).Return(ws, nil)
	wosRepo.On("CreateBatch", ctx, mock.AnythingOfType("[]*domain.WorkOrderService")).Return(created, nil)

	_, err := svc.AddServices(ctx, woID, []AddWorkOrderServiceInput{{ServiceID: wsID}})
	assert.NoError(t, err)
	statusSvc.AssertNotCalled(t, "TransitionTo")
}

func TestRemoveService_Valid_DeletesCalled(t *testing.T) {
	// happy path: valid ownership and non-final WO status triggers DeleteByID
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: woID}
	wo := openWO(woID, domain.WorkOrderStatusInDiagnosis)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wosRepo.On("DeleteSuppliesByWorkOrderServiceID", ctx, wosID).Return(nil)
	wosRepo.On("DeleteByID", ctx, wosID).Return(nil)

	err := svc.RemoveService(ctx, woID, wosID)
	assert.NoError(t, err)
	wosRepo.AssertCalled(t, "DeleteSuppliesByWorkOrderServiceID", ctx, wosID)
	wosRepo.AssertCalled(t, "DeleteByID", ctx, wosID)
}

func TestRemoveService_WosNotFound_ReturnsNotFoundError(t *testing.T) {
	// when wosID does not exist, must propagate the application not-found error
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	wosID := uuid.New()
	wosRepo.On("FindByID", ctx, wosID).Return(nil, application.ErrNotFound)

	err := svc.RemoveService(ctx, uuid.New(), wosID)
	assert.ErrorIs(t, err, application.ErrNotFound)
	wosRepo.AssertNotCalled(t, "DeleteByID")
}

func TestRemoveService_WosWrongWorkOrder_ReturnsOwnershipError(t *testing.T) {
	// wosID exists but belongs to a different work order
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: uuid.New()}
	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)

	err := svc.RemoveService(ctx, woID, wosID)
	assert.ErrorIs(t, err, ErrWorkOrderServiceOwnership)
	wosRepo.AssertNotCalled(t, "DeleteByID")
}

func TestRemoveService_WorkOrderFinalStatus_ReturnsInvalidStatusError(t *testing.T) {
	// must reject removal when WO is in a final status
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: woID}
	wo := openWO(woID, domain.WorkOrderStatusFinished)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)

	err := svc.RemoveService(ctx, woID, wosID)
	assert.ErrorIs(t, err, ErrWorkOrderInvalidStatusForItems)
	wosRepo.AssertNotCalled(t, "DeleteByID")
}

func TestRemoveService_WorkOrderApproved_ReturnsInvalidStatusError(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: woID}
	wo := openWO(woID, domain.WorkOrderStatusApproved)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)

	err := svc.RemoveService(ctx, woID, wosID)

	assert.ErrorIs(t, err, ErrWorkOrderInvalidStatusForItems)
	wosRepo.AssertNotCalled(t, "DeleteSuppliesByWorkOrderServiceID")
	wosRepo.AssertNotCalled(t, "DeleteByID")
}

func TestRemoveSupplyFromService_Valid_DeletesRow(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	supplyID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: woID}
	wo := openWO(woID, domain.WorkOrderStatusInDiagnosis)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wosRepo.On("DeleteSupplyForWorkOrderService", ctx, wosID, supplyID).Return(nil)

	err := svc.RemoveSupplyFromService(ctx, woID, wosID, supplyID)
	assert.NoError(t, err)
	wosRepo.AssertCalled(t, "DeleteSupplyForWorkOrderService", ctx, wosID, supplyID)
}

func TestRemoveSupplyFromService_WosWrongWorkOrder_ReturnsOwnershipError(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: uuid.New()}
	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)

	err := svc.RemoveSupplyFromService(ctx, woID, wosID, uuid.New())
	assert.ErrorIs(t, err, ErrWorkOrderServiceOwnership)
	wosRepo.AssertNotCalled(t, "DeleteSupplyForWorkOrderService")
}

func TestRemoveSupplyFromService_WorkOrderFinalStatus_ReturnsInvalidStatusError(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: woID}
	wo := openWO(woID, domain.WorkOrderStatusDelivered)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)

	err := svc.RemoveSupplyFromService(ctx, woID, wosID, uuid.New())
	assert.ErrorIs(t, err, ErrWorkOrderInvalidStatusForItems)
	wosRepo.AssertNotCalled(t, "DeleteSupplyForWorkOrderService")
}

func TestRemoveSupplyFromService_WorkOrderApproved_ReturnsInvalidStatusError(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	supplyID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: woID}
	wo := openWO(woID, domain.WorkOrderStatusApproved)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)

	err := svc.RemoveSupplyFromService(ctx, woID, wosID, supplyID)

	assert.ErrorIs(t, err, ErrWorkOrderInvalidStatusForItems)
	wosRepo.AssertNotCalled(t, "DeleteSupplyForWorkOrderService")
}

func TestStartService_Success_NoShortage(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{
		ID: wosID, WorkOrderID: woID,
		ApprovalStatus: domain.WorkOrderServiceApprovalApproved,
		Status:         domain.WorkOrderServiceStatusPending,
	}
	wo := openWO(woID, domain.WorkOrderStatusInProgress)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wosRepo.On("HasSupplyShortagesForService", ctx, wosID).Return(false, nil)
	wosRepo.On("MarkServiceAsStarted", ctx, wosID, mock.AnythingOfType("time.Time")).Return(nil)

	err := svc.StartService(ctx, woID, wosID)
	assert.NoError(t, err)
}

func TestStartService_InsufficientStock(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{
		ID: wosID, WorkOrderID: woID,
		ApprovalStatus: domain.WorkOrderServiceApprovalApproved,
		Status:         domain.WorkOrderServiceStatusPending,
	}
	wo := openWO(woID, domain.WorkOrderStatusInProgress)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wosRepo.On("HasSupplyShortagesForService", ctx, wosID).Return(true, nil)

	err := svc.StartService(ctx, woID, wosID)
	assert.ErrorIs(t, err, ErrInsufficientStock)
}

func TestStartService_WrongWorkOrder(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: uuid.New()}
	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)

	err := svc.StartService(ctx, woID, wosID)
	assert.ErrorIs(t, err, ErrWorkOrderServiceOwnership)
}

func TestStartService_NotInProgress(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{
		ID: wosID, WorkOrderID: woID,
		ApprovalStatus: domain.WorkOrderServiceApprovalApproved,
		Status:         domain.WorkOrderServiceStatusPending,
	}
	wo := openWO(woID, domain.WorkOrderStatusReceived)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)

	err := svc.StartService(ctx, woID, wosID)
	assert.ErrorIs(t, err, ErrWorkOrderNotInProgress)
}

func TestStartService_NotApproved(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{
		ID: wosID, WorkOrderID: woID,
		ApprovalStatus: domain.WorkOrderServiceApprovalPending,
		Status:         domain.WorkOrderServiceStatusPending,
	}
	wo := openWO(woID, domain.WorkOrderStatusInProgress)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)

	err := svc.StartService(ctx, woID, wosID)
	assert.ErrorIs(t, err, ErrServiceNotApproved)
}

func TestStartService_NotPending(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{
		ID: wosID, WorkOrderID: woID,
		ApprovalStatus: domain.WorkOrderServiceApprovalApproved,
		Status:         domain.WorkOrderServiceStatusInProgress,
	}
	wo := openWO(woID, domain.WorkOrderStatusInProgress)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)

	err := svc.StartService(ctx, woID, wosID)
	assert.ErrorIs(t, err, ErrServiceNotPending)
}

func TestStartService_FindWosFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	wosID := uuid.New()
	wosRepo.On("FindByID", ctx, wosID).Return(nil, errors.New("db error"))

	err := svc.StartService(ctx, uuid.New(), wosID)
	assert.Error(t, err)
}

func TestStartService_CheckStockFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{
		ID: wosID, WorkOrderID: woID,
		ApprovalStatus: domain.WorkOrderServiceApprovalApproved,
		Status:         domain.WorkOrderServiceStatusPending,
	}
	wo := openWO(woID, domain.WorkOrderStatusInProgress)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wosRepo.On("HasSupplyShortagesForService", ctx, wosID).Return(false, errors.New("db error"))

	err := svc.StartService(ctx, woID, wosID)
	assert.Error(t, err)
}

func TestStartService_AddDeliveryDelayFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{
		ID: wosID, WorkOrderID: woID,
		ApprovalStatus: domain.WorkOrderServiceApprovalApproved,
		Status:         domain.WorkOrderServiceStatusPending,
	}
	wo := openWO(woID, domain.WorkOrderStatusInProgress)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wosRepo.On("HasSupplyShortagesForService", ctx, wosID).Return(true, nil)
	woRepo.On("AddDeliveryDelay", ctx, woID, 2).Return(errors.New("db error"))

	err := svc.StartService(ctx, woID, wosID)
	assert.Error(t, err)
}

func TestStartService_MarkAsStartedFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{
		ID: wosID, WorkOrderID: woID,
		ApprovalStatus: domain.WorkOrderServiceApprovalApproved,
		Status:         domain.WorkOrderServiceStatusPending,
	}
	wo := openWO(woID, domain.WorkOrderStatusInProgress)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wosRepo.On("HasSupplyShortagesForService", ctx, wosID).Return(false, nil)
	wosRepo.On("MarkServiceAsStarted", ctx, wosID, mock.AnythingOfType("time.Time")).Return(errors.New("db error"))

	err := svc.StartService(ctx, woID, wosID)
	assert.Error(t, err)
}

func TestFinalizeService_Success_NotAllFinished(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{
		ID: wosID, WorkOrderID: woID,
		Status: domain.WorkOrderServiceStatusInProgress,
	}
	wo := openWO(woID, domain.WorkOrderStatusInProgress)
	services := []domain.WorkOrderService{
		{ID: wosID, ApprovalStatus: domain.WorkOrderServiceApprovalApproved, Status: domain.WorkOrderServiceStatusFinished},
		{ID: uuid.New(), ApprovalStatus: domain.WorkOrderServiceApprovalApproved, Status: domain.WorkOrderServiceStatusPending},
	}

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wosRepo.On("MarkServiceAsFinished", ctx, wosID, mock.AnythingOfType("time.Time")).Return(nil)
	supplyRepo.On("DecrementStockForService", ctx, wosID).Return(nil)
	wosRepo.On("FindByWorkOrderID", ctx, woID).Return(services, nil)

	err := svc.FinalizeService(ctx, woID, wosID)
	assert.NoError(t, err)
	statusSvc.AssertNotCalled(t, "TransitionTo")
}

func TestFinalizeService_Success_AllFinished_AutoTransition(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{
		ID: wosID, WorkOrderID: woID,
		Status: domain.WorkOrderServiceStatusInProgress,
	}
	wo := openWO(woID, domain.WorkOrderStatusInProgress)
	services := []domain.WorkOrderService{
		{ID: wosID, ApprovalStatus: domain.WorkOrderServiceApprovalApproved, Status: domain.WorkOrderServiceStatusFinished},
	}

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wosRepo.On("MarkServiceAsFinished", ctx, wosID, mock.AnythingOfType("time.Time")).Return(nil)
	supplyRepo.On("DecrementStockForService", ctx, wosID).Return(nil)
	wosRepo.On("FindByWorkOrderID", ctx, woID).Return(services, nil)
	statusSvc.On("TransitionTo", ctx, woID, domain.WorkOrderStatusFinished).Return(wo, nil)

	err := svc.FinalizeService(ctx, woID, wosID)
	assert.NoError(t, err)
	statusSvc.AssertCalled(t, "TransitionTo", ctx, woID, domain.WorkOrderStatusFinished)
}

func TestFinalizeService_WrongWorkOrder(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	wosID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: uuid.New()}
	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)

	err := svc.FinalizeService(ctx, uuid.New(), wosID)
	assert.ErrorIs(t, err, ErrWorkOrderServiceOwnership)
}

func TestFinalizeService_NotInProgress(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{
		ID: wosID, WorkOrderID: woID,
		Status: domain.WorkOrderServiceStatusInProgress,
	}
	wo := openWO(woID, domain.WorkOrderStatusApproved)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)

	err := svc.FinalizeService(ctx, woID, wosID)
	assert.ErrorIs(t, err, ErrWorkOrderNotInProgress)
}

func TestFinalizeService_ServiceNotInProgress(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{
		ID: wosID, WorkOrderID: woID,
		Status: domain.WorkOrderServiceStatusPending,
	}
	wo := openWO(woID, domain.WorkOrderStatusInProgress)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)

	err := svc.FinalizeService(ctx, woID, wosID)
	assert.ErrorIs(t, err, ErrServiceNotInProgress)
}

func TestFinalizeService_FindWosFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	wosID := uuid.New()
	wosRepo.On("FindByID", ctx, wosID).Return(nil, errors.New("db error"))

	err := svc.FinalizeService(ctx, uuid.New(), wosID)
	assert.Error(t, err)
}

func TestFinalizeService_FindWoFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: woID, Status: domain.WorkOrderServiceStatusInProgress}
	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(nil, errors.New("db error"))

	err := svc.FinalizeService(ctx, woID, wosID)
	assert.Error(t, err)
}

func TestFinalizeService_MarkFinishedFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: woID, Status: domain.WorkOrderServiceStatusInProgress}
	wo := openWO(woID, domain.WorkOrderStatusInProgress)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wosRepo.On("MarkServiceAsFinished", ctx, wosID, mock.AnythingOfType("time.Time")).Return(errors.New("db error"))

	err := svc.FinalizeService(ctx, woID, wosID)
	assert.Error(t, err)
}

func TestFinalizeService_DecrementStockFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: woID, Status: domain.WorkOrderServiceStatusInProgress}
	wo := openWO(woID, domain.WorkOrderStatusInProgress)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wosRepo.On("MarkServiceAsFinished", ctx, wosID, mock.AnythingOfType("time.Time")).Return(nil)
	supplyRepo.On("DecrementStockForService", ctx, wosID).Return(errors.New("db error"))

	err := svc.FinalizeService(ctx, woID, wosID)
	assert.Error(t, err)
}

func TestFinalizeService_CheckCompletionFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: woID, Status: domain.WorkOrderServiceStatusInProgress}
	wo := openWO(woID, domain.WorkOrderStatusInProgress)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wosRepo.On("MarkServiceAsFinished", ctx, wosID, mock.AnythingOfType("time.Time")).Return(nil)
	supplyRepo.On("DecrementStockForService", ctx, wosID).Return(nil)
	wosRepo.On("FindByWorkOrderID", ctx, woID).Return(nil, errors.New("db error"))

	err := svc.FinalizeService(ctx, woID, wosID)
	assert.Error(t, err)
}

func TestFinalizeService_AutoTransitionFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: woID, Status: domain.WorkOrderServiceStatusInProgress}
	wo := openWO(woID, domain.WorkOrderStatusInProgress)
	services := []domain.WorkOrderService{
		{ID: wosID, ApprovalStatus: domain.WorkOrderServiceApprovalApproved, Status: domain.WorkOrderServiceStatusFinished},
	}

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wosRepo.On("MarkServiceAsFinished", ctx, wosID, mock.AnythingOfType("time.Time")).Return(nil)
	supplyRepo.On("DecrementStockForService", ctx, wosID).Return(nil)
	wosRepo.On("FindByWorkOrderID", ctx, woID).Return(services, nil)
	statusSvc.On("TransitionTo", ctx, woID, domain.WorkOrderStatusFinished).Return(nil, errors.New("db error"))

	err := svc.FinalizeService(ctx, woID, wosID)
	assert.Error(t, err)
}

func TestAddServices_WorkOrderNotFound(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	woRepo.On("FindByID", ctx, woID).Return(nil, errors.New("not found"))

	items := []AddWorkOrderServiceInput{{ServiceID: uuid.New()}}
	result, err := svc.AddServices(ctx, woID, items)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestAddServices_CreateBatchFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wsID := uuid.New()
	wo := openWO(woID, domain.WorkOrderStatusReceived)
	ws := activeWorkshopService(wsID)

	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wsRepo.On("FindByID", ctx, wsID).Return(ws, nil)
	wosRepo.On("CreateBatch", ctx, mock.Anything).Return(nil, errors.New("db error"))

	items := []AddWorkOrderServiceInput{{ServiceID: wsID}}
	result, err := svc.AddServices(ctx, woID, items)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestAddServices_ServiceNotFound(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wsID := uuid.New()
	wo := openWO(woID, domain.WorkOrderStatusReceived)

	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wsRepo.On("FindByID", ctx, wsID).Return(nil, errors.New("not found"))

	items := []AddWorkOrderServiceInput{{ServiceID: wsID}}
	result, err := svc.AddServices(ctx, woID, items)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestAddServices_TransitionFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wsID := uuid.New()
	wo := openWO(woID, domain.WorkOrderStatusReceived)
	ws := activeWorkshopService(wsID)

	created := []*domain.WorkOrderService{{ID: uuid.New(), WorkOrderID: woID, ServiceID: wsID}}

	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wsRepo.On("FindByID", ctx, wsID).Return(ws, nil)
	wosRepo.On("CreateBatch", ctx, mock.Anything).Return(created, nil)
	statusSvc.On("TransitionTo", ctx, woID, domain.WorkOrderStatusInDiagnosis).Return(nil, errors.New("db error"))

	items := []AddWorkOrderServiceInput{{ServiceID: wsID}}
	result, err := svc.AddServices(ctx, woID, items)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestAddSupplies_SupplyNotFound(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	supplyID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: woID}

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(openWO(woID, domain.WorkOrderStatusInDiagnosis), nil)
	supplyRepo.On("FindByID", ctx, supplyID).Return(nil, errors.New("not found"))

	items := []AddWorkOrderSupplyInput{{SupplyID: supplyID, Quantity: 1}}
	result, err := svc.AddSupplies(ctx, woID, wosID, items)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestAddSupplies_WosNotFound(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	wosID := uuid.New()
	wosRepo.On("FindByID", ctx, wosID).Return(nil, errors.New("not found"))

	items := []AddWorkOrderSupplyInput{{SupplyID: uuid.New(), Quantity: 1}}
	result, err := svc.AddSupplies(ctx, uuid.New(), wosID, items)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestAddSupplies_CreateBatchFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	supplyID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: woID}
	supply := activeSupply(supplyID)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(openWO(woID, domain.WorkOrderStatusInDiagnosis), nil)
	supplyRepo.On("FindByID", ctx, supplyID).Return(supply, nil)
	wosRepo.On("CreateSupplyBatch", ctx, mock.Anything).Return(nil, errors.New("db error"))

	items := []AddWorkOrderSupplyInput{{SupplyID: supplyID, Quantity: 1}}
	result, err := svc.AddSupplies(ctx, woID, wosID, items)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestRemoveService_DeleteSuppliesFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: woID}
	wo := openWO(woID, domain.WorkOrderStatusInDiagnosis)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wosRepo.On("DeleteSuppliesByWorkOrderServiceID", ctx, wosID).Return(errors.New("db error"))

	err := svc.RemoveService(ctx, woID, wosID)
	assert.Error(t, err)
}

func TestRemoveService_DeleteByIDFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: woID}
	wo := openWO(woID, domain.WorkOrderStatusInDiagnosis)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wosRepo.On("DeleteSuppliesByWorkOrderServiceID", ctx, wosID).Return(nil)
	wosRepo.On("DeleteByID", ctx, wosID).Return(errors.New("db error"))

	err := svc.RemoveService(ctx, woID, wosID)
	assert.Error(t, err)
}

func TestRemoveService_FindWorkOrderFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: woID}

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(nil, errors.New("db error"))

	err := svc.RemoveService(ctx, woID, wosID)
	assert.Error(t, err)
}

func TestRemoveSupplyFromService_FindWosFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	wosID := uuid.New()
	wosRepo.On("FindByID", ctx, wosID).Return(nil, errors.New("not found"))

	err := svc.RemoveSupplyFromService(ctx, uuid.New(), wosID, uuid.New())
	assert.Error(t, err)
}

func TestRemoveSupplyFromService_FindWoFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: woID}

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(nil, errors.New("db error"))

	err := svc.RemoveSupplyFromService(ctx, woID, wosID, uuid.New())
	assert.Error(t, err)
}

func TestRemoveSupplyFromService_DeleteFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	supplyID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: woID}
	wo := openWO(woID, domain.WorkOrderStatusInDiagnosis)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wosRepo.On("DeleteSupplyForWorkOrderService", ctx, wosID, supplyID).Return(errors.New("db error"))

	err := svc.RemoveSupplyFromService(ctx, woID, wosID, supplyID)
	assert.Error(t, err)
}

func TestFinalizeService_RejectedServicesIgnored(t *testing.T) {
	// Rejected services should not prevent auto-finalize
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{
		ID: wosID, WorkOrderID: woID,
		Status: domain.WorkOrderServiceStatusInProgress,
	}
	wo := openWO(woID, domain.WorkOrderStatusInProgress)
	services := []domain.WorkOrderService{
		{ID: wosID, ApprovalStatus: domain.WorkOrderServiceApprovalApproved, Status: domain.WorkOrderServiceStatusFinished},
		{ID: uuid.New(), ApprovalStatus: domain.WorkOrderServiceApprovalRejected, Status: domain.WorkOrderServiceStatusPending},
	}

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wosRepo.On("MarkServiceAsFinished", ctx, wosID, mock.AnythingOfType("time.Time")).Return(nil)
	supplyRepo.On("DecrementStockForService", ctx, wosID).Return(nil)
	wosRepo.On("FindByWorkOrderID", ctx, woID).Return(services, nil)
	statusSvc.On("TransitionTo", ctx, woID, domain.WorkOrderStatusFinished).Return(wo, nil)

	err := svc.FinalizeService(ctx, woID, wosID)
	assert.NoError(t, err)
	statusSvc.AssertCalled(t, "TransitionTo", ctx, woID, domain.WorkOrderStatusFinished)
}

func TestStartService_FindWorkOrderFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: woID}
	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(nil, errors.New("db error"))

	err := svc.StartService(ctx, woID, wosID)
	assert.Error(t, err)
}
