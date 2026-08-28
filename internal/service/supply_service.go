package service

import (
	"context"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/google/uuid"
)

type SupplyService interface {
	Create(ctx context.Context, supply *domain.Supply) (*domain.Supply, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Supply, error)
	GetAll(ctx context.Context) ([]domain.Supply, error)
	Update(ctx context.Context, supply *domain.Supply) (*domain.Supply, error)
	Delete(ctx context.Context, id uuid.UUID) error
	PendingPurchases(ctx context.Context) ([]application.SupplyShortageAlert, error)
}

type supplyService struct {
	repo    application.SupplyRepository
	wosRepo application.WorkOrderServiceRepository
}

func NewSupplyService(repo application.SupplyRepository, wosRepo application.WorkOrderServiceRepository) SupplyService {
	return &supplyService{repo: repo, wosRepo: wosRepo}
}

func (s *supplyService) Create(ctx context.Context, supply *domain.Supply) (*domain.Supply, error) {
	return s.repo.Create(ctx, supply)
}

func (s *supplyService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Supply, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *supplyService) GetAll(ctx context.Context) ([]domain.Supply, error) {
	return s.repo.FindAll(ctx)
}

func (s *supplyService) Update(ctx context.Context, supply *domain.Supply) (*domain.Supply, error) {
	return s.repo.Update(ctx, supply)
}

func (s *supplyService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *supplyService) PendingPurchases(ctx context.Context) ([]application.SupplyShortageAlert, error) {
	alerts, err := s.wosRepo.FindApprovedServicesWithShortages(ctx)
	if err != nil {
		return nil, err
	}
	if alerts == nil {
		alerts = []application.SupplyShortageAlert{}
	}
	return alerts, nil
}
