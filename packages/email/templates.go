package email

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
)

//go:embed templates/*.html
var templatesFS embed.FS

var templates = template.Must(template.ParseFS(templatesFS, "templates/*.html"))

type BudgetServiceItem struct {
	Title       string
	Amount      string
	Estimated   string
	ApproveLink string
	RejectLink  string
}

type BudgetEmailData struct {
	CustomerName        string
	WorkOrderCode       string
	PreviousStatusLabel string
	NewStatusLabel      string
	Amount              string
	BudgetLink          string
	Services            []BudgetServiceItem
	ApproveAllLink      string
	RejectAllLink       string
}

func RenderBudgetEmail(data BudgetEmailData) (string, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "budget.html", data); err != nil {
		return "", fmt.Errorf("email: render budget template: %w", err)
	}
	return buf.String(), nil
}

type PurchaseAlertItem struct {
	ServiceTitle string
	SupplyTitle  string
	Required     int
	InStock      int
	ToBuy        int
}

type PurchaseAlertEmailData struct {
	WorkOrderCode  string
	WorkOrderTitle string
	Items          []PurchaseAlertItem
}

func RenderPurchaseAlertEmail(data PurchaseAlertEmailData) (string, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "purchase-alert.html", data); err != nil {
		return "", fmt.Errorf("email: render purchase-alert template: %w", err)
	}
	return buf.String(), nil
}

type StatusChangeEmailData struct {
	CustomerName        string
	WorkOrderCode       string
	PreviousStatusLabel string
	NewStatusLabel      string
	Message             string
}

func RenderStatusChangeEmail(data StatusChangeEmailData) (string, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "status-change.html", data); err != nil {
		return "", fmt.Errorf("email: render status-change template: %w", err)
	}
	return buf.String(), nil
}
