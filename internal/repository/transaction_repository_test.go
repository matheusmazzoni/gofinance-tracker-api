package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/testhelper"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// setupTestTransaction is a helper for user tests
func setupTestTransaction(t *testing.T, testDB *sqlx.DB) (context.Context, *require.Assertions, UserRepository, AccountRepository, TransactionRepository) {
	return context.Background(), require.New(t), NewUserRepository(testDB), NewAccountRepository(testDB), NewTransactionRepository(testDB)
}

func TestTransactionRepository(t *testing.T) {
	testhelper.TruncateTables(t, testDB)

	t.Run("should correctly perform CRUD operations", func(t *testing.T) {
		ctx, require, userRepo, accountRepo, txRepo := setupTestTransaction(t, testDB)

		// Arrange: Create a user and account for the transaction.
		userId, _ := userRepo.Create(ctx, model.User{Name: "Tx CRUD User", Email: "txcrud@test.com", PasswordHash: "hash"})
		accountId, _ := accountRepo.Create(ctx, model.Account{UserId: userId, Name: "Main Account", Type: "checking"})
		var createdTxId int64

		// 1. Test Create
		t.Run("create transaction", func(t *testing.T) {
			txToCreate := model.Transaction{
				UserId:      userId,
				AccountId:   accountId,
				Description: "Weekly Groceries",
				Amount:      decimal.NewFromFloat(150.75),
				Type:        model.Expense,
				Date:        time.Now(),
			}
			var err error
			createdTxId, err = txRepo.Create(ctx, txToCreate)
			require.NoError(err)
			require.NotZero(createdTxId)
		})

		// 2. Test GetById
		t.Run("get transaction by id", func(t *testing.T) {
			require.NotZero(createdTxId, "Create test must run first")

			savedTx, err := txRepo.GetById(ctx, createdTxId, userId)
			require.NoError(err)
			require.Equal("Weekly Groceries", savedTx.Description)
			require.True(decimal.NewFromFloat(150.75).Equal(savedTx.Amount))
		})

		// 3. Test Update
		t.Run("update transaction", func(t *testing.T) {
			require.NotZero(createdTxId, "Create test must run first")

			txToUpdate, _ := txRepo.GetById(ctx, createdTxId, userId)
			txToUpdate.Description = "Updated Groceries Description"

			err := txRepo.Update(ctx, *txToUpdate)
			require.NoError(err)

			updatedTx, _ := txRepo.GetById(ctx, createdTxId, userId)
			require.Equal("Updated Groceries Description", updatedTx.Description)
		})

		// 4. Test Delete
		t.Run("delete transaction", func(t *testing.T) {
			require.NotZero(createdTxId, "Create test must run first")

			err := txRepo.Delete(ctx, createdTxId, userId)
			require.NoError(err)

			_, err = txRepo.GetById(ctx, createdTxId, userId)
			require.ErrorIs(err, sql.ErrNoRows)
		})
	})

	t.Run("should enforce foreign key constraints", func(t *testing.T) {
		ctx, require, userRepo, _, txRepo := setupTestTransaction(t, testDB)
		userId, _ := userRepo.Create(ctx, model.User{Name: "Tx Constraint User", Email: "txconstraint@test.com", PasswordHash: "hash"})

		t.Run("fail for non-existent account_id", func(t *testing.T) {
			tx := model.Transaction{
				UserId:      userId,
				AccountId:   99999, // This account Id does not exist
				Description: "Transaction to nowhere",
				Amount:      decimal.NewFromInt(100),
				Type:        model.Expense,
				Date:        time.Now(),
			}
			_, err := txRepo.Create(ctx, tx)
			require.Error(err)
			require.Contains(err.Error(), `violates foreign key constraint "fk_account"`)
		})
	})

	t.Run("should enforce security scope", func(t *testing.T) {
		ctx, require, userRepo, accountRepo, txRepo := setupTestTransaction(t, testDB)

		// Arrange: Create two users and a transaction for User B
		userA_Id, _ := userRepo.Create(ctx, model.User{Name: "User A", Email: "usera_txsec@test.com", PasswordHash: "hash"})
		userB_Id, _ := userRepo.Create(ctx, model.User{Name: "User B", Email: "userb_txsec@test.com", PasswordHash: "hash"})
		accountB_Id, _ := accountRepo.Create(ctx, model.Account{UserId: userB_Id, Name: "User B Account", Type: "checking"})
		tx_B_Id, _ := txRepo.Create(ctx, model.Transaction{UserId: userB_Id, AccountId: accountB_Id, Description: "User B's private transaction", Amount: decimal.NewFromInt(50), Type: "expense", Date: time.Now()})

		// Act & Assert: User A tries to access User B's transaction
		t.Run("user A cannot get user B's transaction", func(t *testing.T) {
			_, err := txRepo.GetById(ctx, tx_B_Id, userA_Id)
			require.ErrorIs(err, sql.ErrNoRows)
		})

		t.Run("user A cannot update user B's transaction", func(t *testing.T) {
			txToUpdate := model.Transaction{Id: tx_B_Id, UserId: userA_Id, Description: "Hacked", AccountId: accountB_Id, Amount: decimal.NewFromInt(1), Type: "expense", Date: time.Now()}
			err := txRepo.Update(ctx, txToUpdate)
			require.ErrorIs(err, sql.ErrNoRows)
		})

		t.Run("user A cannot delete user B's transaction", func(t *testing.T) {
			err := txRepo.Delete(ctx, tx_B_Id, userA_Id)
			require.ErrorIs(err, sql.ErrNoRows)
		})
	})

	t.Run("should correctly delete transactions by account Id", func(t *testing.T) {
		ctx, require, userRepo, accountRepo, txRepo := setupTestTransaction(t, testDB)

		// Arrange
		userId, _ := userRepo.Create(ctx, model.User{Name: "Cascade User", Email: "cascade@test.com", PasswordHash: "hash"})
		accountA_Id, _ := accountRepo.Create(ctx, model.Account{UserId: userId, Name: "Account A", Type: "checking"})
		accountB_Id, _ := accountRepo.Create(ctx, model.Account{UserId: userId, Name: "Account B", Type: "checking"})
		accountC_Id, _ := accountRepo.Create(ctx, model.Account{UserId: userId, Name: "Account C", Type: "checking"})

		// Create transactions involving Account A
		tx1_Id, _ := txRepo.Create(ctx, model.Transaction{UserId: userId, AccountId: accountA_Id, Description: "Expense from A", Amount: decimal.NewFromInt(10), Type: "expense", Date: time.Now()})
		tx2_Id, _ := txRepo.Create(ctx, model.Transaction{UserId: userId, AccountId: accountB_Id, DestinationAccountId: &accountA_Id, Description: "Transfer to A", Amount: decimal.NewFromInt(20), Type: "transfer", Date: time.Now()})
		// Create a transaction NOT involving Account A
		tx3_Id, _ := txRepo.Create(ctx, model.Transaction{UserId: userId, AccountId: accountC_Id, Description: "Expense from C", Amount: decimal.NewFromInt(30), Type: "expense", Date: time.Now()})

		// Act
		err := txRepo.DeleteByAccountId(ctx, userId, accountA_Id)
		require.NoError(err)

		// Assert
		// Transactions involving Account A should be gone
		_, err = txRepo.GetById(ctx, tx1_Id, userId)
		require.ErrorIs(err, sql.ErrNoRows)
		_, err = txRepo.GetById(ctx, tx2_Id, userId)
		require.ErrorIs(err, sql.ErrNoRows)

		// Transaction NOT involving Account A should still exist
		_, err = txRepo.GetById(ctx, tx3_Id, userId)
		require.NoError(err)
	})
}
