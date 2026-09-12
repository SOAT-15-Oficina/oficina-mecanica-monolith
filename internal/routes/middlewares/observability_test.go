package middlewares

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/observability"
	"github.com/gofiber/fiber/v3"
	recovermw "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	t.Setenv("DD_ENV", "prod")

	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(observability.New(&buf))
	t.Cleanup(func() { slog.SetDefault(previous) })

	return &buf
}

func observabilityApp() *fiber.App {
	app := fiber.New()
	app.Use(Observability())

	app.Get("/work-orders/:id", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	app.Get("/boom", func(c fiber.Ctx) error {
		return fiber.NewError(fiber.StatusInternalServerError, "boom")
	})
	app.Get("/ping", func(c fiber.Ctx) error {
		return c.SendString("Pong")
	})

	return app
}

func accessLine(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.NotEmpty(t, lines[0], "nenhuma linha de log foi emitida")

	var out map[string]any
	require.NoError(t, json.Unmarshal([]byte(lines[len(lines)-1]), &out))
	return out
}

func TestObservability_EmitsAccessLine(t *testing.T) {
	buf := captureLogs(t)

	resp, err := observabilityApp().Test(httptest.NewRequest(http.MethodGet, "/work-orders/123", nil))
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	line := accessLine(t, buf)
	assert.Equal(t, "request", line["msg"])
	assert.Equal(t, http.MethodGet, line[observability.KeyMethod])
	assert.Equal(t, float64(200), line[observability.KeyStatus])
	assert.Equal(t, float64(200), line[observability.KeyHTTPStatusCode])
	assert.GreaterOrEqual(t, line[observability.KeyDurationMS], float64(0))
	assert.NotEmpty(t, line[observability.KeyRequestID])
}

func TestObservability_LogsTheRoutePatternAndNotThePath(t *testing.T) {
	buf := captureLogs(t)

	_, err := observabilityApp().Test(httptest.NewRequest(http.MethodGet, "/work-orders/123", nil))
	require.NoError(t, err)

	assert.Equal(t, "/work-orders/:id", accessLine(t, buf)[observability.KeyRoute])
}

func TestObservability_UsesTheLastRequestIDHeader(t *testing.T) {
	buf := captureLogs(t)

	req := httptest.NewRequest(http.MethodGet, "/work-orders/123", nil)
	req.Header.Add(RequestIDHeader, "valor-do-cliente")
	req.Header.Add(RequestIDHeader, "valor-do-gateway")

	resp, err := observabilityApp().Test(req)
	require.NoError(t, err)

	assert.Equal(t, "valor-do-gateway", accessLine(t, buf)[observability.KeyRequestID])
	assert.Equal(t, "valor-do-gateway", resp.Header.Get(RequestIDHeader))
}

func TestObservability_SplitsACommaSeparatedRequestIDList(t *testing.T) {
	buf := captureLogs(t)

	req := httptest.NewRequest(http.MethodGet, "/work-orders/123", nil)
	req.Header.Set(RequestIDHeader, "valor-do-cliente, valor-do-gateway")

	_, err := observabilityApp().Test(req)
	require.NoError(t, err)

	assert.Equal(t, "valor-do-gateway", accessLine(t, buf)[observability.KeyRequestID])
}

func TestObservability_RejectsAnAbusiveRequestIDHeader(t *testing.T) {
	for name, value := range map[string]string{
		"caractere invalido": `id" injetado`,
		"longo demais":       strings.Repeat("a", maxRequestIDLength+1),
		"vazio":              "   ",
	} {
		t.Run(name, func(t *testing.T) {
			buf := captureLogs(t)

			req := httptest.NewRequest(http.MethodGet, "/work-orders/123", nil)
			req.Header.Set(RequestIDHeader, value)

			_, err := observabilityApp().Test(req)
			require.NoError(t, err)

			got := accessLine(t, buf)[observability.KeyRequestID]
			assert.NotEmpty(t, got)
			assert.NotEqual(t, value, got)
		})
	}
}

func TestObservability_AccessLineLevelFollowsTheStatus(t *testing.T) {
	tests := map[string]struct {
		path   string
		status float64
		level  string
	}{
		"sucesso":      {path: "/work-orders/123", status: 200, level: "INFO"},
		"nao achado":   {path: "/inexistente", status: 404, level: "WARN"},
		"erro interno": {path: "/boom", status: 500, level: "ERROR"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			buf := captureLogs(t)

			_, err := observabilityApp().Test(httptest.NewRequest(http.MethodGet, tt.path, nil))
			require.NoError(t, err)

			line := accessLine(t, buf)
			assert.Equal(t, tt.status, line[observability.KeyStatus])
			assert.Equal(t, tt.level, line["level"])
		})
	}
}

func TestObservability_DoesNotLogHealthProbes(t *testing.T) {
	buf := captureLogs(t)

	resp, err := observabilityApp().Test(httptest.NewRequest(http.MethodGet, "/ping", nil))
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	assert.Empty(t, buf.String())
}

func TestObservability_PutsTheRequestLoggerInTheContext(t *testing.T) {
	buf := captureLogs(t)

	app := fiber.New()
	app.Use(Observability())
	app.Get("/from-service", func(c fiber.Ctx) error {
		observability.FromContext(c.Context()).InfoContext(c.Context(), "vindo do servico")
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/from-service", nil)
	req.Header.Set(RequestIDHeader, "id-do-gateway")

	_, err := app.Test(req)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 2, "esperadas a linha do servico e a linha de acesso")

	var serviceLine map[string]any
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &serviceLine))
	assert.Equal(t, "vindo do servico", serviceLine["msg"])
	assert.Equal(t, "id-do-gateway", serviceLine[observability.KeyRequestID])
}

func TestObservability_LogsAPanicAndRethrowsIt(t *testing.T) {
	buf := captureLogs(t)

	app := fiber.New()
	app.Use(recovermw.New())
	app.Use(Observability())
	app.Get("/panic", func(c fiber.Ctx) error {
		panic("boom")
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/panic", nil))
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

	line := accessLine(t, buf)
	assert.Equal(t, float64(500), line[observability.KeyStatus])
	assert.Equal(t, "ERROR", line["level"])
	assert.Contains(t, line[observability.KeyError], "boom")
}
