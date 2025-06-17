package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// Budget defines a spending limit for a category in a given period.
type Budget struct {
	Id         int64           `json:"id" db:"id"`
	UserId     int64           `json:"-" db:"user_id"`
	CategoryId int64           `json:"category_id" db:"category_id"`
	Amount     decimal.Decimal `json:"amount" db:"amount"`
	Month      int             `json:"month" db:"month"`
	Year       int             `json:"year" db:"year"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at" db:"updated_at"`

	// Extra fields for enriched data from JOINs
	CategoryName string `json:"category_name,omitempty" db:"category_name"`
}
