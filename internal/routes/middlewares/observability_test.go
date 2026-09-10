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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A linha de acesso e a fonte de `oficina.http_request_duration`
// (persistent/datadog_metrics.tf), e `request_id` e o que liga essa linha ao
// access log do API Gateway. Os dois sao contrato com outro repositorio: quebra
// aqui nao acende luz vermelha em lugar nenhum, so esvazia painel.

// captureLogs troca o logger default por um que escreve num buffer, e o devolve
// ao final. O middleware sai do default de proposito -- e o unico ponto do
// codigo onde ainda nao ha logger de requisicao para herdar.
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

// O PADRAO da rota, e nao o caminho concreto. Com o caminho, cada ordem de
// servico viraria uma serie propria na metrica e o painel de latencia por rota
// teria uma linha por OS.
func TestObservability_LogsTheRoutePatternAndNotThePath(t *testing.T) {
	buf := captureLogs(t)

	_, err := observabilityApp().Test(httptest.NewRequest(http.MethodGet, "/work-orders/123", nil))
	require.NoError(t, err)

	assert.Equal(t, "/work-orders/:id", accessLine(t, buf)[observability.KeyRoute])
}

// O valor do gateway vem por ULTIMO, porque ele e injetado com `append:`. Ler o
// primeiro seria adotar como chave de correlacao um header que o cliente
// controla.
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

// Este valor vai para header de resposta, tag de traco e toda linha de log da
// requisicao. Aceitar bytes arbitrarios de um cliente e convidar injecao.
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

// 5xx e erro, 4xx e aviso: e o que permite alertar sobre `status:error` sem
// alertar sobre todo 404 de rota digitada errada.
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

// O kubelet chama /ping e /ready a cada poucos segundos em cada pod. Registrar
// isso e pagar ingestao para dizer que nada aconteceu.
func TestObservability_DoesNotLogHealthProbes(t *testing.T) {
	buf := captureLogs(t)

	resp, err := observabilityApp().Test(httptest.NewRequest(http.MethodGet, "/ping", nil))
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	assert.Empty(t, buf.String())
}

// O logger do contexto e o que os handlers repassam aos servicos. Sem ele, todo
// log de dentro da aplicacao perde o `request_id` e a correlacao para na borda.
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
