package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// TransactionType define o tipo de uma transação.
type TransactionType string

const (
	Income   TransactionType = "income"
	Expense  TransactionType = "expense"
	Transfer TransactionType = "transfer"
)

// Transaction representa uma única operação financeira.
type Transaction struct {
	Id                   int64           `json:"id" db:"id"`
	UserId               int64           `json:"user_id" db:"user_id"`
	Description          string          `json:"description" db:"description"`
	Amount               decimal.Decimal `json:"amount" db:"amount"`
	Date                 time.Time       `json:"date" db:"date"`
	Type                 TransactionType `json:"type" db:"type"`
	AccountId            int64           `json:"account_id" db:"account_id"`
	DestinationAccountId *int64          `json:"destination_account_id,omitempty" db:"destination_account_id"`
	CategoryId           *int64          `json:"category_id,omitempty" db:"category_id"`
	CreatedAt            time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at" db:"updated_at"`

	// Campos populados para respostas de API, não são colunas diretas
	CategoryName *string `json:"category_name,omitempty" db:"category_name"`
	AccountName  string  `json:"account_name" db:"account_name"`
	Tags         []Tag   `json:"tags,omitempty"`
}
