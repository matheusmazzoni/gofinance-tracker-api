package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// AccountType define os tipos de conta possíveis.
type AccountType string

const (
	Checking   AccountType = "checking"    // Conta Corrente
	Savings    AccountType = "savings"     // Poupança
	CreditCard AccountType = "credit_card" // Cartão de Crédito
	Investment AccountType = "investment"  // Investimento
	Cash       AccountType = "cash"        // Dinheiro Físico
	Other      AccountType = "other"       // Outros
)

// Account representa uma conta financeira de um usuário.
type Account struct {
	Id             int64           `json:"id" db:"id"`
	UserId         int64           `json:"-" db:"user_id"`
	Name           string          `json:"name" db:"name"`
	Type           AccountType     `json:"type" db:"type"`
	InitialBalance decimal.Decimal `json:"initial_balance" db:"initial_balance"`
	Balance        decimal.Decimal `json:"balance" db:"-"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at" db:"updated_at"`
}
