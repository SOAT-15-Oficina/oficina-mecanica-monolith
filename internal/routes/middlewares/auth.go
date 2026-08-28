package middlewares

import (
	"strings"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/auth"
	"github.com/gofiber/fiber/v3"
)

const (
	RoleAdmin    = "admin"
	RoleEmployee = "employee"
)

func Auth(jwtSecretKey string) fiber.Handler {
	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing authorization header",
			})
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid authorization header format",
			})
		}

		tokenString := parts[1]
		if tokenString == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing token",
			})
		}

		claims, err := auth.ParseToken(tokenString, jwtSecretKey)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid token",
			})
		}

		c.Locals("token", claims)

		return c.Next()
	}
}

func RequireRoles(roles ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		claims, ok := c.Locals("token").(*auth.AppClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing token claims",
			})
		}

		for _, allowed := range roles {
			if claims.Role == allowed {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "insufficient permissions",
		})
	}
}
