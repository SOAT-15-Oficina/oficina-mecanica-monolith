// Package observability monta o log estruturado da API.
//
// O CONTRATO NAO E DAQUI. Os nomes de campo e de evento sao a taxonomia da
// ADR-0011 (oficina-mecanica-infrastructure, docs/adr/0011-logs-estruturados-com-correlacao.md),
// e sao lidos literalmente pelas metricas de log, pelos paineis e pelos alertas
// do Datadog definidos em persistent/datadog_metrics.tf,
// persistent/datadog_dashboard_*.tf e persistent/datadog_monitors.tf.
//
// Renomear um campo daqui nao quebra nenhum teste de comportamento: quebra um
// painel, em silencio. Por isso os nomes sao constantes com um lugar so, e ha
// teste sobre eles.
//
// A Lambda de autenticacao (oficina-mecanica-serverless) tem um pacote gemeo,
// com os mesmos campos e as mesmas constantes. Sao dois runtimes com um formato
// de log unico -- e o que permite a mesma consulta atravessar os dois.
package observability

import (
	"context"
	"io"
	"log/slog"
	"os"

	slogtrace "github.com/DataDog/dd-trace-go/contrib/log/slog/v2"
)

// Nome deste servico no campo `service`, e a tag `service` do lado do Datadog.
// O Terraform injeta o mesmo valor em DD_SERVICE (ephemeral/k8s.tf); o default
// existe para o teste e para o docker-compose local, onde nao ha Terraform.
const DefaultService = "monolith"

// version e gravada pelo linker no build da imagem:
//
//	-ldflags="-X github.com/.../internal/observability.version=$GITHUB_SHA"
//
// Vem do linker, e nao do ambiente, porque e propriedade do ARTEFATO: uma
// variavel de ambiente pode ser trocada sem trocar a imagem, e o campo passaria
// a mentir sobre qual codigo esta rodando -- que e justamente a pergunta que
// ele existe para responder ("essa regressao veio de qual deploy?").
var version = "dev"

// Campos fixos da taxonomia (ADR-0011, secao 3).
const (
	KeyService     = "service"
	KeyEnv         = "env"
	KeyVersion     = "version"
	KeyRequestID   = "request_id"
	KeyRoute       = "route"
	KeyMethod      = "method"
	KeyStatus      = "status"
	KeyDurationMS  = "duration_ms"
	KeyEvent       = "event"
	KeyIntegration = "integration"
	KeyError       = "error"
	KeyUser        = "user"
	KeyRole        = "role"

	KeyWorkOrderID   = "work_order_id"
	KeyWorkOrderCode = "work_order_code"

	// `from` e `to` sao as duas pontas de work_order.status_changed, e as duas
	// dimensoes de `oficina.work_order_stage_duration` -- o painel de tempo
	// medio por status sai de agrupar por `@to`.
	KeyFrom = "from"
	KeyTo   = "to"

	// A decisao do cliente em approval.decided. Sem ela o evento diz que houve
	// uma decisao e nao diz qual, que e a unica coisa que se quer saber.
	KeyDecision = "decision"

	// Atributo PADRAO do Datadog para status HTTP, emitido em paralelo a
	// `status`. Nao e redundancia gratuita: no pre-processamento de log JSON o
	// Datadog trata `status` como o NIVEL do log (a mesma posicao de `level`) e
	// pode consumir o atributo. Se isso acontecer, `@status` some do log e o
	// group_by da metrica de latencia fica sem a dimensao; `@http.status_code`
	// continua la, e a correcao vira uma linha no Terraform em vez de um novo
	// deploy da aplicacao.
	KeyHTTPStatusCode = "http.status_code"
)

