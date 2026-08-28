package service

import (
	"context"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/google/uuid"
)

type VehicleService interface {
	Create(ctx context.Context, vehicle *domain.Vehicle) (*domain.Vehicle, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Vehicle, error)
	GetAll(ctx context.Context) ([]domain.Vehicle, error)
	GetAllWithFilters(ctx context.Context, filters domain.VehicleListFilters) ([]domain.Vehicle, error)
	Update(ctx context.Context, vehicle *domain.Vehicle) (*domain.Vehicle, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type vehicleService struct {
	repo application.VehicleRepository
}

func NewVehicleService(repo application.VehicleRepository) VehicleService {
	return &vehicleService{repo: repo}
}

func (s *vehicleService) Create(ctx context.Context, vehicle *domain.Vehicle) (*domain.Vehicle, error) {
	if err := vehicle.Validate(); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, vehicle)
}

func (s *vehicleService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Vehicle, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *vehicleService) GetAll(ctx context.Context) ([]domain.Vehicle, error) {
	return s.repo.FindAll(ctx)
}

func (s *vehicleService) GetAllWithFilters(ctx context.Context, filters domain.VehicleListFilters) ([]domain.Vehicle, error) {
	return s.repo.FindAllWithFilters(ctx, filters)
}

func (s *vehicleService) Update(ctx context.Context, vehicle *domain.Vehicle) (*domain.Vehicle, error) {
	if err := vehicle.Validate(); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, vehicle)
}

func (s *vehicleService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
