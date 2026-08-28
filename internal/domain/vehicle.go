package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	plateOldFormat         = regexp.MustCompile(`^[A-Z]{3}[0-9]{4}$`)
	plateMercosulFormat    = regexp.MustCompile(`^[A-Z]{3}[0-9][A-Z][0-9]{2}$`)
	ErrInvalidLicensePlate = errors.New("invalid license plate format")
)

type VehicleValidationError struct {
	Err error
}

func (e *VehicleValidationError) Error() string { return e.Err.Error() }
func (e *VehicleValidationError) Unwrap() error { return e.Err }

type VehicleListFilters struct {
	CustomerID uuid.UUID
}

type Vehicle struct {
	ID           uuid.UUID `json:"id"`
	LicensePlate string    `json:"license_plate"`
	CustomerID   uuid.UUID `json:"customer_id"`
	Model        string    `json:"model"`
	Year         int       `json:"year"`
	Brand        string    `json:"brand"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (v *Vehicle) Normalize() {
	v.LicensePlate = strings.ToUpper(strings.ReplaceAll(v.LicensePlate, "-", ""))
}

func (v *Vehicle) Validate() error {
	v.Normalize()
	if v.LicensePlate == "" {
		return &VehicleValidationError{Err: errors.New("license plate is required")}
	}
	if !plateOldFormat.MatchString(v.LicensePlate) && !plateMercosulFormat.MatchString(v.LicensePlate) {
		return &VehicleValidationError{Err: ErrInvalidLicensePlate}
	}
	if v.CustomerID == uuid.Nil {
		return &VehicleValidationError{Err: errors.New("customer_id is required")}
	}
	if v.Brand == "" {
		return &VehicleValidationError{Err: errors.New("brand is required")}
	}
	if v.Model == "" {
		return &VehicleValidationError{Err: errors.New("model is required")}
	}
	if v.Year < 1900 || v.Year > time.Now().Year()+1 {
		return &VehicleValidationError{Err: errors.New("invalid year")}
	}
	return nil
}
