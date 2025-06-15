package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// Budget define um limite de gastos para uma categoria em um período.
type Budget struct {
	Id         int64           `json:"id" db:"id"`
	UserId     int64           `json:"user_id" db:"user_id"`
	CategoryId int64           `json:"category_id" db:"category_id"`
	Amount     decimal.Decimal `json:"amount" db:"amount"` // Valor total do orçamento
	Month      int             `json:"month" db:"month"`   // Mês (1-12)
	Year       int             `json:"year" db:"year"`     // Ano (ex: 2025)
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at" db:"updated_at"`
}
