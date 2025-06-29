package dto

import (
	"time"

	"github.com/shopspring/decimal"
)

// StatementPeriod represents the start and end dates of a billing cycle.
type StatementPeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// StatementResponse is the DTO for returning a full credit card statement.
type StatementResponse struct {
	AccountName    string                `json:"account_name"`
	StatementTotal decimal.Decimal       `json:"statement_total"`
	PaymentDueDate time.Time             `json:"payment_due_date"`
	Period         StatementPeriod       `json:"period"`
	Transactions   []TransactionResponse `json:"transactions"` // We reuse the existing TransactionResponse DTO
}