// Eventos de dominio (ADR-0011, secao 4).
//
// O nome vai em `event`, NUNCA no `msg`: `msg` e texto para humano e muda na
// primeira refatoracao de mensagem; `event` e identificador, e e ele que as
// metricas de log contam e os alertas consultam. Alertar sobre texto de
// mensagem e a forma mais rapida de ter um alerta que para de disparar sem
// ninguem perceber.
const (
	EventWorkOrderCreated            = "work_order.created"
	EventWorkOrderStatusChanged      = "work_order.status_changed"
	EventWorkOrderTransitionRejected = "work_order.transition_rejected"
	EventBudgetSent                  = "budget.sent"
	EventBudgetSendFailed            = "budget.send_failed"
	EventApprovalDecided             = "approval.decided"
	EventPurchaseAlertSent           = "purchase_alert.sent"
)

// Valores de `decision` em approval.decided.
const (
	DecisionApproved = "approved"
	DecisionRejected = "rejected"
)

// Dependencias externas, para o campo `integration` em linhas de ERROR. E por
// ele que `oficina.integration_error` agrupa o painel de "erros e falhas nas
// integracoes".
const (
	IntegrationSES        = "ses"
	IntegrationRDS        = "rds"
	IntegrationAPIGateway = "apigateway"
)

// Setup monta o logger do processo e o instala como default do slog.
func Setup() *slog.Logger {
	logger := New(os.Stdout)
	slog.SetDefault(logger)
	return logger
}

// New monta o logger com os tres campos que valem para toda linha do processo.
// O destino e parametro para o teste conseguir ler o que foi escrito.
func New(w io.Writer) *slog.Logger {
	env := Env()
	service := envOr("DD_SERVICE", DefaultService)

	var handler slog.Handler
	if env == EnvLocal {
		// JSON e ilegivel no terminal sem jq, e no desenvolvimento local nao ha
		// Datadog para consumi-lo (ADR-0011, "Consequencias").
		handler = slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		// O wrapper do dd-trace-go acrescenta `dd.trace_id` e `dd.span_id` a
		// linha quando ha traco no contexto -- e o que permite pular de um log
		// para o traco daquela requisicao. O caminho inverso do `request_id`,
		// que liga a linha ao access log do API Gateway.
		handler = slogtrace.WrapHandler(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return slog.New(handler).With(
		slog.String(KeyService, service),
		slog.String(KeyEnv, env),
		slog.String(KeyVersion, version),
	)
}

// EnvLocal e o ambiente sem Datadog do outro lado: o do docker-compose e o do
// teste.
const EnvLocal = "local"

// Env resolve o ambiente. DD_ENV primeiro porque e a variavel que o Datadog
// entende e que o Terraform injeta; SERVER_ENVIRONMENT como segunda opcao
// porque e a que o resto da configuracao ja usava antes disto existir.
func Env() string {
	if v := os.Getenv("DD_ENV"); v != "" {
		return v
	}
	return envOr("SERVER_ENVIRONMENT", EnvLocal)
}

// Version devolve a versao gravada no binario.
func Version() string { return version }

type loggerKey struct{}

// WithLogger guarda o logger JA DECORADO com os campos da requisicao no
// contexto. Todo log de dentro de um handler sai dele, e nunca do default: e o
// que garante que `request_id` acompanha a linha ate o fundo da pilha, inclusive
// dentro dos servicos, que nao conhecem o Fiber.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// FromContext devolve o logger da requisicao, ou o default quando nao houver
// (boot do processo, job de migration, teste que nao passou por WithLogger).
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok && logger != nil {
		return logger
	}
	return slog.Default()
}

// Event, Integration e Err existem para o nome do campo aparecer uma vez so no
// codigo de chamada. `slog.String("event", ...)` espalhado e onde o erro de
// digitacao vira painel vazio.
func Event(name string) slog.Attr { return slog.String(KeyEvent, name) }

func Integration(name string) slog.Attr { return slog.String(KeyIntegration, name) }

func Err(err error) slog.Attr {
	if err == nil {
		return slog.Attr{}
	}
	return slog.String(KeyError, err.Error())
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
