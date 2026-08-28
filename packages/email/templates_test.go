package email

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderBudgetEmail_Success(t *testing.T) {
	data := BudgetEmailData{
		CustomerName:        "João",
		WorkOrderCode:       "OS-001",
		PreviousStatusLabel: "Em diagnóstico",
		NewStatusLabel:      "Aguardando aprovação",
		Amount:              "R$ 150,00",
		BudgetLink:          "https://example.com/budget",
		Services: []BudgetServiceItem{
			{Title: "Troca de óleo", Amount: "R$ 80,00", Estimated: "30 min", ApproveLink: "https://approve", RejectLink: "https://reject"},
		},
		ApproveAllLink: "https://approve-all",
		RejectAllLink:  "https://reject-all",
	}

	body, err := RenderBudgetEmail(data)
	assert.NoError(t, err)
	assert.NotEmpty(t, body)
	assert.Contains(t, body, "João")
	assert.Contains(t, body, "OS-001")
	assert.Contains(t, body, "Em diagnóstico")
	assert.Contains(t, body, "Aguardando aprovação")
	assert.Contains(t, body, "https://approve-all")
	assert.Contains(t, body, "https://reject-all")
}

func TestRenderStatusChangeEmail_Success(t *testing.T) {
	data := StatusChangeEmailData{
		CustomerName:        "Maria",
		WorkOrderCode:       "OS-202",
		PreviousStatusLabel: "Aprovada",
		NewStatusLabel:      "Em execução",
		Message:             "A execução dos serviços da sua ordem de serviço foi iniciada.",
	}

	body, err := RenderStatusChangeEmail(data)
	assert.NoError(t, err)
	assert.NotEmpty(t, body)
	assert.Contains(t, body, "Maria")
	assert.Contains(t, body, "OS-202")
	assert.Contains(t, body, "Aprovada")
	assert.Contains(t, body, "Em execução")
	assert.Contains(t, body, "Status anterior:")
	assert.Contains(t, body, "Novo status:")
}

func TestRenderPurchaseAlertEmail_Success(t *testing.T) {
	data := PurchaseAlertEmailData{
		WorkOrderCode:  "OS-20260101-AB12",
		WorkOrderTitle: "Revisão",
		Items: []PurchaseAlertItem{
			{ServiceTitle: "Troca de óleo", SupplyTitle: "Filtro", Required: 5, InStock: 2, ToBuy: 3},
		},
	}

	body, err := RenderPurchaseAlertEmail(data)
	assert.NoError(t, err)
	assert.NotEmpty(t, body)
	assert.Contains(t, body, "OS-20260101-AB12")
}
