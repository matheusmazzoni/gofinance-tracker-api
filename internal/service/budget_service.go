package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/repository"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

// BudgetService encapsulates the business logic for budgets.
type BudgetService struct {
	budgetRepo      repository.BudgetRepository
	categoryRepo    repository.CategoryRepository
	transactionRepo repository.TransactionRepository
}

// EnrichedBudget is a struct that holds the budget and its calculated spending.
type EnrichedBudget struct {
	model.Budget
	SpentAmount decimal.Decimal
	Balance     decimal.Decimal
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

// CreateBudget handles logic for creating a new budget with validation.
func (s *BudgetService) CreateBudget(ctx context.Context, budget model.Budget) (*model.Budget, error) {
	// Business logic: Check if the category exists and belongs to the user.
	category, err := s.categoryRepo.GetById(ctx, budget.CategoryId, budget.UserId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("category not found for this user")
		}
		return nil, err
	}

	// Business logic: Enforce that budgets can only be set for 'expense' categories.
	if category.Type != model.Expense {
		return nil, errors.New("budgets can only be set for expense categories")
	}

	id, err := s.budgetRepo.Create(ctx, budget)
	if err != nil {
		return nil, err
	}
	return s.budgetRepo.GetById(ctx, id, budget.UserId)
}

// GetEnrichedBudgetById retrieves a single budget and calculates its spending.
func (s *BudgetService) GetEnrichedBudgetById(ctx context.Context, id, userId int64) (*EnrichedBudget, error) {
	budget, err := s.budgetRepo.GetById(ctx, id, userId)
	if err != nil {
		return nil, err
	}

	// Calculate start and end dates for the budget's month
	startDate := time.Date(budget.Year, time.Month(budget.Month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0) // First day of the next month

	// Calculate spent amount
	spent, err := s.transactionRepo.SumExpensesByCategoryAndPeriod(ctx, userId, budget.CategoryId, startDate, endDate)
	if err != nil {
		log.Error().Err(err).Msg("Failed to calculate spent amount for budget")
		return nil, err
	}

	enrichedBudget := &EnrichedBudget{
		Budget:      *budget,
		SpentAmount: spent,
		Balance:     budget.Amount.Sub(spent), // Balance = Amount - Spent
	}

	return enrichedBudget, nil
}

// ListEnrichedBudgetsByPeriod retrieves all budgets for a user in a given period, with calculated spending.
func (s *BudgetService) ListEnrichedBudgetsByPeriod(ctx context.Context, userId int64, month, year int) ([]EnrichedBudget, error) {
	budgets, err := s.budgetRepo.ListByUserAndPeriod(ctx, userId, month, year)
	if err != nil {
		return nil, err
	}

	var enrichedBudgets []EnrichedBudget
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)

	for _, budget := range budgets {
		spent, err := s.transactionRepo.SumExpensesByCategoryAndPeriod(ctx, userId, budget.CategoryId, startDate, endDate)
		if err != nil {
			log.Warn().Err(err).Int64("budget_id", budget.Id).Msg("Failed to get spent amount for budget in list")
			spent = decimal.Zero // Default to zero if calculation fails for one item
		}

		enrichedBudgets = append(enrichedBudgets, EnrichedBudget{
			Budget:      budget,
			SpentAmount: spent,
			Balance:     budget.Amount.Sub(spent),
		})
	}

	return enrichedBudgets, nil
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
