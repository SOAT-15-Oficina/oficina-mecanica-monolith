package routes

import (
	"bufio"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/config"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var fiberPathParameter = regexp.MustCompile(`:([^/]+)`)

func TestOpenAPICoversRegisteredAPIRoutes(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app, nil, &config.Config{
		Server: &config.ServerConfig{BaseURL: "http://localhost:8080"},
		JWT:    &config.JWTConfig{SecretKey: "test-secret"},
	}, nil)

	ignored := map[string]bool{
		"GET /": true, "GET /web*": true,
		"GET /swagger": true, "GET /docs/swagger.yaml": true,
	}
	registered := map[string]bool{}
	for _, route := range app.GetRoutes(true) {
		method := strings.ToUpper(route.Method)
		if method != "GET" && method != "POST" && method != "PUT" && method != "DELETE" {
			continue
		}
		path := route.Path
		if len(path) > 1 {
			path = strings.TrimSuffix(path, "/")
		}
		key := method + " " + fiberPathParameter.ReplaceAllString(path, `{$1}`)
		if !ignored[key] {
			registered[key] = true
		}
	}

	documented := readOpenAPIOperations(t, "../../docs/swagger.yaml")
	assert.Equal(t, sortedKeys(registered), sortedKeys(documented))
}

func readOpenAPIOperations(t *testing.T, path string) map[string]bool {
	t.Helper()
	file, err := os.Open(path)
	require.NoError(t, err)
	defer file.Close()

	operations := map[string]bool{}
	currentPath := ""
	inPaths := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case line == "paths:":
			inPaths = true
		case line == "components:":
			inPaths = false
		case inPaths && strings.HasPrefix(line, "  /") && strings.HasSuffix(line, ":"):
			currentPath = strings.TrimSuffix(strings.TrimSpace(line), ":")
		case inPaths && currentPath != "":
			method := strings.TrimSuffix(strings.TrimSpace(line), ":")
			if strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "      ") &&
				(method == "get" || method == "post" || method == "put" || method == "delete") {
				operations[strings.ToUpper(method)+" "+currentPath] = true
			}
		}
	}
	require.NoError(t, scanner.Err())
	return operations
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
