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
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Budget), args.Error(1)
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
		mockBudgetRepo := new(MockBudgetRepository)
		mockCategoryRepo := new(MockCategoryRepository)
		mockTxRepo := new(MockTransactionRepository)
		budgetService := NewBudgetService(mockBudgetRepo, mockCategoryRepo, mockTxRepo)

		t.Run("should correctly calculate spent and balance amounts", func(t *testing.T) {
			// Arrange
			userId, month, year := int64(1), 6, 2025
			startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
			endDate := startDate.AddDate(0, 1, 0)

			// Mock the list of budgets returned from the DB
			budgetsFromRepo := []model.Budget{
				{Id: 1, UserId: userId, CategoryId: 10, Amount: decimal.NewFromInt(800)},
				{Id: 2, UserId: userId, CategoryId: 11, Amount: decimal.NewFromInt(200)},
			}
			mockBudgetRepo.On("ListByUserAndPeriod", ctx, userId, month, year).Return(budgetsFromRepo, nil).Once()

			// Mock the spent amount calculated for each category
			// For "Food" (Id 10), simulate $550 spent
			mockTxRepo.On("SumExpensesByCategoryAndPeriod", ctx, userId, int64(10), startDate, endDate).Return(decimal.NewFromFloat(550.50), nil).Once()
			// For "Transport" (Id 11), simulate $0 spent
			mockTxRepo.On("SumExpensesByCategoryAndPeriod", ctx, userId, int64(11), startDate, endDate).Return(decimal.Zero, nil).Once()

			// Act
			enrichedBudgets, err := budgetService.ListEnrichedBudgetsByPeriod(ctx, userId, month, year)

			// Assert
			assert.NoError(t, err)
			assert.Len(t, enrichedBudgets, 2)

			// Check Food budget
			assert.Equal(t, int64(10), enrichedBudgets[0].CategoryId)
			assert.True(t, decimal.NewFromInt(800).Equal(enrichedBudgets[0].Amount))           // Budgeted
			assert.True(t, decimal.NewFromFloat(550.50).Equal(enrichedBudgets[0].SpentAmount)) // Spent
			assert.True(t, decimal.NewFromFloat(249.50).Equal(enrichedBudgets[0].Balance))     // Remaining

			// Check Transport budget
			assert.Equal(t, int64(11), enrichedBudgets[1].CategoryId)
			assert.True(t, decimal.NewFromInt(200).Equal(enrichedBudgets[1].Amount))  // Budgeted
			assert.True(t, decimal.Zero.Equal(enrichedBudgets[1].SpentAmount))        // Spent
			assert.True(t, decimal.NewFromInt(200).Equal(enrichedBudgets[1].Balance)) // Remaining

			mockBudgetRepo.AssertExpectations(t)
			mockTxRepo.AssertExpectations(t)
		})
	})
}
