package middlewares

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/ext"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/auth"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/observability"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-Id"

const maxRequestIDLength = 128

var healthProbePaths = map[string]bool{
	"/ping":  true,
	"/ready": true,
}

func Observability() fiber.Handler {
	return func(c fiber.Ctx) (err error) {
		if healthProbePaths[c.Path()] {
			return c.Next()
		}

		start := time.Now()

		requestID := resolveRequestID(c)
		c.Set(RequestIDHeader, requestID)

		span, ctx := tracer.StartSpanFromContext(c.Context(), "http.request",
			tracer.SpanType(ext.SpanTypeWeb),
			tracer.Tag(ext.HTTPMethod, c.Method()),
			tracer.Tag(ext.HTTPURL, c.Path()),
			tracer.Tag(observability.KeyRequestID, requestID),
			tracer.Measured(),
		)

		logger := slog.Default().With(slog.String(observability.KeyRequestID, requestID))
		c.SetContext(observability.WithLogger(ctx, logger))

		defer func() {
			if recovered := recover(); recovered != nil {
				finishRequest(c, span, logger, start, fiber.StatusInternalServerError,
					fmt.Errorf("panic: %v", recovered))
				panic(recovered)
			}

			finishRequest(c, span, logger, start, responseStatus(c, err), err)
		}()

		err = c.Next()
		return err
	}
}

func finishRequest(c fiber.Ctx, span *tracer.Span, logger *slog.Logger, start time.Time, status int, err error) {
	route := c.Route().Path

	attrs := []slog.Attr{
		slog.String(observability.KeyRoute, route),
		slog.String(observability.KeyMethod, c.Method()),
		slog.Int(observability.KeyStatus, status),
		slog.Int(observability.KeyHTTPStatusCode, status),
		slog.Float64(observability.KeyDurationMS, millisSince(start)),
	}
	if claims, ok := c.Locals("token").(*auth.AppClaims); ok && claims != nil {
		attrs = append(attrs,
			slog.String(observability.KeyUser, claims.User),
			slog.String(observability.KeyRole, claims.Role))
	}
	if err != nil {
		attrs = append(attrs, observability.Err(err))
	}

	logger.LogAttrs(c.Context(), accessLevel(status), "request", attrs...)

	span.SetTag(ext.ResourceName, c.Method()+" "+route)
	span.SetTag(ext.HTTPCode, status)
	span.Finish(tracer.WithError(spanError(status, err)))
}

func resolveRequestID(c fiber.Ctx) string {
	values := c.Request().Header.PeekAll(RequestIDHeader)
	if len(values) > 0 {
		raw := string(values[len(values)-1])

		parts := strings.Split(raw, ",")
		if id := sanitizeRequestID(parts[len(parts)-1]); id != "" {
			return id
		}
	}

	return uuid.NewString()
}

func sanitizeRequestID(raw string) string {
	id := strings.TrimSpace(raw)
	if id == "" || len(id) > maxRequestIDLength {
		return ""
	}

	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-', r == '_', r == '.', r == ':', r == '=':
		default:
			return ""
		}
	}

	return id
}

func responseStatus(c fiber.Ctx, err error) int {
	if err == nil {
		return c.Response().StatusCode()
	}

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		return fiberErr.Code
	}

	return fiber.StatusInternalServerError
}

func accessLevel(status int) slog.Level {
	switch {
	case status >= http.StatusInternalServerError:
		return slog.LevelError
	case status >= http.StatusBadRequest:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}

func spanError(status int, err error) error {
	if status < http.StatusInternalServerError {
		return nil
	}
	if err != nil {
		return err
	}
	return errors.New(http.StatusText(status))
}

func millisSince(start time.Time) float64 {
	return float64(time.Since(start).Microseconds()) / 1000
}
