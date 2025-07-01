package dto

import (
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/shopspring/decimal"
)

type AccountRequest struct {
	Name                string            `json:"name" binding:"required,min=1,max=100" example:"Nubank Account" minLength:"2" maxLength:"100"`
	Type                model.AccountType `json:"type" binding:"required,oneof=checking savings credit_card other" example:"checking" enums:"checking,savings,credit_card,other"`
	InitialBalance      *decimal.Decimal  `json:"initial_balance" binding:"required" example:"1000.50"`
	CreditLimit         *decimal.Decimal  `json:"credit_limit,omitempty" binding:"omitempty" example:"5000.00"`
	StatementClosingDay *int              `json:"statement_closing_day,omitempty" binding:"omitempty" example:"28"`
	PaymentDueDay       *int              `json:"payment_due_day,omitempty" binding:"omitempty" example:"5" `
}

// Validate contains the custom, struct-level validation logic for a AccountRequest.
func (req *AccountRequest) Validate(sl validator.StructLevel) {
	isCreditCard := req.Type == model.CreditCard

	if isCreditCard {
		// Credit card fields are required.
		if req.StatementClosingDay == nil {
			sl.ReportError(req.StatementClosingDay, "statement_closing_day", "StatementClosingDay", "required_for_credit_card", "")
		} else if *req.StatementClosingDay <= 0 || *req.StatementClosingDay > 31 {
			sl.ReportError(req.StatementClosingDay, "statement_closing_day", "StatementClosingDay", "day", strconv.Itoa(*req.StatementClosingDay))
		}

		if req.PaymentDueDay == nil {
			sl.ReportError(req.PaymentDueDay, "payment_due_day", "PaymentDueDay", "required_for_credit_card", "")
		} else if *req.PaymentDueDay <= 0 || *req.PaymentDueDay > 31 {
			sl.ReportError(req.PaymentDueDay, "payment_due_day", "PaymentDueDay", "day", strconv.Itoa(*req.PaymentDueDay))
		}

		if req.CreditLimit == nil {
			sl.ReportError(req.CreditLimit, "credit_limit", "CreditLimit", "required_for_credit_card", "")
		} else {
			if req.CreditLimit.IsZero() || req.CreditLimit.IsNegative() {
				sl.ReportError(req.CreditLimit, "credit_limit", "CreditLimit", "gt_credit_card", "0")
			}
		}
	} else {
		//If not a credit card, these fields are not allowed.
		if req.StatementClosingDay != nil {
			sl.ReportError(req.StatementClosingDay, "statement_closing_day", "StatementClosingDay", "not_allowed_for_non_credit_card", "")
		}
		if req.PaymentDueDay != nil {
			sl.ReportError(req.PaymentDueDay, "payment_due_day", "PaymentDueDay", "not_allowed_for_non_credit_card", "")
		}
		if req.CreditLimit != nil {
			sl.ReportError(req.CreditLimit, "credit_limit", "CreditLimit", "not_allowed_for_non_credit_card", "")
		}
	}
}

type AccountResponse struct {
	Id                  int64             `json:"id,omitempty"`
	Name                string            `json:"name,omitempty"`
	Type                model.AccountType `json:"type,omitempty"`
	Balance             *decimal.Decimal  `json:"balance,omitempty"`
	InitialBalance      *decimal.Decimal  `json:"initial_balance,omitempty"`
	CreditLimit         *decimal.Decimal  `json:"credit_limit,omitempty"`
	PaymentDueDay       *int              `json:"due_day,omitempty"`
	StatementClosingDay *int              `json:"closing_day,omitempty"`
}
