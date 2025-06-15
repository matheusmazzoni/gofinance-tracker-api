package dto

import (
	"time"

	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/shopspring/decimal"
)

type CreateTransactionRequest struct {
	Description          string                `json:"description"`
	Amount               decimal.Decimal       `json:"amount"`
	Date                 time.Time             `json:"date"`
	Type                 model.TransactionType `json:"type"`
	AccountId            int64                 `json:"account_id"`
	CategoryId           *int64                `json:"category_id"`            // Opcional
	DestinationAccountId *int64                `json:"destination_account_id"` // Opcional, mas necessário para transferências
}

type UpdateTransactionRequest struct {
	Description          string                `json:"description" binding:"required"`
	Amount               decimal.Decimal       `json:"amount" binding:"required"`
	Date                 time.Time             `json:"date" binding:"required"`
	Type                 model.TransactionType `json:"type" binding:"required"`
	AccountId            int64                 `json:"account_id" binding:"required"`
	CategoryId           *int64                `json:"category_id"`
	DestinationAccountId *int64                `json:"destination_account_id"`
}

// PatchTransactionRequest define o corpo para uma atualização parcial de transação.
type PatchTransactionRequest struct {
	Description *string               `json:"description"`
	Amount      *decimal.Decimal      `json:"amount"`
	Date        *time.Time            `json:"date"`
	Type        model.TransactionType `json:"type"`
	AccountId   *int64                `json:"account_id"`
	CategoryId  *int64                `json:"category_id"`
}

// TransactionResponse é o DTO de resposta, com dados enriquecidos e prontos para o frontend.
type TransactionResponse struct {
	Id                   int64                 `json:"id,omitempty"`
	Description          string                `json:"description,omitempty"`
	Amount               decimal.Decimal       `json:"amount,omitempty"`
	Date                 time.Time             `json:"date,omitempty"`
	Type                 model.TransactionType `json:"type,omitempty"`
	AccountId            int64                 `json:"account_id,omitempty"`
	AccountName          string                `json:"account_name,omitempty"`
	CategoryId           *int64                `json:"category_id,omitempty"`
	CategoryName         *string               `json:"category_name,omitempty"`
	DestinationAccountId *int64                `json:"destination_account_id,omitempty"`
	CreatedAt            time.Time             `json:"created_at,omitempty"`
}
