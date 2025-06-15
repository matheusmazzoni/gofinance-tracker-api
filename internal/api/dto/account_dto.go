package dto

import (
	"time"

	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/shopspring/decimal"
)

// CreateAccountRequest define o corpo da requisição para criar uma nova conta.
type CreateAccountRequest struct {
	Name           string            `json:"name" binding:"required,min=2"`
	Type           model.AccountType `json:"type" binding:"required"`
	InitialBalance decimal.Decimal   `json:"initial_balance" binding:"required"`
}

type CreateAccountResponse struct {
	Id int64 `json:"id"`
}

// UpdateAccountRequest define o corpo da requisição para atualizar uma conta.
type UpdateAccountRequest struct {
	Name           string            `json:"name" binding:"required,min=2"`
	Type           model.AccountType `json:"type" binding:"required"`
	InitialBalance decimal.Decimal   `json:"initial_balance" binding:"required"`
}

// AccountResponse define a estrutura de resposta para uma ou mais contas.
// É o que o cliente da API realmente verá.
type AccountResponse struct {
	Id             int64             `json:"id"`
	Name           string            `json:"name"`
	Type           model.AccountType `json:"type"`
	InitialBalance decimal.Decimal   `json:"initial_balance"`
	Balance        decimal.Decimal   `json:"balance"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}
