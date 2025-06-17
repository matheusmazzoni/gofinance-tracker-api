package service

import (
	"context"
	"errors"

	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/repository"
)

// BudgetService encapsulates the business logic for budgets.
type BudgetService struct {
	budgetRepo      repository.BudgetRepository
	categoryRepo    repository.CategoryRepository    // Needed for validations
	transactionRepo repository.TransactionRepository // Will be needed for calculations
}

// NewBudgetService creates a new instance of BudgetService.
func NewBudgetService(
	budgetRepo repository.BudgetRepository,
	categoryRepo repository.CategoryRepository,
	transactionRepo repository.TransactionRepository,
) *BudgetService {
	return &BudgetService{
		budgetRepo:      budgetRepo,
		categoryRepo:    categoryRepo,
		transactionRepo: transactionRepo,
	}
}

// CreateBudget handles logic for creating a new budget.
func (s *BudgetService) CreateBudget(ctx context.Context, budget model.Budget) (*model.Budget, error) {
	// Future business logic: Check if the category exists and belongs to the user.
	category, err := s.categoryRepo.GetById(ctx, budget.CategoryId, budget.UserId)
	if err != nil {
		return nil, errors.New("category not found for this user")
	}

	// Future business logic: Enforce budgets only for 'expense' categories.
	if category.Type != model.Expense {
		return nil, errors.New("budgets can only be set for expense categories")
	}

	id, err := s.budgetRepo.Create(ctx, budget)
	if err != nil {
		return nil, err
	}
	return s.budgetRepo.GetById(ctx, id, budget.UserId)
}

// GetBudgetById retrieves a single budget.
func (s *BudgetService) GetBudgetById(ctx context.Context, id, userId int64) (*model.Budget, error) {
	return s.budgetRepo.GetById(ctx, id, userId)
}

// ListBudgetsByPeriod retrieves all budgets for a user in a given month and year.
func (s *BudgetService) ListBudgetsByPeriod(ctx context.Context, userId int64, month, year int) ([]model.Budget, error) {
	// The complex logic of calculating spent amounts will be added here later.
	return s.budgetRepo.ListByUserAndPeriod(ctx, userId, month, year)
}

// UpdateBudget handles updating a budget's amount.
func (s *BudgetService) UpdateBudget(ctx context.Context, id, userId int64, budget model.Budget) (*model.Budget, error) {
	budget.Id = id
	budget.UserId = userId

	if err := s.budgetRepo.Update(ctx, budget); err != nil {
		return nil, err
	}

	return s.budgetRepo.GetById(ctx, id, userId)
}

// DeleteBudget handles deleting a budget.
func (s *BudgetService) DeleteBudget(ctx context.Context, id, userId int64) error {
	return s.budgetRepo.Delete(ctx, id, userId)
}
