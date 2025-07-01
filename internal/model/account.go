package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type AccountType string

const (
	Checking   AccountType = "checking"
	Savings    AccountType = "savings"
	CreditCard AccountType = "credit_card"
	Other      AccountType = "other"
)

type Account struct {
	Id                  int64            `json:"id" db:"id"`
	UserId              int64            `json:"-" db:"user_id"`
	Name                string           `json:"name" db:"name"`
	Type                AccountType      `json:"type" db:"type"`
	InitialBalance      decimal.Decimal  `json:"initial_balance" db:"initial_balance"`
	Balance             decimal.Decimal  `json:"balance" db:"-"`
	CreatedAt           time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time        `json:"updated_at" db:"updated_at"`
	CreditLimit         *decimal.Decimal `json:"credit_limit,omitempty" db:"credit_limit"`
	StatementClosingDay *int             `json:"statement_closing_day,omitempty" db:"statement_closing_day"`
	PaymentDueDay       *int             `json:"payment_due_day,omitempty" db:"payment_due_day"`
}
