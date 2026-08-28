package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockRepo struct {
	mock.Mock
}

func (m *mockRepo) Create(ctx context.Context, ws *domain.WorkshopService) (*domain.WorkshopService, error) {
	args := m.Called(ctx, ws)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkshopService), args.Error(1)
}

func (m *mockRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.WorkshopService, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkshopService), args.Error(1)
}

func (m *mockRepo) List(ctx context.Context, filters domain.WorkshopServiceListFilters) ([]domain.WorkshopService, int, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).([]domain.WorkshopService), args.Int(1), args.Error(2)
}

func (m *mockRepo) Update(ctx context.Context, ws *domain.WorkshopService) (*domain.WorkshopService, error) {
	args := m.Called(ctx, ws)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkshopService), args.Error(1)
}

func (m *mockRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockRepo) Deactivate(ctx context.Context, id uuid.UUID) (*domain.WorkshopService, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkshopService), args.Error(1)
}

func (m *mockRepo) ExistsByTitle(ctx context.Context, title string, excludeID *uuid.UUID) (bool, error) {
	args := m.Called(ctx, title, excludeID)
	return args.Bool(0), args.Error(1)
}

func (m *mockRepo) HasWorkOrderLinks(ctx context.Context, id uuid.UUID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

func (m *mockRepo) GetAvgExecutionTime(ctx context.Context, filters domain.AvgExecutionTimeFilters) ([]domain.AvgExecutionTimeResult, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).([]domain.AvgExecutionTimeResult), args.Error(1)
}

func (m *mockRepo) SubtractSuppliesFromStock(ctx context.Context, serviceID uuid.UUID) error {
	return m.Called(ctx, serviceID).Error(0)
}

func newTestService() *domain.WorkshopService {
	return &domain.WorkshopService{
		Title:                "Troca de oleo",
		Description:          "Troca completa",
		PriceCents:           5000,
		EstimatedTimeMinutes: 30,
	}
}

func savedTestService() *domain.WorkshopService {
	return &domain.WorkshopService{
		ID:                   uuid.New(),
		Title:                "Troca de oleo",
		Description:          "Troca completa",
		PriceCents:           5000,
		EstimatedTimeMinutes: 30,
		Active:               true,
		CreatedAt:            time.Now().UTC(),
		UpdatedAt:            time.Now().UTC(),
	}
}

func TestCreate_Success(t *testing.T) {
	// should create a valid service when title is unique
	repo := new(mockRepo)
	svc := NewWorkshopServiceManager(repo)
	ctx := context.Background()
	ws := newTestService()

	repo.On("ExistsByTitle", ctx, ws.Title, (*uuid.UUID)(nil)).Return(false, nil)
	repo.On("Create", ctx, ws).Return(savedTestService(), nil)

	result, err := svc.Create(ctx, ws)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, ws.Active)
	repo.AssertExpectations(t)
}

func TestCreate_DuplicateTitle(t *testing.T) {
	// should return error when title already exists
	repo := new(mockRepo)
	svc := NewWorkshopServiceManager(repo)
	ctx := context.Background()
	ws := newTestService()

	repo.On("ExistsByTitle", ctx, ws.Title, (*uuid.UUID)(nil)).Return(true, nil)

	result, err := svc.Create(ctx, ws)
	assert.ErrorIs(t, err, ErrWorkshopServiceTitleAlreadyExists)
	assert.Nil(t, result)
	repo.AssertNotCalled(t, "Create")
}

func TestCreate_ValidationError(t *testing.T) {
	// should return domain validation error for invalid data
	repo := new(mockRepo)
	svc := NewWorkshopServiceManager(repo)
	ctx := context.Background()
	ws := &domain.WorkshopService{
		Title:                "",
		PriceCents:           5000,
		EstimatedTimeMinutes: 30,
	}

	result, err := svc.Create(ctx, ws)
	assert.ErrorIs(t, err, domain.ErrWorkshopServiceTitleRequired)
	assert.Nil(t, result)
	repo.AssertNotCalled(t, "ExistsByTitle")
}

func TestGetByID_Success(t *testing.T) {
	// should return the service when found
	repo := new(mockRepo)
	svc := NewWorkshopServiceManager(repo)
	ctx := context.Background()
	expected := savedTestService()

	repo.On("FindByID", ctx, expected.ID).Return(expected, nil)

	result, err := svc.GetByID(ctx, expected.ID)
	assert.NoError(t, err)
	assert.Equal(t, expected.ID, result.ID)
}

