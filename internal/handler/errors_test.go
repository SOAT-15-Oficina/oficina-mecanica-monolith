package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDbErrResponse_NoRows(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c fiber.Ctx) error {
		handled, resp := dbErrResponse(c, pgx.ErrNoRows, "item not found")
		if handled {
			return resp
		}
		return c.SendStatus(200)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestDbErrResponse_UniqueViolation_KnownConstraint(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c fiber.Ctx) error {
		pgErr := &pgconn.PgError{Code: "23505", ConstraintName: "users_username_key"}
		handled, resp := dbErrResponse(c, pgErr, "not found")
		if handled {
			return resp
		}
		return c.SendStatus(200)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
}

func TestDbErrResponse_UniqueViolation_UnknownConstraint(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c fiber.Ctx) error {
		pgErr := &pgconn.PgError{Code: "23505", ConstraintName: "unknown_key", Detail: "some detail"}
		handled, resp := dbErrResponse(c, pgErr, "not found")
		if handled {
			return resp
		}
		return c.SendStatus(200)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
}

func TestDbErrResponse_ForeignKeyViolation_KnownConstraint(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c fiber.Ctx) error {
		pgErr := &pgconn.PgError{Code: "23503", ConstraintName: "fk_vehicles_customer"}
		handled, resp := dbErrResponse(c, pgErr, "not found")
		if handled {
			return resp
		}
		return c.SendStatus(200)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
}

func TestDbErrResponse_ForeignKeyViolation_UnknownConstraint(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c fiber.Ctx) error {
		pgErr := &pgconn.PgError{Code: "23503", ConstraintName: "unknown_fk", Detail: "fk detail"}
		handled, resp := dbErrResponse(c, pgErr, "not found")
		if handled {
			return resp
		}
		return c.SendStatus(200)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
}

func TestDbErrResponse_NotNullViolation(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c fiber.Ctx) error {
		pgErr := &pgconn.PgError{Code: "23502", Message: "null value in column"}
		handled, resp := dbErrResponse(c, pgErr, "not found")
		if handled {
			return resp
		}
		return c.SendStatus(200)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestDbErrResponse_CheckViolation(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c fiber.Ctx) error {
		pgErr := &pgconn.PgError{Code: "23514", Message: "check constraint violated"}
		handled, resp := dbErrResponse(c, pgErr, "not found")
		if handled {
			return resp
		}
		return c.SendStatus(200)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestDbErrResponse_UnknownPgError(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c fiber.Ctx) error {
		pgErr := &pgconn.PgError{Code: "99999"}
		handled, resp := dbErrResponse(c, pgErr, "not found")
		if handled {
			return resp
		}
		return c.SendStatus(200)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestDbErrResponse_NonPgError(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c fiber.Ctx) error {
		handled, resp := dbErrResponse(c, assert.AnError, "not found")
		if handled {
			return resp
		}
		return c.SendStatus(200)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}
