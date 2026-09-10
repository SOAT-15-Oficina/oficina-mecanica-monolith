package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Estes testes protegem um contrato que nenhum outro teste protege: os nomes
// dos campos e dos eventos sao consumidos por consultas do Datadog que vivem em
// outro repositorio (persistent/datadog_metrics.tf, persistent/datadog_monitors.tf).
// Renomear `event` para `event_name` compilaria, passaria em toda a suite e
// esvaziaria quatro paineis sem uma linha vermelha em lugar nenhum.

func decodeLine(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var out map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out))
	return out
}

func TestNew_EmitsFixedFieldsAsJSON(t *testing.T) {
	t.Setenv("DD_ENV", "prod")
	t.Setenv("DD_SERVICE", "monolith")

	var buf bytes.Buffer
	New(&buf).Info("hello")

	line := decodeLine(t, &buf)
	assert.Equal(t, "monolith", line[KeyService])
	assert.Equal(t, "prod", line[KeyEnv])
	assert.Equal(t, Version(), line[KeyVersion])
	assert.Equal(t, "hello", line["msg"])
	assert.Equal(t, "INFO", line["level"])
}

// DD_ENV e a variavel que o Datadog entende e que o Terraform injeta;
// SERVER_ENVIRONMENT e a que o resto da configuracao ja usava.
func TestEnv_PrefersDDEnvAndFallsBackToServerEnvironment(t *testing.T) {
	t.Setenv("DD_ENV", "prod")
	t.Setenv("SERVER_ENVIRONMENT", "homolog")
	assert.Equal(t, "prod", Env())

	t.Setenv("DD_ENV", "")
	assert.Equal(t, "homolog", Env())

	t.Setenv("SERVER_ENVIRONMENT", "")
	assert.Equal(t, EnvLocal, Env())
}

// Sem ambiente definido nao ha Datadog do outro lado -- e um humano lendo um
// terminal, e JSON sem jq e ilegivel.
func TestNew_UsesTextHandlerLocally(t *testing.T) {
	t.Setenv("DD_ENV", "")
	t.Setenv("SERVER_ENVIRONMENT", "")

	var buf bytes.Buffer
	New(&buf).Info("hello")

	assert.NotContains(t, buf.String(), `"msg"`)
	assert.Contains(t, buf.String(), "env=local")
}

func TestEventAndIntegrationUseTheContractedKeys(t *testing.T) {
	t.Setenv("DD_ENV", "prod")

	var buf bytes.Buffer
	New(&buf).LogAttrs(context.Background(), slog.LevelError, "boom",
		Event(EventBudgetSendFailed),
		Integration(IntegrationSES),
		Err(errors.New("connection refused")))

	line := decodeLine(t, &buf)
	assert.Equal(t, "budget.send_failed", line[KeyEvent])
	assert.Equal(t, "ses", line[KeyIntegration])
	assert.Equal(t, "connection refused", line[KeyError])
	assert.Equal(t, "ERROR", line["level"])
}

// Os nomes dos eventos sao literais nas queries do Terraform. Um teste de
// igualdade e feio e e exatamente o ponto: mudar um deles tem de exigir mudar
// tambem o repositorio de infraestrutura, no mesmo PR.
func TestEventNamesMatchTheTerraformQueries(t *testing.T) {
	assert.Equal(t, "work_order.created", EventWorkOrderCreated)
	assert.Equal(t, "work_order.status_changed", EventWorkOrderStatusChanged)
	assert.Equal(t, "work_order.transition_rejected", EventWorkOrderTransitionRejected)
	assert.Equal(t, "budget.sent", EventBudgetSent)
	assert.Equal(t, "budget.send_failed", EventBudgetSendFailed)
	assert.Equal(t, "approval.decided", EventApprovalDecided)
	assert.Equal(t, "purchase_alert.sent", EventPurchaseAlertSent)
}

func TestErr_OmitsTheFieldWhenThereIsNoError(t *testing.T) {
	t.Setenv("DD_ENV", "prod")

	var buf bytes.Buffer
	New(&buf).LogAttrs(context.Background(), slog.LevelInfo, "ok", Err(nil))

	assert.NotContains(t, decodeLine(t, &buf), KeyError)
}

func TestFromContext_ReturnsTheLoggerThatWasStored(t *testing.T) {
	t.Setenv("DD_ENV", "prod")

	var buf bytes.Buffer
	logger := New(&buf).With(slog.String(KeyRequestID, "abc-123"))

	FromContext(WithLogger(context.Background(), logger)).Info("hello")

	assert.Equal(t, "abc-123", decodeLine(t, &buf)[KeyRequestID])
}

// O caso que importa: um contexto sem logger nao pode derrubar o processo nem
// engolir a linha. Acontece no boot, no job de migration e em todo teste que
// nao passa pelo middleware.
func TestFromContext_FallsBackToTheDefault(t *testing.T) {
	assert.NotNil(t, FromContext(context.Background()))
	assert.Equal(t, slog.Default(), FromContext(context.Background()))
}

// Sem DD_TRACE_ENABLED=true nao ha agente para receber traco -- e um tracer sem
// destino gasta goroutine e enche o log de aviso de flush falhado.
func TestStartTracer_IsANoOpWhenDisabled(t *testing.T) {
	t.Setenv("DD_TRACE_ENABLED", "")

	var buf bytes.Buffer
	stop := StartTracer(New(&buf))
	stop()

	assert.Empty(t, buf.String())
}
