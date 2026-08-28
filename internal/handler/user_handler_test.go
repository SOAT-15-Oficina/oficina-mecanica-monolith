package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- mock UserService ---

type mockUserService struct {
	mock.Mock
}

func (m *mockUserService) Register(ctx context.Context, username, password string, role domain.UserRole) (*domain.User, error) {
	args := m.Called(ctx, username, password, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserService) Login(ctx context.Context, username, password string) (string, error) {
	args := m.Called(ctx, username, password)
	return args.String(0), args.Error(1)
}

func (m *mockUserService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserService) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserService) GetAll(ctx context.Context) ([]domain.User, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.User), args.Error(1)
}

func (m *mockUserService) Update(ctx context.Context, id uuid.UUID, username string, role domain.UserRole) (*domain.User, error) {
	args := m.Called(ctx, id, username, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserService) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// --- helpers ---

func setupUserApp(svc *mockUserService) *fiber.App {
	app := fiber.New()
	h := NewUserHandler(svc)
	app.Get("/users", h.GetAll)
	app.Get("/users/:id", h.GetByID)
	app.Put("/users/:id", h.Update)
	app.Delete("/users/:id", h.Delete)
	return app
}

func sampleUser() *domain.User {
	return &domain.User{
		ID:       uuid.New(),
		Username: "testuser",
		Role:     domain.UserRoleEmployee,
	}
}

// --- tests ---

func TestUserGetAll_Success(t *testing.T) {
	svc := new(mockUserService)
	app := setupUserApp(svc)

	svc.On("GetAll", mock.Anything).Return([]domain.User{*sampleUser()}, nil)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestUserGetAll_Empty(t *testing.T) {
	svc := new(mockUserService)
	app := setupUserApp(svc)

	svc.On("GetAll", mock.Anything).Return([]domain.User{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestUserGetAll_Error(t *testing.T) {
	svc := new(mockUserService)
	app := setupUserApp(svc)

	svc.On("GetAll", mock.Anything).Return(nil, errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestUserGetByID_Success(t *testing.T) {
	svc := new(mockUserService)
	app := setupUserApp(svc)
	u := sampleUser()

	svc.On("GetByID", mock.Anything, u.ID).Return(u, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/"+u.ID.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestUserGetByID_InvalidID(t *testing.T) {
	svc := new(mockUserService)
	app := setupUserApp(svc)

	req := httptest.NewRequest(http.MethodGet, "/users/not-a-uuid", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestUserGetByID_NotFound(t *testing.T) {
	svc := new(mockUserService)
	app := setupUserApp(svc)
	id := uuid.New()

	svc.On("GetByID", mock.Anything, id).Return(nil, pgx.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/users/"+id.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestUserUpdate_Success(t *testing.T) {
	svc := new(mockUserService)
	app := setupUserApp(svc)
	u := sampleUser()

	svc.On("Update", mock.Anything, u.ID, "newname", domain.UserRoleAdmin).Return(u, nil)

	body := `{"username":"newname","role":"admin"}`
	req := httptest.NewRequest(http.MethodPut, "/users/"+u.ID.String(), bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestUserUpdate_InvalidID(t *testing.T) {
	svc := new(mockUserService)
	app := setupUserApp(svc)

	body := `{"username":"newname","role":"admin"}`
	req := httptest.NewRequest(http.MethodPut, "/users/not-a-uuid", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestUserUpdate_InvalidJSON(t *testing.T) {
	svc := new(mockUserService)
	app := setupUserApp(svc)
	id := uuid.New()

	req := httptest.NewRequest(http.MethodPut, "/users/"+id.String(), bytes.NewReader([]byte("{invalid")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestUserUpdate_Error(t *testing.T) {
	svc := new(mockUserService)
	app := setupUserApp(svc)
	id := uuid.New()

	svc.On("Update", mock.Anything, id, "newname", domain.UserRoleAdmin).Return(nil, errors.New("update failed"))

	body := `{"username":"newname","role":"admin"}`
	req := httptest.NewRequest(http.MethodPut, "/users/"+id.String(), bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestUserDelete_Success(t *testing.T) {
	svc := new(mockUserService)
	app := setupUserApp(svc)
	id := uuid.New()

	svc.On("Delete", mock.Anything, id).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/users/"+id.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestUserDelete_InvalidID(t *testing.T) {
	svc := new(mockUserService)
	app := setupUserApp(svc)

	req := httptest.NewRequest(http.MethodDelete, "/users/bad-id", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestUserDelete_NotFound(t *testing.T) {
	svc := new(mockUserService)
	app := setupUserApp(svc)
	id := uuid.New()

	svc.On("Delete", mock.Anything, id).Return(pgx.ErrNoRows)

	req := httptest.NewRequest(http.MethodDelete, "/users/"+id.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}
