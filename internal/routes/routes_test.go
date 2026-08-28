package routes

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/config"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubDatabasePinger struct {
	err error
}

func (s stubDatabasePinger) Ping(context.Context) error {
	return s.err
}

func TestRegisterRoutes(t *testing.T) {
	app := fiber.New()
	cfg := &config.Config{
		Server: &config.ServerConfig{BaseURL: "http://localhost:3000"},
		JWT:    &config.JWTConfig{SecretKey: "test-secret"},
	}
	// Should not panic with nil db and nil email provider
	RegisterRoutes(app, nil, cfg, nil)

	// Verify ping route works
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	// A pod without a usable database must not receive traffic.
	req = httptest.NewRequest(http.MethodGet, "/ready", nil)
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)
}

func TestReadinessReflectsDatabaseAvailability(t *testing.T) {
	tests := []struct {
		name       string
		pinger     databasePinger
		statusCode int
	}{
		{name: "database available", pinger: stubDatabasePinger{}, statusCode: fiber.StatusOK},
		{name: "database unavailable", pinger: stubDatabasePinger{err: errors.New("unavailable")}, statusCode: fiber.StatusServiceUnavailable},
		{name: "database missing", pinger: nil, statusCode: fiber.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			registerHealthRoutes(app, tt.pinger)

			req := httptest.NewRequest(http.MethodGet, "/ready", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.statusCode, resp.StatusCode)
		})
	}
}

func TestSwagger(t *testing.T) {
	app := fiber.New()
	registerSwagger(app)

	req := httptest.NewRequest(http.MethodGet, "/swagger", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestSwaggerYAML_NotFound(t *testing.T) {
	app := fiber.New()
	registerSwagger(app)

	req := httptest.NewRequest(http.MethodGet, "/docs/swagger.yaml", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}
