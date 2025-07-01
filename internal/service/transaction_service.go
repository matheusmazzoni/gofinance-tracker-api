package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/matheusmazzoni/gofinance-tracker-api/internal/api/dto"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/repository"
)

// Sentinel errors are used throughout the service to provide specific,
// checkable error types to the calling layer (e.g., the API handler).
var (
	ErrAmountNotPositive               = errors.New("transaction amount must be positive")
	ErrSourceAccountNotFound           = errors.New("source account not found or does not belong to the user")
	ErrDestinationAccountRequired      = errors.New("destination account is required for a transfer")
	ErrSameAccounts                    = errors.New("source and destination accounts cannot be the same")
	ErrDestinationAccountNotFound      = errors.New("destination account not found or does not belong to the user")
	ErrNewAccountNotFound              = errors.New("new account not found or does not belong to the user")
	ErrSourceAccountTransferCreditCard = errors.New("transfer transaction is not allowed for source account as credit_card")
)

// TransactionService encapsulates the business logic for transactions.
type TransactionService struct {
	repo        repository.TransactionRepository
	accountRepo repository.AccountRepository
}

// NewTransactionService creates a new instance of the TransactionService.
func NewTransactionService(repo repository.TransactionRepository, accountRepo repository.AccountRepository) *TransactionService {
	return &TransactionService{
		repo:        repo,
		accountRepo: accountRepo,
	}
}

// CreateTransaction handles the business logic for creating a transaction,
// including validation of accounts and amounts.
func (s *TransactionService) CreateTransaction(ctx context.Context, tx model.Transaction) (int64, error) {
	// Business logic validation starts here.
	if tx.Amount.IsNegative() || tx.Amount.IsZero() {
		return 0, ErrAmountNotPositive
	}

	// Verify that the source account exists and belongs to the user.
	sourceAccount, err := s.accountRepo.GetById(ctx, tx.AccountId, tx.UserId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrSourceAccountNotFound
		}
		return 0, fmt.Errorf("failed to get source account: %w", err)
	}

	// Credit card limit validation for expense transactions
	if sourceAccount.Type == model.CreditCard && sourceAccount.CreditLimit != nil {

		// Get current balance (negative value represents debt)
		currentBalance, err := s.accountRepo.GetCurrentBalance(ctx, tx.AccountId, tx.UserId)
		if err != nil {
			return 0, fmt.Errorf("failed to get current balance for credit card validation: %w", err)
		}

		// Calculate what the new balance would be after this expense
		// Example: current balance -800, expense 50 -> new balance -850
		newBalance := currentBalance.Sub(tx.Amount)

		// Check if the absolute value of the new debt exceeds the credit limit
		// Example: abs(-850) = 850, limit = 1000 -> OK
		// Example: abs(-1100) = 1100, limit = 1000 -> ERROR
		if newBalance.Abs().GreaterThan(*sourceAccount.CreditLimit) {
			availableCredit := sourceAccount.CreditLimit.Add(currentBalance)
			return 0, fmt.Errorf(
				"transaction exceeds credit card limit. Limit: %s, Current Debt: %s, New Debt: %s, Avaliable Credit: %s",
				sourceAccount.CreditLimit.String(),
				currentBalance.Abs().String(),
				newBalance.Abs().String(),
				availableCredit.String(),
			)
		}
	}

	// Transfer-specific validations
	if tx.Type == model.Transfer {
		// Transfer must have a destination account
		if tx.DestinationAccountId == nil {
			return 0, ErrDestinationAccountRequired
		}
		// Cannot transfer to the same account
		if tx.AccountId == *tx.DestinationAccountId {
			return 0, ErrSameAccounts
		}
		// Credit cards cannot be used as source for transfers
		if sourceAccount.Type == model.CreditCard {
			return 0, ErrSourceAccountTransferCreditCard
		}

		// Verify destination account exists and belongs to the user
		_, err := s.accountRepo.GetById(ctx, *tx.DestinationAccountId, tx.UserId)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, ErrDestinationAccountNotFound
			}
			return 0, err
		}
	}

	return s.repo.Create(ctx, tx)
}

// GetTransactionById retrieves a single transaction, ensuring it belongs to the specified user.
func (s *TransactionService) GetTransactionById(ctx context.Context, id, userId int64) (*model.Transaction, error) {
	return s.repo.GetById(ctx, id, userId)
}

// ListTransactions lists all transactions for a specific user.
func (s *TransactionService) ListTransactions(ctx context.Context, userId int64, filters repository.ListTransactionFilters) ([]model.Transaction, error) {
	return s.repo.List(ctx, userId, filters)
}

// UpdateTransaction handles the logic for updating an entire transaction entity.
// It requires the userId to ensure authorization.
func (s *TransactionService) UpdateTransaction(ctx context.Context, tx model.Transaction) (*model.Transaction, error) {

	// Business validations for the new data.
	if tx.Amount.IsNegative() || tx.Amount.IsZero() {
		return nil, ErrAmountNotPositive
	}
	if tx.Type == model.Transfer {
		if tx.DestinationAccountId == nil {
			return nil, ErrDestinationAccountRequired
		}
		if tx.AccountId == *tx.DestinationAccountId {
			return nil, ErrSameAccounts
		}
	}

	// Persist the changes.
	err := s.repo.Update(ctx, tx)
	if err != nil {
		return nil, err
	}

	// Return the newly updated transaction to the handler.
	return s.repo.GetById(ctx, tx.Id, tx.UserId)
}

// PatchTransaction applies a partial update to a transaction. It fetches the
// original transaction and merges the requested changes before saving.
func (s *TransactionService) PatchTransaction(ctx context.Context, id, userId int64, req dto.PatchTransactionRequest) (*model.Transaction, error) {
	// 1. Fetch the original state of the transaction to ensure it exists and belongs to the user.
	txToUpdate, err := s.repo.GetById(ctx, id, userId)
	if err != nil {
		return nil, err // Returns sql.ErrNoRows if not found.
	}

	// 2. Apply changes from the request DTO to the model.
	if req.Description != nil {
		txToUpdate.Description = *req.Description
	}
	if req.Amount != nil {
		if req.Amount.IsNegative() || req.Amount.IsZero() {
			return nil, ErrAmountNotPositive
		}
		txToUpdate.Amount = *req.Amount
	}
	if req.Date != nil {
		txToUpdate.Date = *req.Date
	}
	if req.AccountId != nil {
		// Extra validation: ensure the new account exists and belongs to the user.
		_, err := s.accountRepo.GetById(ctx, *req.AccountId, userId)
		if err != nil {
			return nil, ErrNewAccountNotFound
		}
		txToUpdate.AccountId = *req.AccountId
	}
	if req.CategoryId != nil {
		txToUpdate.CategoryId = req.CategoryId
	}

	// 3. Save the merged, validated object.
	if err := s.repo.Update(ctx, *txToUpdate); err != nil {
		return nil, err
	}

	// Return the fully updated transaction to the handler.
	return s.repo.GetById(ctx, id, userId)
}

// DeleteTransaction handles the deletion of a transaction.
// It requires the userId to ensure a user can only delete their own transactions.
func (s *TransactionService) DeleteTransaction(ctx context.Context, id, userId int64) error {
	// Future business logic, like creating an audit log, could be added here
	// before calling the repository.
	return s.repo.Delete(ctx, id, userId)
}
