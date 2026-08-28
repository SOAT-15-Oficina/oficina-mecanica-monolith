package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func validVehicle() *Vehicle {
	return &Vehicle{
		LicensePlate: "ABC1234",
		CustomerID:   uuid.New(),
		Brand:        "Toyota",
		Model:        "Corolla",
		Year:         2020,
	}
}

func TestVehicleValidate_ValidOldFormat(t *testing.T) {
	v := validVehicle()
	v.LicensePlate = "ABC1234"
	assert.NoError(t, v.Validate())
}

func TestVehicleValidate_ValidOldFormatWithHyphen(t *testing.T) {
	v := validVehicle()
	v.LicensePlate = "ABC-1234"
	assert.NoError(t, v.Validate())
	assert.Equal(t, "ABC1234", v.LicensePlate)
}

func TestVehicleValidate_ValidOldFormatLowercase(t *testing.T) {
	v := validVehicle()
	v.LicensePlate = "abc1234"
	assert.NoError(t, v.Validate())
	assert.Equal(t, "ABC1234", v.LicensePlate)
}

func TestVehicleValidate_ValidMercosul(t *testing.T) {
	v := validVehicle()
	v.LicensePlate = "ABC1D23"
	assert.NoError(t, v.Validate())
}

func TestVehicleValidate_EmptyLicensePlate(t *testing.T) {
	v := validVehicle()
	v.LicensePlate = ""
	err := v.Validate()
	assert.Error(t, err)
	assert.EqualError(t, err, "license plate is required")
}

func TestVehicleValidate_InvalidFourLetters(t *testing.T) {
	v := validVehicle()
	v.LicensePlate = "ABCD123"
	assert.ErrorIs(t, v.Validate(), ErrInvalidLicensePlate)
}

func TestVehicleValidate_InvalidFiveDigits(t *testing.T) {
	v := validVehicle()
	v.LicensePlate = "ABC12345"
	assert.ErrorIs(t, v.Validate(), ErrInvalidLicensePlate)
}

func TestVehicleValidate_InvalidMixedFormat(t *testing.T) {
	v := validVehicle()
	v.LicensePlate = "1BC1234"
	assert.ErrorIs(t, v.Validate(), ErrInvalidLicensePlate)
}

func TestVehicleValidate_MissingCustomerID(t *testing.T) {
	v := validVehicle()
	v.CustomerID = uuid.Nil
	err := v.Validate()
	assert.Error(t, err)
	assert.EqualError(t, err, "customer_id is required")
}

func TestVehicleValidate_MissingBrand(t *testing.T) {
	v := validVehicle()
	v.Brand = ""
	err := v.Validate()
	assert.Error(t, err)
	assert.EqualError(t, err, "brand is required")
}

func TestVehicleValidate_MissingModel(t *testing.T) {
	v := validVehicle()
	v.Model = ""
	err := v.Validate()
	assert.Error(t, err)
	assert.EqualError(t, err, "model is required")
}

func TestVehicleValidate_YearTooOld(t *testing.T) {
	v := validVehicle()
	v.Year = 1899
	err := v.Validate()
	assert.Error(t, err)
	assert.EqualError(t, err, "invalid year")
}

func TestVehicleValidate_YearTooFuture(t *testing.T) {
	v := validVehicle()
	v.Year = time.Now().Year() + 2
	err := v.Validate()
	assert.Error(t, err)
	assert.EqualError(t, err, "invalid year")
}

func TestVehicleNormalize_RemovesHyphenAndUppercases(t *testing.T) {
	v := &Vehicle{LicensePlate: "abc-1234"}
	v.Normalize()
	assert.Equal(t, "ABC1234", v.LicensePlate)
}
