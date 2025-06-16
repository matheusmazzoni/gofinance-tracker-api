package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/matheusmazzoni/gofinance-tracker-api/internal/api/dto"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/repository"
)

// Sentinel errors are used throughout the service to provide specific,
// checkable error types to the calling layer (e.g., the API handler).
var (
	ErrAmountNotPositive          = errors.New("transaction amount must be positive")
	ErrSourceAccountNotFound      = errors.New("source account not found or does not belong to the user")
	ErrDestinationAccountRequired = errors.New("destination account is required for a transfer")
	ErrSameAccounts               = errors.New("source and destination accounts cannot be the same")
	ErrDestinationAccountNotFound = errors.New("destination account not found or does not belong to the user")
	ErrNewAccountNotFound         = errors.New("new account not found or does not belong to the user")
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
	// The repository's GetById method handles checking both account ID and user ID.
	_, err := s.accountRepo.GetById(ctx, tx.AccountId, tx.UserId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Return a clear, checkable error if the account is not found.
			return 0, ErrSourceAccountNotFound
		}
		// Return other unexpected database errors.
		return 0, err
	}

	// If it's a transfer, validate the destination account.
	if tx.Type == model.Transfer {
		if tx.DestinationAccountId == nil {
			return 0, ErrDestinationAccountRequired
		}
		if tx.AccountId == *tx.DestinationAccountId {
			return 0, ErrSameAccounts
		}

		// Verify the destination account also exists and belongs to the user.
		_, err := s.accountRepo.GetById(ctx, *tx.DestinationAccountId, tx.UserId)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, ErrDestinationAccountNotFound
			}
			return 0, err
		}
	}

	// If all validations pass, we can safely create the transaction.
	return s.repo.Create(ctx, tx)
}

// GetTransactionById retrieves a single transaction, ensuring it belongs to the specified user.
func (s *TransactionService) GetTransactionById(ctx context.Context, id, userId int64) (*model.Transaction, error) {
	return s.repo.GetById(ctx, id, userId)
}

// ListTransactionsByUserId lists all transactions for a specific user.
func (s *TransactionService) ListTransactionsByUserId(ctx context.Context, userId int64) ([]model.Transaction, error) {
	return s.repo.ListByUserId(ctx, userId)
}

// UpdateTransaction handles the logic for updating an entire transaction entity.
// It requires the userId to ensure authorization.
func (s *TransactionService) UpdateTransaction(ctx context.Context, id, userId int64, tx model.Transaction) (*model.Transaction, error) {
	// First, ensure the object to be updated has the correct ID and ownership.
	tx.Id = id
	tx.UserId = userId

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
	return s.repo.GetById(ctx, id, userId)
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
