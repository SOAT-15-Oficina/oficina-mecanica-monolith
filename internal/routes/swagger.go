package routes

import (
	"os"

	"github.com/gofiber/fiber/v3"
)

func registerSwagger(app *fiber.App) {
	app.Get("/docs/swagger.yaml", func(c fiber.Ctx) error {
		data, err := os.ReadFile("docs/swagger.yaml")
		if err != nil {
			return c.Status(fiber.StatusNotFound).SendString("swagger.yaml not found")
		}
		c.Set("Content-Type", "text/yaml")
		return c.Send(data)
	})

	app.Get("/swagger", func(c fiber.Ctx) error {
		c.Set("Content-Type", "text/html")
		return c.SendString(swaggerHTML)
	})
}

const swaggerHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Swagger UI - Oficina Mecanica API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>html{box-sizing:border-box;overflow-y:scroll}*,*:before,*:after{box-sizing:inherit}body{margin:0;background:#fafafa}</style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: "/docs/swagger.yaml",
      dom_id: "#swagger-ui",
      deepLinking: true,
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
      layout: "BaseLayout",
    });
  </script>
</body>
</html>`
