package domain

type WorkOrderServiceStatus string

const (
	WorkOrderServiceStatusPending            WorkOrderServiceStatus = "PENDENTE"
	WorkOrderServiceStatusAwaitingApproval   WorkOrderServiceStatus = "AGUARDANDO_APROVACAO"
	WorkOrderServiceStatusInProgress         WorkOrderServiceStatus = "EM_EXECUCAO"
	WorkOrderServiceStatusFinished           WorkOrderServiceStatus = "FINALIZADO"
)

type WorkOrderServiceApprovalStatus string

const (
	WorkOrderServiceApprovalPending  WorkOrderServiceApprovalStatus = "PENDENTE"
	WorkOrderServiceApprovalApproved WorkOrderServiceApprovalStatus = "APROVADO"
	WorkOrderServiceApprovalRejected WorkOrderServiceApprovalStatus = "REPROVADO"
)
