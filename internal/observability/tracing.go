package observability

import (
	"log/slog"
	"os"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
)

func StartTracer(logger *slog.Logger) func() {
	if os.Getenv("DD_TRACE_ENABLED") != "true" {
		return func() {}
	}

	if err := tracer.Start(
		tracer.WithService(envOr("DD_SERVICE", DefaultService)),
		tracer.WithEnv(Env()),
		tracer.WithServiceVersion(version),
	); err != nil {
		logger.Error("apm: tracer nao iniciou", Err(err))
		return func() {}
	}

	logger.Info("apm: tracer iniciado")
	return tracer.Stop
}
