package service

import (
	"context"
	"testing"
	"time"

	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockBudgetRepository struct {
	mock.Mock
}

func (m *MockBudgetRepository) Create(ctx context.Context, budget model.Budget) (int64, error) {
	args := m.Called(ctx, budget)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockBudgetRepository) GetById(ctx context.Context, id, userId int64) (*model.Budget, error) {
	args := m.Called(ctx, id, userId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Budget), args.Error(1)
}

func (m *MockBudgetRepository) ListByUserAndPeriod(ctx context.Context, userId int64, month, year int) ([]model.Budget, error) {
	args := m.Called(ctx, userId, month, year)

	data := args.Get(0)
	err := args.Error(1)

	if data == nil {
		return nil, err
	}

	return data.([]model.Budget), err
}

func (m *MockBudgetRepository) Update(ctx context.Context, budget model.Budget) error {
	args := m.Called(ctx, budget)
	return args.Error(0)
}

func (m *MockBudgetRepository) Delete(ctx context.Context, id, userId int64) error {
	// Note: Fixed a bug here. Original was m.Called(ctx, userId, userId).
	args := m.Called(ctx, id, userId)
	return args.Error(0)
}

func TestBudgetService(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.Disabled)
	ctx := context.Background()

	t.Run("CreateBudget", func(t *testing.T) {
		mockBudgetRepo := new(MockBudgetRepository)
		mockCategoryRepo := new(MockCategoryRepository)

		// This service doesn't use txRepo in the CreateBudget method, so we can pass nil
		budgetService := NewBudgetService(mockBudgetRepo, mockCategoryRepo, nil)

		t.Run("should fail if category is not an expense type", func(t *testing.T) {
			// Arrange
			budgetToCreate := model.Budget{UserId: 1, CategoryId: 1}
			// Simulate the repository returning an "income" category
			incomeCategory := &model.Category{Id: 1, Type: model.Income}
			mockCategoryRepo.On("GetById", ctx, int64(1), int64(1)).Return(incomeCategory, nil).Once()

			// Act
			_, err := budgetService.CreateBudget(ctx, budgetToCreate)

			// Assert
			assert.Error(t, err)
			assert.Equal(t, "budgets can only be set for expense categories", err.Error())
			mockBudgetRepo.AssertNotCalled(t, "Create") // Ensure we failed before trying to save
		})
	})

	t.Run("ListEnrichedBudgetsByPeriod", func(t *testing.T) {
		t.Run("should correctly calculate spent and balance amounts", func(t *testing.T) {
			// Arrange
			mockBudgetRepo := new(MockBudgetRepository)
			mockTxRepo := new(MockTransactionRepository)
			// Pass the mock for transactionRepo to the service
			budgetService := NewBudgetService(mockBudgetRepo, nil, mockTxRepo)

			ctx := context.Background()
			userId, month, year := int64(1), 6, 2025
			startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
			endDate := startDate.AddDate(0, 1, 0)

			// 1. Mock the list of budgets returned from the database
			budgetsFromRepo := []model.Budget{
				{Id: 1, UserId: userId, CategoryId: 10, Amount: decimal.NewFromInt(800)},
			}
			mockBudgetRepo.On("ListByUserAndPeriod", ctx, userId, month, year).Return(budgetsFromRepo, nil).Once()

			// --- THIS IS THE CRUCIAL PART ---
			// 2. Mock the spent amount calculation.
			// We MUST create a `decimal.Decimal` object, not use a raw float like 550.50.
			expectedSpentAmount := decimal.NewFromFloat(550.50)

			// We pass this decimal.Decimal object to the Return() function.
			mockTxRepo.On("SumExpensesByCategoryAndPeriod", ctx, userId, int64(10), startDate, endDate).Return(expectedSpentAmount, nil).Once()
			// --- END CRUCIAL PART ---

			// Act
			enrichedBudgets, err := budgetService.ListEnrichedBudgetsByPeriod(ctx, userId, month, year)

			// Assert
			require.NoError(t, err)
			require.Len(t, enrichedBudgets, 1)

			foodBudget := enrichedBudgets[0]
			expectedBudgetedAmount := decimal.NewFromInt(800)
			expectedBalance := decimal.NewFromFloat(249.50) // 800 - 550.50

			assert.True(t, expectedBudgetedAmount.Equal(foodBudget.Amount), "budgeted amount should be correct")
			assert.True(t, expectedSpentAmount.Equal(foodBudget.SpentAmount), "spent amount should be calculated correctly")
			assert.True(t, expectedBalance.Equal(foodBudget.Balance), "remaining balance should be calculated correctly")

			mockBudgetRepo.AssertExpectations(t)
			mockTxRepo.AssertExpectations(t)
		})
	})
}
