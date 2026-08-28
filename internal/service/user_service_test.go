package service

import (
	"context"
	"errors"
	"testing"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockUserRepo mocks repository.UserRepository
type mockUserRepo struct {
	mock.Mock
}

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserRepo) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserRepo) FindAll(ctx context.Context) ([]domain.User, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.User), args.Error(1)
}

func (m *mockUserRepo) Update(ctx context.Context, user *domain.User) (*domain.User, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// --- Update ---

func TestUserUpdate_InvalidRole(t *testing.T) {
	repo := new(mockUserRepo)
	svc := NewUserService(repo)
	ctx := context.Background()
	id := uuid.New()
	existing := &domain.User{ID: id, Username: "alice", Role: domain.UserRoleAdmin}

	repo.On("FindByID", ctx, id).Return(existing, nil)

	result, err := svc.Update(ctx, id, "alice", "hacker")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestUserUpdate_Success(t *testing.T) {
	repo := new(mockUserRepo)
	svc := NewUserService(repo)
	ctx := context.Background()
	id := uuid.New()
	existing := &domain.User{ID: id, Username: "alice", Role: domain.UserRoleAdmin}
	updated := &domain.User{ID: id, Username: "alice2", Role: domain.UserRoleEmployee}

	repo.On("FindByID", ctx, id).Return(existing, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.User")).Return(updated, nil)

	result, err := svc.Update(ctx, id, "alice2", domain.UserRoleEmployee)
	require.NoError(t, err)
	assert.Equal(t, "alice2", result.Username)
}

func TestUserGetByID_Success(t *testing.T) {
	repo := new(mockUserRepo)
	svc := NewUserService(repo)
	ctx := context.Background()
	id := uuid.New()
	user := &domain.User{ID: id, Username: "alice", Role: domain.UserRoleAdmin}

	repo.On("FindByID", ctx, id).Return(user, nil)

	result, err := svc.GetByID(ctx, id)
	assert.NoError(t, err)
	assert.Equal(t, id, result.ID)
}

func TestUserGetByID_NotFound(t *testing.T) {
	repo := new(mockUserRepo)
	svc := NewUserService(repo)
	ctx := context.Background()
	id := uuid.New()

	repo.On("FindByID", ctx, id).Return(nil, pgx.ErrNoRows)

	result, err := svc.GetByID(ctx, id)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestUserGetAll_Success(t *testing.T) {
	repo := new(mockUserRepo)
	svc := NewUserService(repo)
	ctx := context.Background()

	repo.On("FindAll", ctx).Return([]domain.User{{ID: uuid.New(), Username: "alice"}}, nil)

	results, err := svc.GetAll(ctx)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestUserDelete_Success(t *testing.T) {
	repo := new(mockUserRepo)
	svc := NewUserService(repo)
	ctx := context.Background()
	id := uuid.New()

	repo.On("Delete", ctx, id).Return(nil)

	err := svc.Delete(ctx, id)
	assert.NoError(t, err)
}

func TestUserDelete_Error(t *testing.T) {
	repo := new(mockUserRepo)
	svc := NewUserService(repo)
	ctx := context.Background()
	id := uuid.New()

	repo.On("Delete", ctx, id).Return(errors.New("db error"))

	err := svc.Delete(ctx, id)
	assert.Error(t, err)
}
