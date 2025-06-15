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
	// Disable logging for tests to keep output clean
	zerolog.SetGlobalLevel(zerolog.Disabled)
	ctx := context.Background()

	t.Run("CreateCategory", func(t *testing.T) {
		// Arrange
		mockRepo := new(MockCategoryRepository)
		categoryService := NewCategoryService(mockRepo)

		catToCreate := model.Category{UserId: 1, Name: "Groceries"}
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
			mockRepo := new(MockCategoryRepository)
			categoryService := NewCategoryService(mockRepo)

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
			mockRepo := new(MockCategoryRepository)
			categoryService := NewCategoryService(mockRepo)

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
		mockRepo := new(MockCategoryRepository)
		categoryService := NewCategoryService(mockRepo)

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
		mockRepo := new(MockCategoryRepository)
		categoryService := NewCategoryService(mockRepo)

		catToUpdate := model.Category{Name: "Utilities"}
		expectedCatAfterUpdate := &model.Category{Id: 5, UserId: 1, Name: "Utilities"}

		// Setup mock expectations for the sequence of calls
		mockRepo.On("Update", ctx, mock.AnythingOfType("model.Category")).Return(nil).Once()
		mockRepo.On("GetById", ctx, int64(5), int64(1)).Return(expectedCatAfterUpdate, nil).Once()

		// Act
		updatedCat, err := categoryService.UpdateCategory(ctx, 5, 1, catToUpdate)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedCatAfterUpdate, updatedCat)
		mockRepo.AssertExpectations(t)
	})

	t.Run("DeleteCategory", func(t *testing.T) {
		// Arrange
		mockRepo := new(MockCategoryRepository)
		categoryService := NewCategoryService(mockRepo)

		// Note: The business logic to check for transactions is not yet implemented in the service.
		// This test verifies the current behavior, which is a direct call to the repository.
		mockRepo.On("Delete", ctx, int64(7), int64(1)).Return(nil).Once()

		// Act
		err := categoryService.DeleteCategory(ctx, 7, 1)

		// Assert
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}
