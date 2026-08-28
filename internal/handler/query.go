package handler

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
)

// queryWithAlias reads the canonical snake_case query parameter while keeping
// one legacy alias during the compatibility window. Sending different values
// for both names is ambiguous and therefore rejected.
func queryWithAlias(c fiber.Ctx, canonical, alias string) (string, error) {
	canonicalValue := c.Query(canonical)
	aliasValue := c.Query(alias)

	if canonicalValue != "" && aliasValue != "" && canonicalValue != aliasValue {
		return "", fmt.Errorf("conflicting query parameters %s and %s", canonical, alias)
	}
	if canonicalValue != "" {
		return canonicalValue, nil
	}
	return aliasValue, nil
}
