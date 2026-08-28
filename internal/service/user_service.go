package service

import (
	"context"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/google/uuid"
)

// Cadastro e emissao de credencial vivem no oficina-mecanica-serverless. Aqui
// restam a leitura e a manutencao administrativa de usuarios, que o dominio de
// ordens de servico consome via GetByUsername.
type UserService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	GetAll(ctx context.Context) ([]domain.User, error)
	Update(ctx context.Context, id uuid.UUID, username string, role domain.UserRole) (*domain.User, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type userService struct {
	repo application.UserRepository
}

func NewUserService(repo application.UserRepository) UserService {
	return &userService{repo: repo}
}

func (s *userService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *userService) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	return s.repo.FindByUsername(ctx, username)
}

func (s *userService) GetAll(ctx context.Context) ([]domain.User, error) {
	return s.repo.FindAll(ctx)
}

func (s *userService) Update(ctx context.Context, id uuid.UUID, username string, role domain.UserRole) (*domain.User, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if username != "" {
		existing.Username = username
	}
	if role != "" {
		if role != domain.UserRoleAdmin && role != domain.UserRoleEmployee {
			return nil, application.NewValidationError("invalid role: must be 'admin' or 'employee'")
		}
		existing.Role = role
	}

	return s.repo.Update(ctx, existing)
}

func (s *userService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