func TestGetByID_NotFound(t *testing.T) {
	// should return pgx.ErrNoRows when not found
	repo := new(mockRepo)
	svc := NewWorkshopServiceManager(repo)
	ctx := context.Background()
	id := uuid.New()

	repo.On("FindByID", ctx, id).Return(nil, pgx.ErrNoRows)

	result, err := svc.GetByID(ctx, id)
	assert.ErrorIs(t, err, pgx.ErrNoRows)
	assert.Nil(t, result)
}

func TestList_DefaultPagination(t *testing.T) {
	// should default page=1 and limit=10 when not provided
	repo := new(mockRepo)
	svc := NewWorkshopServiceManager(repo)
	ctx := context.Background()

	expectedFilters := domain.WorkshopServiceListFilters{Page: 1, Limit: 10}
	repo.On("List", ctx, expectedFilters).Return([]domain.WorkshopService{}, 0, nil)

	items, total, err := svc.List(ctx, domain.WorkshopServiceListFilters{})
	assert.NoError(t, err)
	assert.Empty(t, items)
	assert.Equal(t, 0, total)
	repo.AssertExpectations(t)
}

func TestList_WithFilters(t *testing.T) {
	// should pass active filter through to repository
	repo := new(mockRepo)
	svc := NewWorkshopServiceManager(repo)
	ctx := context.Background()
	active := true

	filters := domain.WorkshopServiceListFilters{Active: &active, Page: 1, Limit: 5}
	repo.On("List", ctx, filters).Return([]domain.WorkshopService{*savedTestService()}, 1, nil)

	items, total, err := svc.List(ctx, filters)
	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, 1, total)
}

func TestUpdate_Success(t *testing.T) {
	// should apply partial update and return updated service
	repo := new(mockRepo)
	svc := NewWorkshopServiceManager(repo)
	ctx := context.Background()
	existing := savedTestService()

	newTitle := "Alinhamento"
	input := WorkshopServiceUpdateInput{Title: &newTitle}

	repo.On("FindByID", ctx, existing.ID).Return(existing, nil)
	repo.On("ExistsByTitle", ctx, newTitle, &existing.ID).Return(false, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.WorkshopService")).Return(&domain.WorkshopService{
		ID:                   existing.ID,
		Title:                newTitle,
		Description:          existing.Description,
		PriceCents:           existing.PriceCents,
		EstimatedTimeMinutes: existing.EstimatedTimeMinutes,
		Active:               existing.Active,
		CreatedAt:            existing.CreatedAt,
		UpdatedAt:            time.Now().UTC(),
	}, nil)

	result, err := svc.Update(ctx, existing.ID, input)
	assert.NoError(t, err)
	assert.Equal(t, newTitle, result.Title)
}

func TestUpdate_DuplicateTitle(t *testing.T) {
	// should reject update when new title already exists for another service
	repo := new(mockRepo)
	svc := NewWorkshopServiceManager(repo)
	ctx := context.Background()
	existing := savedTestService()

	newTitle := "Balanceamento"
	input := WorkshopServiceUpdateInput{Title: &newTitle}

	repo.On("FindByID", ctx, existing.ID).Return(existing, nil)
	repo.On("ExistsByTitle", ctx, newTitle, &existing.ID).Return(true, nil)

	result, err := svc.Update(ctx, existing.ID, input)
	assert.ErrorIs(t, err, ErrWorkshopServiceTitleAlreadyExists)
	assert.Nil(t, result)
	repo.AssertNotCalled(t, "Update")
}

func TestUpdate_NotFound(t *testing.T) {
	// should return error when service to update does not exist
	repo := new(mockRepo)
	svc := NewWorkshopServiceManager(repo)
	ctx := context.Background()
	id := uuid.New()

	newTitle := "Teste"
	input := WorkshopServiceUpdateInput{Title: &newTitle}

	repo.On("FindByID", ctx, id).Return(nil, pgx.ErrNoRows)

	result, err := svc.Update(ctx, id, input)
	assert.ErrorIs(t, err, pgx.ErrNoRows)
	assert.Nil(t, result)
}

