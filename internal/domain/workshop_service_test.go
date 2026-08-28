package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func validWorkshopService() *WorkshopService {
	return &WorkshopService{
		Title:                "Troca de oleo",
		Description:          "Troca de oleo do motor",
		PriceCents:           5000,
		EstimatedTimeMinutes: 30,
		Active:               true,
	}
}

func TestValidate_ValidService(t *testing.T) {
	// should accept a valid service with all fields
	ws := validWorkshopService()
	assert.NoError(t, ws.Validate())
}

func TestValidate_EmptyTitle(t *testing.T) {
	// should reject when title is empty
	ws := validWorkshopService()
	ws.Title = ""
	assert.ErrorIs(t, ws.Validate(), ErrWorkshopServiceTitleRequired)
}

func TestValidate_WhitespaceOnlyTitle(t *testing.T) {
	// should reject when title is only whitespace (normalized to empty)
	ws := validWorkshopService()
	ws.Title = "   "
	assert.ErrorIs(t, ws.Validate(), ErrWorkshopServiceTitleRequired)
}

func TestValidate_TitleTooShort(t *testing.T) {
	// should reject when title has less than 3 characters
	ws := validWorkshopService()
	ws.Title = "ab"
	assert.ErrorIs(t, ws.Validate(), ErrWorkshopServiceTitleLength)
}

func TestValidate_TitleTooLong(t *testing.T) {
	// should reject when title exceeds 150 characters
	ws := validWorkshopService()
	ws.Title = strings.Repeat("a", 151)
	assert.ErrorIs(t, ws.Validate(), ErrWorkshopServiceTitleLength)
}

func TestValidate_TitleExactBoundaries(t *testing.T) {
	// should accept title at minimum (3) and maximum (150) boundaries
	ws := validWorkshopService()

	ws.Title = "abc"
	assert.NoError(t, ws.Validate())

	ws.Title = strings.Repeat("a", 150)
	assert.NoError(t, ws.Validate())
}

func TestValidate_DescriptionTooLong(t *testing.T) {
	// should reject when description exceeds 500 characters
	ws := validWorkshopService()
	ws.Description = strings.Repeat("a", 501)
	assert.ErrorIs(t, ws.Validate(), ErrWorkshopServiceDescriptionLength)
}

func TestValidate_EmptyDescription(t *testing.T) {
	// should accept empty description since it is optional
	ws := validWorkshopService()
	ws.Description = ""
	assert.NoError(t, ws.Validate())
}

func TestValidate_ZeroPrice(t *testing.T) {
	// should reject when price is zero
	ws := validWorkshopService()
	ws.PriceCents = 0
	assert.ErrorIs(t, ws.Validate(), ErrWorkshopServicePriceMustBePositive)
}

func TestValidate_NegativePrice(t *testing.T) {
	// should reject when price is negative
	ws := validWorkshopService()
	ws.PriceCents = -100
	assert.ErrorIs(t, ws.Validate(), ErrWorkshopServicePriceMustBePositive)
}

func TestValidate_ZeroDuration(t *testing.T) {
	// should reject when estimated time is zero
	ws := validWorkshopService()
	ws.EstimatedTimeMinutes = 0
	assert.ErrorIs(t, ws.Validate(), ErrWorkshopServiceDurationMustBePositive)
}

func TestValidate_NegativeDuration(t *testing.T) {
	// should reject when estimated time is negative
	ws := validWorkshopService()
	ws.EstimatedTimeMinutes = -10
	assert.ErrorIs(t, ws.Validate(), ErrWorkshopServiceDurationMustBePositive)
}

func TestNormalize_TrimsWhitespace(t *testing.T) {
	// should trim leading/trailing whitespace from title and description
	ws := &WorkshopService{
		Title:       "  Troca de oleo  ",
		Description: "  descricao  ",
	}
	ws.Normalize()
	assert.Equal(t, "Troca de oleo", ws.Title)
	assert.Equal(t, "descricao", ws.Description)
}

func TestDeactivate(t *testing.T) {
	// should set active to false and update the timestamp
	ws := validWorkshopService()
	ws.Active = true
	ws.Deactivate()
	assert.False(t, ws.Active)
	assert.False(t, ws.UpdatedAt.IsZero())
}
