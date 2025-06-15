package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/testhelper"
	"github.com/stretchr/testify/require"
)

// setupTestCategory is a helper for category tests
func setupTestCategory(t *testing.T, testDB *sqlx.DB) (context.Context, *require.Assertions, UserRepository, AccountRepository, TransactionRepository, CategoryRepository) {
	return context.Background(), require.New(t), NewUserRepository(testDB), NewAccountRepository(testDB), NewTransactionRepository(testDB), NewCategoryRepository(testDB)
}
func TestCategoryRepository(t *testing.T) {
	testhelper.TruncateTables(t, testDB)

	t.Run("should correctly perform CRUD operations", func(t *testing.T) {
		ctx, require, userRepo, _, _, categoryRepo := setupTestCategory(t, testDB)

		// Arrange
		userId, _ := userRepo.Create(ctx, model.User{Name: "Category User", Email: "cat@test.com", PasswordHash: "hash"})
		var createdCatId int64

		// Act: Create
		catToCreate := model.Category{UserId: userId, Name: "Groceries", Type: model.Expense}
		createdCatId, err := categoryRepo.Create(ctx, catToCreate)
		require.NoError(err)
		require.NotZero(createdCatId)

		// Assert: GetById
		savedCat, err := categoryRepo.GetById(ctx, createdCatId, userId)
		require.NoError(err)
		require.Equal("Groceries", savedCat.Name)

		// Act: Update
		savedCat.Name = "Food & Groceries"
		err = categoryRepo.Update(ctx, *savedCat)
		require.NoError(err)
		updatedCat, _ := categoryRepo.GetById(ctx, createdCatId, userId)
		require.Equal("Food & Groceries", updatedCat.Name)

		// Act: Delete
		err = categoryRepo.Delete(ctx, createdCatId, userId)
		require.NoError(err)
		_, err = categoryRepo.GetById(ctx, createdCatId, userId)
		require.ErrorIs(err, sql.ErrNoRows)
	})

	t.Run("should enforce unique constraint on (user_id, name)", func(t *testing.T) {
		ctx, require, userRepo, _, _, categoryRepo := setupTestCategory(t, testDB)

		// Arrange
		userId, _ := userRepo.Create(ctx, model.User{Name: "Unique Cat User", Email: "unicat@test.com", PasswordHash: "hash"})
		cat := model.Category{UserId: userId, Name: "Unique Category"}
		_, err := categoryRepo.Create(ctx, cat)
		require.NoError(err, "first creation should succeed")

		// Act
		_, err = categoryRepo.Create(ctx, cat) // Try to create it again

		// Assert
		require.Error(err)
		require.Contains(err.Error(), "violates unique constraint")
	})
}