func TestUpdate_ActiveFlag(t *testing.T) {
	repo := new(mockRepo)
	svc := NewWorkshopServiceManager(repo)
	ctx := context.Background()
	existing := savedTestService()
	active := false

	input := WorkshopServiceUpdateInput{Active: &active}
	updated := *existing
	updated.Active = false
	updated.UpdatedAt = time.Now().UTC()

	repo.On("FindByID", ctx, existing.ID).Return(existing, nil)
	repo.On("ExistsByTitle", ctx, existing.Title, &existing.ID).Return(false, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.WorkshopService")).Return(&updated, nil)

	result, err := svc.Update(ctx, existing.ID, input)
	assert.NoError(t, err)
	assert.False(t, result.Active)
	repo.AssertExpectations(t)
}

func TestDelete_HardDelete(t *testing.T) {
	// should hard delete when service has no work order links
	repo := new(mockRepo)
	svc := NewWorkshopServiceManager(repo)
	ctx := context.Background()
	existing := savedTestService()

	repo.On("FindByID", ctx, existing.ID).Return(existing, nil)
	repo.On("HasWorkOrderLinks", ctx, existing.ID).Return(false, nil)
	repo.On("Delete", ctx, existing.ID).Return(nil)

	result, err := svc.Delete(ctx, existing.ID)
	assert.NoError(t, err)
	assert.True(t, result.Deleted)
	assert.False(t, result.Deactivated)
	assert.Nil(t, result.DeactivatedResource)
}

func TestDelete_SoftDelete(t *testing.T) {
	// should soft delete (deactivate) when service has work order links
	repo := new(mockRepo)
	svc := NewWorkshopServiceManager(repo)
	ctx := context.Background()
	existing := savedTestService()
	deactivated := *existing
	deactivated.Active = false

	repo.On("FindByID", ctx, existing.ID).Return(existing, nil)
	repo.On("HasWorkOrderLinks", ctx, existing.ID).Return(true, nil)
	repo.On("Deactivate", ctx, existing.ID).Return(&deactivated, nil)

	result, err := svc.Delete(ctx, existing.ID)
	assert.NoError(t, err)
	assert.False(t, result.Deleted)
	assert.True(t, result.Deactivated)
	assert.NotNil(t, result.DeactivatedResource)
	assert.False(t, result.DeactivatedResource.Active)
}

func TestDelete_NotFound(t *testing.T) {
	// should return error when service to delete does not exist
	repo := new(mockRepo)
	svc := NewWorkshopServiceManager(repo)
	ctx := context.Background()
	id := uuid.New()

	repo.On("FindByID", ctx, id).Return(nil, pgx.ErrNoRows)

	result, err := svc.Delete(ctx, id)
	assert.ErrorIs(t, err, pgx.ErrNoRows)
	assert.Nil(t, result)
}

func TestUpdate_AllFields(t *testing.T) {
	repo := new(mockRepo)
	svc := NewWorkshopServiceManager(repo)
	ctx := context.Background()
	id := uuid.New()

	current := &domain.WorkshopService{
		ID: id, Title: "Original", Description: "Desc", PriceCents: 100,
		EstimatedTimeMinutes: 30, Active: true, CreatedAt: time.Now(),
	}
	title := "Updated"
	desc := "New desc"
	price := 200
	estTime := 60
	active := false

	repo.On("FindByID", ctx, id).Return(current, nil)
	repo.On("ExistsByTitle", ctx, "Updated", &id).Return(false, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.WorkshopService")).Return(current, nil)

	input := WorkshopServiceUpdateInput{
		Title: &title, Description: &desc, PriceCents: &price,
		EstimatedTimeMinutes: &estTime, Active: &active,
	}
	result, err := svc.Update(ctx, id, input)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdate_ValidationError(t *testing.T) {
	repo := new(mockRepo)
	svc := NewWorkshopServiceManager(repo)
	ctx := context.Background()
	id := uuid.New()

	current := &domain.WorkshopService{
		ID: id, Title: "Original", Description: "Desc", PriceCents: 100,
		EstimatedTimeMinutes: 30, Active: true,
	}
	title := "ab" // too short - validation error

	repo.On("FindByID", ctx, id).Return(current, nil)

	input := WorkshopServiceUpdateInput{Title: &title}
	result, err := svc.Update(ctx, id, input)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestUpdate_ExistsByTitleError(t *testing.T) {
	repo := new(mockRepo)
	svc := NewWorkshopServiceManager(repo)
	ctx := context.Background()
	id := uuid.New()

	current := &domain.WorkshopService{
		ID: id, Title: "Valid Title", Description: "Desc", PriceCents: 100,
		EstimatedTimeMinutes: 30, Active: true,
	}
	title := "New Title"

	repo.On("FindByID", ctx, id).Return(current, nil)
	repo.On("ExistsByTitle", ctx, "New Title", &id).Return(false, errors.New("db error"))

	input := WorkshopServiceUpdateInput{Title: &title}
	result, err := svc.Update(ctx, id, input)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestUpdate_RepoUpdateError(t *testing.T) {
	repo := new(mockRepo)
	svc := NewWorkshopServiceManager(repo)
	ctx := context.Background()
	id := uuid.New()

	current := &domain.WorkshopService{
		ID: id, Title: "Valid Title", Description: "Desc", PriceCents: 100,
		EstimatedTimeMinutes: 30, Active: true,
	}

	repo.On("FindByID", ctx, id).Return(current, nil)
	repo.On("ExistsByTitle", ctx, "Valid Title", &id).Return(false, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.WorkshopService")).Return(nil, errors.New("db error"))

	input := WorkshopServiceUpdateInput{}
	result, err := svc.Update(ctx, id, input)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestDelete_HasLinksError(t *testing.T) {
	repo := new(mockRepo)
	svc := NewWorkshopServiceManager(repo)
	ctx := context.Background()
	id := uuid.New()

	ws := &domain.WorkshopService{ID: id, Title: "Test", Active: true}
	repo.On("FindByID", ctx, id).Return(ws, nil)
	repo.On("HasWorkOrderLinks", ctx, id).Return(false, errors.New("db error"))

	result, err := svc.Delete(ctx, id)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestDelete_DeactivateError(t *testing.T) {
	repo := new(mockRepo)
	svc := NewWorkshopServiceManager(repo)
	ctx := context.Background()
	id := uuid.New()

	ws := &domain.WorkshopService{ID: id, Title: "Test", Active: true}
	repo.On("FindByID", ctx, id).Return(ws, nil)
	repo.On("HasWorkOrderLinks", ctx, id).Return(true, nil)
	repo.On("Deactivate", ctx, id).Return(nil, errors.New("db error"))

	result, err := svc.Delete(ctx, id)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestDelete_HardDeleteError(t *testing.T) {
	repo := new(mockRepo)
	svc := NewWorkshopServiceManager(repo)
	ctx := context.Background()
	id := uuid.New()

	ws := &domain.WorkshopService{ID: id, Title: "Test", Active: true}
	repo.On("FindByID", ctx, id).Return(ws, nil)
	repo.On("HasWorkOrderLinks", ctx, id).Return(false, nil)
	repo.On("Delete", ctx, id).Return(errors.New("db error"))

	result, err := svc.Delete(ctx, id)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestCreate_ExistsByTitleError(t *testing.T) {
	repo := new(mockRepo)
	svc := NewWorkshopServiceManager(repo)
	ctx := context.Background()

	ws := &domain.WorkshopService{
		Title: "Valid Title", Description: "Description", PriceCents: 5000, EstimatedTimeMinutes: 30,
	}
	repo.On("ExistsByTitle", ctx, "Valid Title", (*uuid.UUID)(nil)).Return(false, errors.New("db error"))

	result, err := svc.Create(ctx, ws)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestCreate_RepoCreateError(t *testing.T) {
	repo := new(mockRepo)
	svc := NewWorkshopServiceManager(repo)
	ctx := context.Background()

	ws := &domain.WorkshopService{
		Title: "Valid Title", Description: "Description", PriceCents: 5000, EstimatedTimeMinutes: 30,
	}
	repo.On("ExistsByTitle", ctx, "Valid Title", (*uuid.UUID)(nil)).Return(false, nil)
	repo.On("Create", ctx, mock.AnythingOfType("*domain.WorkshopService")).Return(nil, errors.New("db error"))

	result, err := svc.Create(ctx, ws)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestWorkshopService_GetAvgExecutionTime_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := NewWorkshopServiceManager(repo)
	ctx := context.Background()

	filters := domain.AvgExecutionTimeFilters{}
	expected := []domain.AvgExecutionTimeResult{
		{ServiceID: uuid.New(), Title: "Troca de óleo", AvgRealTimeMinutes: 45.5, ExecutionCount: 10},
	}

	repo.On("GetAvgExecutionTime", ctx, filters).Return(expected, nil)

	results, err := svc.GetAvgExecutionTime(ctx, filters)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, 45.5, results[0].AvgRealTimeMinutes)
}

func TestWorkshopService_GetAvgExecutionTime_RepoError(t *testing.T) {
	repo := new(mockRepo)
	svc := NewWorkshopServiceManager(repo)
	ctx := context.Background()

	filters := domain.AvgExecutionTimeFilters{}

	repo.On("GetAvgExecutionTime", ctx, filters).Return([]domain.AvgExecutionTimeResult{}, errors.New("db error"))

	results, err := svc.GetAvgExecutionTime(ctx, filters)
	assert.Error(t, err)
	assert.Empty(t, results)
}
