package middlewares

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/auth"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const secret = "test-secret"

func newApp() *fiber.App {
	app := fiber.New()
	app.Use(Auth(secret))
	app.Get("/protected", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	return app
}

func validToken(t *testing.T, role string) string {
	t.Helper()
	tok, err := auth.GenerateToken("alice", role, secret)
	require.NoError(t, err)
	return tok
}

func doGet(app *fiber.App, authHeader string) *http.Response {
	return doGetURL(app, "/protected", authHeader)
}

func doGetURL(app *fiber.App, url, authHeader string) *http.Response {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	resp, _ := app.Test(req)
	return resp
}

func TestAuth_MissingHeader(t *testing.T) {
	resp := doGet(newApp(), "")
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestAuth_InvalidFormat_NoBearer(t *testing.T) {
	resp := doGet(newApp(), "Token abc123")
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestAuth_InvalidFormat_BearerOnly(t *testing.T) {
	resp := doGet(newApp(), "Bearer")
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestAuth_InvalidToken(t *testing.T) {
	resp := doGet(newApp(), "Bearer not.valid.token")
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestAuth_ValidToken(t *testing.T) {
	resp := doGet(newApp(), "Bearer "+validToken(t, "admin"))
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func newRoleApp(roles ...string) *fiber.App {
	app := fiber.New()
	app.Use(Auth(secret))
	app.Get("/role", RequireRoles(roles...), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	return app
}

func TestRequireRoles_Allowed(t *testing.T) {
	app := newRoleApp(RoleAdmin)
	resp := doGetURL(app, "/role", "Bearer "+validToken(t, RoleAdmin))
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestRequireRoles_Forbidden(t *testing.T) {
	app := newRoleApp(RoleAdmin)
	resp := doGetURL(app, "/role", "Bearer "+validToken(t, RoleEmployee))
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestRequireRoles_MissingClaims(t *testing.T) {
	// RequireRoles without Auth middleware — no token claims in context
	app := fiber.New()
	app.Get("/role", RequireRoles(RoleAdmin), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/role", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestRequireRoles_MultipleRolesAllowed(t *testing.T) {
	app := newRoleApp(RoleAdmin, RoleEmployee)
	resp := doGetURL(app, "/role", "Bearer "+validToken(t, RoleEmployee))
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func bodyError(t *testing.T, resp *http.Response) string {
	t.Helper()
	b, _ := io.ReadAll(resp.Body)
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		return ""
	}
	return m["error"]
}

func TestAuth_MissingHeader_ErrorMessage(t *testing.T) {
	resp := doGet(newApp(), "")
	assert.Contains(t, bodyError(t, resp), "missing authorization header")
}

func TestAuth_InvalidToken_ErrorMessage(t *testing.T) {
	resp := doGet(newApp(), "Bearer bad.token.here")
	assert.Contains(t, bodyError(t, resp), "invalid token")
}
