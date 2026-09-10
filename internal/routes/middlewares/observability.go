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

// RequestIDHeader e o nome do header nos dois sentidos: e por ele que o
// identificador entra (injetado pelo API Gateway) e por ele que ele volta na
// resposta, para quem chamou conseguir citar a requisicao num chamado.
const RequestIDHeader = "X-Request-Id"

// maxRequestIDLength corta um header abusivo antes de ele virar uma tag de
// traco e um campo de log de 4KB.
const maxRequestIDLength = 128

// healthProbePaths nao geram linha de acesso nem span.
//
// O kubelet chama /ping e /ready a cada 5 e 10 segundos, em cada um dos ate 10
// pods: sao centenas de milhares de linhas por dia dizendo que nada aconteceu.
// Elas nao respondem nenhuma pergunta dos paineis e pagam ingestao no Datadog.
//
// O uptime do /ping continua coberto, e melhor: o teste sintetico
// (persistent/datadog_monitors.tf) o chama pela URL publica, medindo o que o
// usuario sente em vez do que o kubelet ve de dentro do no.
var healthProbePaths = map[string]bool{
	"/ping":  true,
	"/ready": true,
}

// Observability e o PRIMEIRO middleware da cadeia, antes de Auth (ADR-0011,
// secao 2). Ele faz quatro coisas, e nenhuma delas o handler precisa saber:
//
//  1. resolve o `request_id` da requisicao -- herdado da borda ou gerado;
//  2. poe um *slog.Logger ja decorado com ele no context.Context da requisicao,
//     que e o mesmo contexto que os handlers passam aos servicos;
//  3. abre o span de APM da requisicao;
//  4. emite a linha de acesso ao final, com rota, status e duracao.
//
// A linha de acesso e a fonte de `oficina.http_request_duration`
// (persistent/datadog_metrics.tf), que sustenta o painel de latencia por rota.
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

			// A mesma chave do log. E o que permite achar o traco a partir de
			// uma linha de log e vice-versa, sem depender de o `dd.trace_id`
			// ter sido injetado naquela linha especifica.
			tracer.Tag(observability.KeyRequestID, requestID),

			// Sem `Measured`, a latencia deste span so aparece dentro de um
			// traco; com ele, o Datadog calcula percentis por rota do lado dele.
			tracer.Measured(),
		)

		logger := slog.Default().With(slog.String(observability.KeyRequestID, requestID))
		c.SetContext(observability.WithLogger(ctx, logger))

		// A linha de acesso e o span saem num `defer` por causa do PANICO.
		//
		// Nao ha middleware de recover nesta aplicacao, entao um panico no
		// handler sobe ate o fasthttp, que fecha a conexao: hoje ele nao deixa
		// rastro nenhum -- nem linha de log, nem span, e o span aberto fica
		// pendurado. E justamente a requisicao que mais se quer ver.
		//
		// O panico e re-lancado depois de registrado: o comportamento da
		// aplicacao nao muda, so deixa de ser invisivel.
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

// finishRequest emite a linha de acesso e fecha o span.
//
// A ROTA SO PODE SER LIDA AQUI. Antes de `c.Next()`, `c.Route()` ainda e a rota
// do proprio middleware -- o campo iria para o painel como "/".
//
// E e o PADRAO da rota (`/work-orders/:id`), e nao o caminho concreto: com o
// caminho, cada ordem de servico viraria uma serie propria na metrica e o painel
// de latencia por rota teria uma linha por OS.
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

// resolveRequestID pega o identificador da borda, ou fabrica um.
//
// LE O ULTIMO VALOR, e nao o primeiro. O API Gateway injeta o dele com
// `append:` (ephemeral/apigateway_routes.tf), entao um X-Request-Id que o
// cliente tenha mandado vem ANTES do valor do gateway. So o ultimo nasceu na
// borda, e header de cliente e entrada nao confiavel -- nao pode virar chave de
// correlacao sem passar por filtro.
func resolveRequestID(c fiber.Ctx) string {
	values := c.Request().Header.PeekAll(RequestIDHeader)
	if len(values) > 0 {
		raw := string(values[len(values)-1])

		// Uma unica linha de header pode carregar a lista inteira
		// ("do-cliente, do-gateway"), dependendo de quem a montou.
		parts := strings.Split(raw, ",")
		if id := sanitizeRequestID(parts[len(parts)-1]); id != "" {
			return id
		}
	}

	// Chamada interna, teste local, ou header que nao passou no filtro.
	return uuid.NewString()
}

// sanitizeRequestID devolve "" para qualquer coisa que nao pareca um
// identificador. Este valor vai para um header de resposta, para uma tag de
// traco e para toda linha de log da requisicao: aceitar bytes arbitrarios de um
// cliente e convidar injecao de log.
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

// responseStatus resolve o status que o cliente de fato recebeu.
//
// Quando o handler devolve erro, quem escreve o status e o ErrorHandler do
// Fiber, DEPOIS deste middleware -- ler a resposta agora daria 200 numa
// requisicao que falhou, e o painel de taxa de erro nasceria mentindo.
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

// accessLevel traduz o status HTTP para o nivel da linha de acesso. E o que
// permite alertar sobre `status:error` sem alertar sobre todo 404 de rota
// digitada errada.
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

// spanError marca o span como erro so no 5xx. Um 404 ou um 401 e resposta
// correta da aplicacao; contar os dois como erro no APM faz a taxa de erro do
// servico seguir o comportamento do cliente, e nao a saude do sistema.
func spanError(status int, err error) error {
	if status < http.StatusInternalServerError {
		return nil
	}
	if err != nil {
		return err
	}
	return errors.New(http.StatusText(status))
}

// millisSince devolve a duracao em milissegundos com casas decimais. Inteiro
// arredondaria para 0 a maior parte das requisicoes, e uma distribuicao de
// zeros nao responde nada sobre latencia.
func millisSince(start time.Time) float64 {
	return float64(time.Since(start).Microseconds()) / 1000
}
