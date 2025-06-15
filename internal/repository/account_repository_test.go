package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/testhelper"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// setupTestAccountRepository helper function prepares the environment for a test.
func setupTestAccountRepository(t *testing.T, testDb *sqlx.DB) (context.Context, *require.Assertions, UserRepository, AccountRepository, TransactionRepository) {
	return context.Background(), require.New(t), NewUserRepository(testDb), NewAccountRepository(testDb), NewTransactionRepository(testDb)
}

// TestAccountRepository groups all tests for the account repository.
func TestAccountRepository(t *testing.T) {
	testhelper.TruncateTables(t, testDB)

	t.Run("should correctly perform all CRUD operations", func(t *testing.T) {
		ctx, require, userRepo, accountRepo, _ := setupTestAccountRepository(t, testDB)

		userId, err := userRepo.Create(ctx, model.User{Name: "CRUD User", Email: "crud@test.com", PasswordHash: "hash"})
		require.NoError(err)

		// Test Create
		accToCreate := model.Account{UserId: userId, Name: "My Checking Account", Type: "checking", InitialBalance: decimal.NewFromInt(500)}
		createdId, err := accountRepo.Create(ctx, accToCreate)
		require.NoError(err)
		require.NotZero(createdId)

		// Test GetById
		savedAcc, err := accountRepo.GetById(ctx, createdId, userId)
		require.NoError(err)
		require.Equal("My Checking Account", savedAcc.Name)

		// Test Update
		savedAcc.Name = "My Updated Checking Account"
		err = accountRepo.Update(ctx, *savedAcc)
		require.NoError(err)
		updatedAcc, err := accountRepo.GetById(ctx, createdId, userId)
		require.NoError(err)
		require.Equal("My Updated Checking Account", updatedAcc.Name)

		// Test Delete
		err = accountRepo.Delete(ctx, createdId, userId)
		require.NoError(err)
		_, err = accountRepo.GetById(ctx, createdId, userId)
		require.ErrorIs(err, sql.ErrNoRows)
	})

	t.Run("should enforce database constraints and security", func(t *testing.T) {
		ctx, require, userRepo, accountRepo, txRepo := setupTestAccountRepository(t, testDB)

		userA_Id, _ := userRepo.Create(ctx, model.User{Name: "User A", Email: "usera@test.com", PasswordHash: "hash"})
		userB_Id, _ := userRepo.Create(ctx, model.User{Name: "User B", Email: "userb@test.com", PasswordHash: "hash"})
		accountB_Id, _ := accountRepo.Create(ctx, model.Account{UserId: userB_Id, Name: "User B Account", Type: "checking"})

		t.Run("security: user A cannot get user B's account", func(t *testing.T) {
			_, err := accountRepo.GetById(ctx, accountB_Id, userA_Id)
			require.ErrorIs(err, sql.ErrNoRows, "should return not found when trying to get another user's account")
		})

		t.Run("security: user A cannot update user B's account", func(t *testing.T) {
			accountToUpdate := model.Account{Id: accountB_Id, UserId: userA_Id, Name: "Hacked Name"} // User A tries to update User B's account
			err := accountRepo.Update(ctx, accountToUpdate)
			require.ErrorIs(err, sql.ErrNoRows, "should return not found when trying to update another user's account")
		})

		t.Run("security: user A cannot delete user B's account", func(t *testing.T) {
			err := accountRepo.Delete(ctx, accountB_Id, userA_Id)
			require.ErrorIs(err, sql.ErrNoRows, "should return not found when trying to delete another user's account")
		})

		t.Run("constraint: should fail to delete account with transactions", func(t *testing.T) {
			accId, _ := accountRepo.Create(ctx, model.Account{UserId: userA_Id, Name: "Account With Tx", Type: "checking"})
			_, _ = txRepo.Create(ctx, model.Transaction{UserId: userA_Id, Description: "A transaction", AccountId: accId, Amount: decimal.NewFromInt(10), Type: "expense", Date: time.Now()})

			err := accountRepo.Delete(ctx, accId, userA_Id)

			require.Error(err)
			require.Contains(err.Error(), "violates foreign key constraint", "should fail because of the FK constraint on transactions table")
		})
	})

	t.Run("should handle listing edge cases", func(t *testing.T) {
		ctx, require, userRepo, accountRepo, _ := setupTestAccountRepository(t, testDB)

		t.Run("list for a user with no accounts", func(t *testing.T) {
			userWithNoAccountsId, _ := userRepo.Create(ctx, model.User{Name: "No Accounts User", Email: "noaccounts@test.com", PasswordHash: "hash"})

			accounts, err := accountRepo.ListByUserId(ctx, userWithNoAccountsId)

			require.NoError(err)
			require.Len(accounts, 0, "slice of accounts should be empty")
		})
	})

}

