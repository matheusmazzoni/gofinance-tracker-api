package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockCategoryRepository struct {
	mock.Mock
}

func (m *MockCategoryRepository) Create(ctx context.Context, cat model.Category) (int64, error) {
	args := m.Called(ctx, cat)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCategoryRepository) GetById(ctx context.Context, id, userId int64) (*model.Category, error) {
	args := m.Called(ctx, id, userId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Category), args.Error(1)
}

func (m *MockCategoryRepository) GetByName(ctx context.Context, name string, userId int64) (*model.Category, error) {
	args := m.Called(ctx, name, userId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Category), args.Error(1)
}

func (m *MockCategoryRepository) ListByUserId(ctx context.Context, userId int64) ([]model.Category, error) {
	args := m.Called(ctx, userId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Category), args.Error(1)
}

func (m *MockCategoryRepository) Update(ctx context.Context, cat model.Category) error {
	args := m.Called(ctx, cat)
	return args.Error(0)
}

func (m *MockCategoryRepository) Delete(ctx context.Context, id, userId int64) error {
	args := m.Called(ctx, id, userId)
	return args.Error(0)
}

func TestCategoryService(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.Disabled)
	ctx := context.Background()

	// Setup helper to avoid code repetition in each test case.
	setup := func() (*CategoryService, *MockCategoryRepository, *MockTransactionRepository) {
		mockRepo := new(MockCategoryRepository)
		mockTransactionRepo := new(MockTransactionRepository)
		categoryService := NewCategoryService(mockRepo, mockTransactionRepo)
		return categoryService, mockRepo, mockTransactionRepo
	}

	t.Run("CreateCategory", func(t *testing.T) {
		// Arrange
		categoryService, mockRepo, _ := setup()
		catToCreate := model.Category{UserId: int64(1), Name: "Groceries", Type: model.Expense}

		mockRepo.On("GetByName", ctx, catToCreate.Name, catToCreate.UserId).Return(nil, sql.ErrNoRows).Once()
		mockRepo.On("Create", ctx, catToCreate).Return(int64(1), nil).Once()

		// Act
		id, err := categoryService.CreateCategory(ctx, catToCreate)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, int64(1), id)
		mockRepo.AssertExpectations(t)
	})

	t.Run("GetCategoryById", func(t *testing.T) {
		t.Run("should return category when found", func(t *testing.T) {
			// Arrange
			categoryService, mockRepo, _ := setup()
			expectedCategory := &model.Category{Id: 1, UserId: 1, Name: "Rent"}
			mockRepo.On("GetById", ctx, int64(1), int64(1)).Return(expectedCategory, nil).Once()

			// Act
			category, err := categoryService.GetCategoryById(ctx, 1, 1)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, expectedCategory, category)
			mockRepo.AssertExpectations(t)
		})

		t.Run("should return error when not found", func(t *testing.T) {
			// Arrange
			categoryService, mockRepo, _ := setup()

			mockRepo.On("GetById", ctx, int64(99), int64(1)).Return(nil, sql.ErrNoRows).Once()

			// Act
			category, err := categoryService.GetCategoryById(ctx, 99, 1)

			// Assert
			assert.Error(t, err)
			assert.Nil(t, category)
			assert.ErrorIs(t, err, sql.ErrNoRows)
			mockRepo.AssertExpectations(t)
		})
	})

	t.Run("ListCategoriesByUserId", func(t *testing.T) {
		// Arrange
		categoryService, mockRepo, _ := setup()

		expectedCategories := []model.Category{
			{Id: 1, Name: "Food"},
			{Id: 2, Name: "Transport"},
		}
		mockRepo.On("ListByUserId", ctx, int64(1)).Return(expectedCategories, nil).Once()

		// Act
		categories, err := categoryService.ListCategoriesByUserId(ctx, 1)

		// Assert
		assert.NoError(t, err)
		assert.Len(t, categories, 2)
		assert.Equal(t, "Food", categories[0].Name)
		mockRepo.AssertExpectations(t)
	})

	t.Run("UpdateCategory", func(t *testing.T) {
		// Arrange
		categoryService, mockRepo, _ := setup()

		catToUpdate := model.Category{Name: "Utilities"}
		catId := int64(5)
		userId := int64(1)
		expectedCatAfterUpdate := &model.Category{Id: catId, UserId: userId, Name: "Utilities"}

		// FIX: Simulate that the new name "Utilities" is NOT taken by another category.
		mockRepo.On("GetByName", ctx, catToUpdate.Name, userId).Return(nil, sql.ErrNoRows).Once()
		// Now, expect the Update and subsequent GetById calls to happen.
		mockRepo.On("Update", ctx, mock.AnythingOfType("model.Category")).Return(nil).Once()
		mockRepo.On("GetById", ctx, catId, userId).Return(expectedCatAfterUpdate, nil).Once()

		// Act
		updatedCat, err := categoryService.UpdateCategory(ctx, catId, userId, catToUpdate)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedCatAfterUpdate, updatedCat)
		mockRepo.AssertExpectations(t)
	})

	t.Run("DeleteCategory", func(t *testing.T) {
		t.Run("should delete when category is not in use", func(t *testing.T) {
			// Arrange
			categoryService, mockRepo, mockTransactionRepo := setup()
			catId := int64(7)
			userId := int64(1)

			// Now, expect the Delete call to happen.
			mockRepo.On("Delete", ctx, catId, userId).Return(nil).Once()

			// Act
			err := categoryService.DeleteCategory(ctx, catId, userId)

			// Assert
			assert.NoError(t, err)
			mockRepo.AssertExpectations(t)
			mockTransactionRepo.AssertExpectations(t)
		})
	})
}
