package dto

import (
	"time"

	"github.com/shopspring/decimal"
)

// CreateBudgetRequest defines the body for creating a new budget.
type CreateBudgetRequest struct {
	CategoryId int64           `json:"category_id" binding:"required"`
	Amount     decimal.Decimal `json:"amount" binding:"required"`
	Month      int             `json:"month" binding:"required,min=1,max=12"`
	Year       int             `json:"year" binding:"required"`
}

// UpdateBudgetRequest defines the body for updating a budget's amount.
type UpdateBudgetRequest struct {
	Amount decimal.Decimal `json:"amount" binding:"required,gt=0"`
}

// BudgetResponse is the DTO for returning a budget with its real-time progress.
// This is the main object the frontend will consume for the dashboard.
type BudgetResponse struct {
	Id           int64           `json:"id"`
	CategoryId   int64           `json:"category_id"`
	CategoryName string          `json:"category_name"`
	Amount       decimal.Decimal `json:"amount"`       // The planned budget amount
	SpentAmount  decimal.Decimal `json:"spent_amount"` // The calculated amount spent so far
	Balance      decimal.Decimal `json:"balance"`      // The remaining balance (Amount - Spent)
	Month        int             `json:"month"`
	Year         int             `json:"year"`
	CreatedAt    time.Time       `json:"created_at"`
}
