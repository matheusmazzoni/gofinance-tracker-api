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

// setupTestUser is a helper for user tests
func setupTestUser(t *testing.T, testDB *sqlx.DB) (context.Context, *require.Assertions, UserRepository, AccountRepository, TransactionRepository) {
	return context.Background(), require.New(t), NewUserRepository(testDB), NewAccountRepository(testDB), NewTransactionRepository(testDB)
}

func TestUserRepository(t *testing.T) {
	testhelper.TruncateTables(t, testDB)

	t.Run("should create and get a user", func(t *testing.T) {
		ctx, require, userRepo, _, _ := setupTestUser(t, testDB)

		// Arrange
		userToCreate := model.User{
			Name:         "John Doe",
			Email:        "john.doe@example.com",
			PasswordHash: "some_strong_hash",
		}

		// Act: Create
		createdId, err := userRepo.Create(ctx, userToCreate)
		require.NoError(err)
		require.NotZero(createdId)

		// Assert: GetByEmail
		foundUser, err := userRepo.GetByEmail(ctx, "john.doe@example.com")
		require.NoError(err)
		require.Equal("John Doe", foundUser.Name)
		require.Equal(createdId, foundUser.Id)

		// Assert: GetById
		foundUserById, err := userRepo.GetById(ctx, createdId)
		require.NoError(err)
		require.Equal("John Doe", foundUserById.Name)
	})

	t.Run("should fail to create user with duplicate email", func(t *testing.T) {
		ctx, require, userRepo, _, _ := setupTestUser(t, testDB)

		// Arrange
		user := model.User{Name: "Jane Doe", Email: "jane.doe@example.com", PasswordHash: "hash"}
		_, err := userRepo.Create(ctx, user)
		require.NoError(err, "first creation should succeed")

		// Act: Try to create another user with the same email
		user2 := model.User{Name: "Jane Smith", Email: "jane.doe@example.com", PasswordHash: "hash2"}
		_, err = userRepo.Create(ctx, user2)

		// Assert
		require.Error(err)
		require.Contains(err.Error(), "violates unique constraint")
	})

	t.Run("get by email should return error for non-existent user", func(t *testing.T) {
		ctx, require, userRepo, _, _ := setupTestUser(t, testDB)

		// Act
		_, err := userRepo.GetByEmail(ctx, "nonexistent@example.com")

		// Assert
		require.Error(err)
		require.ErrorIs(err, sql.ErrNoRows)
	})
}