func TestAccountRepositoryGetCurrentBalance(t *testing.T) {
	ctx, require, userRepo, accountRepo, txRepo := setupTestAccountRepository(t, testDB)

	t.Run("with all transaction types", func(t *testing.T) {
		userId, err := userRepo.Create(ctx, model.User{Name: "Balance User", Email: "balance@test.com", PasswordHash: "hash"})
		require.NoError(err)

		// Cria duas contas para o usuário
		accountA_Id, err := accountRepo.Create(ctx, model.Account{UserId: userId, Name: "Conta Corrente", Type: "checking", InitialBalance: decimal.NewFromFloat(1000.00)})
		require.NoError(err)

		accountB_Id, err := accountRepo.Create(ctx, model.Account{UserId: userId, Name: "Cartão Salário", Type: "savings", InitialBalance: decimal.NewFromFloat(500.00)})
		require.NoError(err)

		// Cria as transações
		// 1. Receita de R$ 200 na Conta Corrente
		_, err = txRepo.Create(ctx, model.Transaction{UserId: userId, Description: "Salário", Amount: decimal.NewFromFloat(200.0), Type: model.Income, AccountId: accountA_Id, Date: time.Now()})
		require.NoError(err)

		// 2. Despesa de R$ 50 na Conta Corrente
		_, err = txRepo.Create(ctx, model.Transaction{UserId: userId, Description: "Almoço", Amount: decimal.NewFromFloat(50.0), Type: model.Expense, AccountId: accountA_Id, Date: time.Now()})
		require.NoError(err)

		// 3. Transferência de R$ 300 da Conta Corrente para a Conta Salário
		_, err = txRepo.Create(ctx, model.Transaction{UserId: userId, Description: "Resgate", Amount: decimal.NewFromFloat(300.0), Type: model.Transfer, AccountId: accountA_Id, DestinationAccountId: &accountB_Id, Date: time.Now()})
		require.NoError(err)

		// ACT: Executar a função que queremos testar
		balanceA, err := accountRepo.GetCurrentBalance(ctx, accountA_Id, userId)
		require.NoError(err)

		balanceB, err := accountRepo.GetCurrentBalance(ctx, accountB_Id, userId)
		require.NoError(err)

		// ASSERT: Verificar se os resultados são os esperados
		// Saldo Conta A = 1000 (inicial) + 200 (receita) - 50 (despesa) - 300 (transferência enviada) = 950
		expectedBalanceA := decimal.NewFromFloat(850.00)

		// Saldo Conta B = 500 (inicial) + 300 (transferência recebida) = 800
		expectedBalanceB := decimal.NewFromFloat(800.00)

		require.True(expectedBalanceA.Equal(balanceA), fmt.Sprintf("Expected Balance A to be %s, but got %s", expectedBalanceA.String(), balanceA.String()))
		require.True(expectedBalanceB.Equal(balanceB), fmt.Sprintf("Expected Balance B to be %s, but got %s", expectedBalanceB.String(), balanceB.String()))

	})
	t.Run("no transactions", func(t *testing.T) {
		userId, err := userRepo.Create(ctx, model.User{Name: "No-Tx User", Email: "notx@test.com", PasswordHash: "hash"})
		require.NoError(err)

		initialBalance := decimal.NewFromFloat(123.45)
		accountID, _ := accountRepo.Create(ctx, model.Account{UserId: userId, Name: "Conta Parada", Type: "savings", InitialBalance: initialBalance})

		currentBalance, err := accountRepo.GetCurrentBalance(ctx, accountID, userId)

		require.NoError(err)
		require.True(initialBalance.Equal(currentBalance), "Expected current balance to be equal to initial balance")
	})
}
