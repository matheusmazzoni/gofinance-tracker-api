package dto

import (
	"time"

	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/shopspring/decimal"
)

type CreateAccountRequest struct {
	Name                string            `json:"name" binding:"required,min=2"`
	Type                model.AccountType `json:"type" binding:"required"`
	InitialBalance      decimal.Decimal   `json:"initial_balance" binding:"required"`
	StatementClosingDay *int              `json:"statement_closing_day"`
	PaymentDueDay       *int              `json:"payment_due_day"`
}

type CreateAccountResponse struct {
	Id int64 `json:"id"`
}

type UpdateAccountRequest struct {
	Name                string            `json:"name" binding:"required,min=2"`
	Type                model.AccountType `json:"type" binding:"required"`
	InitialBalance      decimal.Decimal   `json:"initial_balance" binding:"required"`
	StatementClosingDay *int              `json:"statement_closing_day"`
	PaymentDueDay       *int              `json:"payment_due_day"`
}
type AccountResponse struct {
	Id                  int64             `json:"id"`
	Name                string            `json:"name"`
	Type                model.AccountType `json:"type"`
	InitialBalance      decimal.Decimal   `json:"initial_balance"`
	Balance             decimal.Decimal   `json:"balance"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
	StatementClosingDay *int              `json:"statement_closing_day"`
	PaymentDueDay       *int              `json:"payment_due_day"`
}
