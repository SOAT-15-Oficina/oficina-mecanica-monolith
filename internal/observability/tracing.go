package observability

import (
	"log/slog"
	"os"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
)

// StartTracer liga o APM e devolve a funcao que o desliga.
//
// Fase 3 da RFC-0004. O que ele acrescenta sobre a linha de acesso ja emitida
// pelo middleware e a DECOMPOSICAO: a linha diz que a requisicao levou 800ms, o
// traco diz onde os 800ms foram. As duas convivem de proposito -- o painel de
// latencia por rota se sustenta so com o log, e continua util depois como
// contra-prova do traco.
//
// O agente e um DaemonSet, entao o endereco dele e o NO onde o pod caiu: chega
// por DD_AGENT_HOST via fieldRef (ephemeral/k8s.tf), e o dd-trace-go o le
// sozinho do ambiente, junto de DD_SERVICE, DD_ENV e DD_TRACE_SAMPLE_RATE.
//
// DESLIGADO POR PADRAO. Sem DD_TRACE_ENABLED=true a funcao nao sobe tracer
// nenhum: no docker-compose local e no CI nao ha agente para receber traco, e um
// tracer sem destino gasta goroutine e enche o log de aviso de flush falhado.
func StartTracer(logger *slog.Logger) func() {
	if os.Getenv("DD_TRACE_ENABLED") != "true" {
		return func() {}
	}

	// `version` fecha o unified service tagging do lado do traco. Ela nao vem do
	// ambiente (ver o comentario em logger.go): o Terraform nao a conhece, quem
	// a conhece e o build.
	if err := tracer.Start(
		tracer.WithService(envOr("DD_SERVICE", DefaultService)),
		tracer.WithEnv(Env()),
		tracer.WithServiceVersion(version),
	); err != nil {
		// Nao derruba o processo: APM e observabilidade, nao funcionalidade.
		// Uma API que se recusa a servir porque nao achou o agente troca um
		// problema de visibilidade por uma indisponibilidade.
		logger.Error("apm: tracer nao iniciou", Err(err))
		return func() {}
	}

	logger.Info("apm: tracer iniciado")
	return tracer.Stop
}
