package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/google/uuid"
)

const (
	shortageExtraDays                 = 2
	minutesPerDay                     = 24 * 60
	shortageExtraEstimatedTimeMinutes = shortageExtraDays * minutesPerDay
)

type BudgetService interface {
	GenerateAndSendBudget(ctx context.Context, workOrderID uuid.UUID, previousStatus *domain.WorkOrderStatus) error
}

type budgetService struct {
	woRepo   application.WorkOrderRepository
	wosRepo  application.WorkOrderServiceRepository
	custRepo application.CustomerRepository
	notifier application.BudgetNotificationSender
	baseURL  string
}

func NewBudgetService(
	woRepo application.WorkOrderRepository,
	wosRepo application.WorkOrderServiceRepository,
	custRepo application.CustomerRepository,
	notifier application.BudgetNotificationSender,
	baseURL string,
) BudgetService {
	return &budgetService{
		woRepo:   woRepo,
		wosRepo:  wosRepo,
		custRepo: custRepo,
		notifier: notifier,
		baseURL:  baseURL,
	}
}

func (s *budgetService) GenerateAndSendBudget(ctx context.Context, workOrderID uuid.UUID, previousStatus *domain.WorkOrderStatus) error {
	services, err := s.wosRepo.FindByWorkOrderID(ctx, workOrderID)
	if err != nil {
		return fmt.Errorf("budget: find services: %w", err)
	}

	shortagesByServiceID, err := s.wosRepo.FindSupplyShortagesByWorkOrderID(ctx, workOrderID)
	if err != nil {
		return fmt.Errorf("budget: find supply shortages: %w", err)
	}

	totalCents, err := s.wosRepo.CalculateTotalForWorkOrder(ctx, workOrderID)
	if err != nil {
		return fmt.Errorf("budget: calculate total: %w", err)
	}

	wo, err := s.woRepo.FindByID(ctx, workOrderID)
	if err != nil {
		return fmt.Errorf("budget: find work order: %w", err)
	}

	customer, err := s.custRepo.FindByID(ctx, wo.CustomerID)
	if err != nil {
		return fmt.Errorf("budget: find customer: %w", err)
	}

	var serviceItems []application.BudgetNotificationService
	for _, svc := range services {
		estimatedTimeMinutes := svc.ServiceEstimatedTimeMinutesSnapshot
		if shortagesByServiceID[svc.ID] {
			estimatedTimeMinutes += shortageExtraEstimatedTimeMinutes
		}

		serviceItems = append(serviceItems, application.BudgetNotificationService{
			Title:       svc.ServiceTitleSnapshot,
			Amount:      formatCents(svc.ServicePriceCentsSnapshot),
			Estimated:   formatEstimatedTimeMinutes(estimatedTimeMinutes),
			ApproveLink: fmt.Sprintf("%s/public/approvals/services/%s/approve", s.baseURL, svc.ID),
			RejectLink:  fmt.Sprintf("%s/public/approvals/services/%s/reject", s.baseURL, svc.ID),
		})
	}

	previousStatusLabel := ""
	if previousStatus != nil {
		previousStatusLabel = domain.WorkOrderStatusLabel(*previousStatus)
	}

	notification := application.BudgetNotification{
		CustomerName:        customer.Name,
		CustomerEmail:       customer.Email,
		WorkOrderID:         workOrderID,
		WorkOrderCode:       wo.Code,
		PreviousStatusLabel: previousStatusLabel,
		NewStatusLabel:      domain.WorkOrderStatusLabel(domain.WorkOrderStatusWaitingApproval),
		Amount:              formatCents(totalCents),
		BudgetLink:          fmt.Sprintf("%s/work-orders/%s", s.baseURL, workOrderID),
		Services:            serviceItems,
		ApproveAllLink:      fmt.Sprintf("%s/public/approvals/work-orders/%s/approve-all", s.baseURL, workOrderID),
		RejectAllLink:       fmt.Sprintf("%s/public/approvals/work-orders/%s/reject-all", s.baseURL, workOrderID),
	}

	if s.notifier == nil {
		log.Printf("budget: notifier not configured for work order %s", workOrderID)
		return nil
	}

	if err := s.notifier.SendBudget(ctx, notification); err != nil {
		log.Printf("budget: send email for work order %s: %v", workOrderID, err)
		return nil
	}

	wo.TotalEstimatedPriceCents = totalCents
	now := time.Now()
	wo.QuoteSentAt = &now
	if _, err := s.woRepo.Update(ctx, wo); err != nil {
		return fmt.Errorf("budget: update work order: %w", err)
	}

	return nil
}

func formatCents(cents int) string {
	reais := cents / 100
	centavos := cents % 100
	return fmt.Sprintf("R$ %d,%02d", reais, centavos)
}

func formatEstimatedTimeMinutes(minutes int) string {
	if minutes <= 0 {
		return "0 min"
	}

	days := minutes / minutesPerDay
	remainder := minutes % minutesPerDay
	hours := remainder / 60
	mins := remainder % 60

	result := ""
	if days > 0 {
		if days == 1 {
			result = "1 dia"
		} else {
			result = fmt.Sprintf("%d dias", days)
		}
	}
	if hours > 0 {
		if result != "" {
			result += " e "
		}
		if hours == 1 {
			result += "1 hora"
		} else {
			result += fmt.Sprintf("%d horas", hours)
		}
	}
	if mins > 0 || result == "" {
		if result != "" {
			result += " e "
		}
		if mins == 1 {
			result += "1 min"
		} else {
			result += fmt.Sprintf("%d min", mins)
		}
	}

	return result
}
