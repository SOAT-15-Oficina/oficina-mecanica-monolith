package service

import (
	"time"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
)

// Apoio para os eventos de dominio da ADR-0011 emitidos pelos servicos deste
// pacote. Os nomes de campo e de evento estao em internal/observability; aqui
// fica so o calculo que depende do modelo de dominio.

// statusEnteredAt devolve quando a ordem de servico entrou no status informado.
//
// E o que permite `work_order.status_changed` carregar `duration_ms` -- o tempo
// que a OS passou no status ANTERIOR --, e e dessa duracao que sai o painel de
// tempo medio por status (`oficina.work_order_stage_duration`).
//
// As colunas de timestamp cobrem quase toda a maquina de estados, e foi por
// isso que a tabela de historico de status pode ser removida sem perder a
// metrica (ver docs/banco-de-dados.md no repositorio de infraestrutura).
//
// EM_DIAGNOSTICO e CANCELADA sao a excecao: nao tem coluna propria. Para elas
// vale `updated_at`, que e quando a linha mudou pela ultima vez -- exato quando
// a transicao foi a ultima escrita (o caso comum) e uma aproximacao quando algo
// mais editou a OS no meio. Devolver uma aproximacao rotulada e melhor do que
// devolver nada: sem isto, o diagnostico -- que e onde a OS costuma ficar
// parada -- seria a unica etapa ausente do painel.
func statusEnteredAt(wo *domain.WorkOrder, status domain.WorkOrderStatus) time.Time {
	switch status {
	case domain.WorkOrderStatusReceived:
		return wo.ReceivedAt
	case domain.WorkOrderStatusWaitingApproval:
		return valueOr(wo.QuoteSentAt, wo.UpdatedAt)
	case domain.WorkOrderStatusApproved:
		return valueOr(wo.ApprovedAt, wo.UpdatedAt)
	case domain.WorkOrderStatusInProgress:
		return valueOr(wo.StartedAt, wo.UpdatedAt)
	case domain.WorkOrderStatusFinished:
		return valueOr(wo.FinishedAt, wo.UpdatedAt)
	case domain.WorkOrderStatusDelivered:
		return valueOr(wo.DeliveredAt, wo.UpdatedAt)
	default:
		return wo.UpdatedAt
	}
}

// durationMillis devolve a duracao em milissegundos com casas decimais, na
// mesma unidade de `@duration_ms` da linha de acesso -- a metrica de log e uma
// so e nao aceita duas unidades.
func durationMillis(from, to time.Time) float64 {
	if from.IsZero() || to.Before(from) {
		return 0
	}
	return float64(to.Sub(from).Microseconds()) / 1000
}

func valueOr(value *time.Time, fallback time.Time) time.Time {
	if value == nil || value.IsZero() {
		return fallback
	}
	return *value
}
