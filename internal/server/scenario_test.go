package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/api/dto"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/repository"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/testhelper"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE: This test file assumes a `setup_test.go` file exists in this package
// that provides the `testDB` global variable via TestMain.

// TestBusinessScenarios validates complex, multi-step user workflows.
func TestBusinessScenarios(t *testing.T) {
	require := require.New(t)
	gin.SetMode(gin.TestMode)

	server := NewServer(testCfg, testDB, &testLogger)

	ctx := context.Background()
	userRepo := repository.NewUserRepository(testDB)

	t.Run("Scenario: 'I Messed Up My Entry' Correction Flow", func(t *testing.T) {
		// This test simulates a user creating, editing, and deleting a transaction
		// and verifies the account balance is correctly recalculated at each step.
		testhelper.TruncateTables(t, testDB)

		// Arrange 1: Create a user and an account with an initial balance.
		userID, _ := userRepo.Create(ctx, model.User{Name: "Correction User", Email: "correction@test.com", PasswordHash: "hash"})
		token := generateTestToken(t, userID, testCfg.JWTSecretKey)
		accountID := createAccount(t, server.router, token, "Checking Account", "checking", "500.00")

		// Assert 1: Initial balance is correct.
		balance := getAccountBalance(t, server.router, token, accountID)
		require.True(decimal.NewFromInt(500).Equal(balance))

		// Act 1: Create an incorrect expense of $100
		expenseID := createTransaction(t, server.router, token, accountID, "Dinner", "expense", "100.00")

		// Assert 2: Balance is updated correctly after creation.
		balance = getAccountBalance(t, server.router, token, accountID)
		require.Equal(decimal.NewFromInt(400), balance, "Balance should be 400 after $100 expense")

		// Act 2: Edit the expense to the correct amount of $80.
		updateTransaction(t, server.router, token, expenseID, accountID, "Dinner (Corrected)", "expense", "80.00")

		// Assert 3: Balance is recalculated correctly after update.
		balance = getAccountBalance(t, server.router, token, accountID)
		require.Equal(decimal.NewFromInt(420), balance, "Balance should be $420 after correction to $80")

		// Act 3: Delete the expense entirely.
		deleteTransaction(t, server.router, token, expenseID)

		// Assert 4: Balance returns to its original state.
		balance = getAccountBalance(t, server.router, token, accountID)
		require.True(decimal.NewFromInt(500).Equal(balance), "Balance should return to 500 after deletion")
	})

	t.Run("Scenario: Credit Card Payment Workflow", func(t *testing.T) {
		// This test simulates paying off a credit card balance from a checking account.
		testhelper.TruncateTables(t, testDB)

		// Arrange: Create user, a checking account with $2000, and a credit card with $0.
		userID, _ := userRepo.Create(ctx, model.User{Name: "Payment User", Email: "payment@test.com", PasswordHash: "hash"})
		token := generateTestToken(t, userID, testCfg.JWTSecretKey)
		checkingID := createAccount(t, server.router, token, "My Checking", "checking", "2000.00")
		creditCardID := createAccount(t, server.router, token, "My Credit Card", "credit_card", "0.00")

		// Arrange: Add $450 in expenses to the credit card.
		createTransaction(t, server.router, token, creditCardID, "Groceries", "expense", "150.00")
		createTransaction(t, server.router, token, creditCardID, "Online Shopping", "expense", "300.00")

		// Assert 1: Verify pre-payment balances.
		require.True(decimal.NewFromInt(2000).Equal(getAccountBalance(t, server.router, token, checkingID)))
		require.True(decimal.NewFromInt(-450).Equal(getAccountBalance(t, server.router, token, creditCardID)), "Credit card balance should be negative after expenses")

		// Act: Pay the credit card bill by transferring $450 from Checking to Credit Card.
		createTransfer(t, server.router, token, "CC Payment", "450.00", checkingID, creditCardID)

		// Assert 2: Verify final balances.
		assert.True(t, decimal.NewFromInt(1550).Equal(getAccountBalance(t, server.router, token, checkingID)), "Checking account balance should decrease")
		assert.True(t, decimal.Zero.Equal(getAccountBalance(t, server.router, token, creditCardID)), "Credit card balance should be zero after payment")
	})

	t.Run("Scenario: Application-Level Cascade Delete", func(t *testing.T) {
		// This test verifies that deleting an account successfully deletes all associated transactions.
		testhelper.TruncateTables(t, testDB)

		// Arrange: Create user and multiple accounts/transactions.
		userID, _ := userRepo.Create(ctx, model.User{Name: "Cascade User", Email: "cascade@test.com", PasswordHash: "hash"})
		token := generateTestToken(t, userID, testCfg.JWTSecretKey)

		accountToDeleteID := createAccount(t, server.router, token, "Account To Delete", "checking", "100")
		otherAccountID := createAccount(t, server.router, token, "Other Account", "savings", "200")

		tx1ID := createTransaction(t, server.router, token, accountToDeleteID, "Expense from deleted account", "expense", "10")
		tx2ID := createTransfer(t, server.router, token, "Transfer from deleted account", "20", accountToDeleteID, otherAccountID)
		tx3ID := createTransaction(t, server.router, token, otherAccountID, "Expense from other account", "expense", "30")

		// Act: Delete the account that has history.
		deleteAccount(t, server.router, token, accountToDeleteID)

		// Assert: Verify resources were deleted or preserved correctly.
		// 1. The account itself is gone.
		assertAccountNotFound(t, server.router, token, accountToDeleteID)
		// 2. Transactions linked to the deleted account are gone.
		assertTransactionNotFound(t, server.router, token, tx1ID)
		assertTransactionNotFound(t, server.router, token, tx2ID)
		// 3. The unrelated transaction still exists.
		assertTransactionFound(t, server.router, token, tx3ID)
	})
}

// =============================================================================
// API HELPER FUNCTIONS FOR TESTS
// =============================================================================

func createAccount(t *testing.T, router *gin.Engine, token, name, accType, initialBalance string) int64 {
	body := fmt.Sprintf(`{"name": "%s", "type": "%s", "initial_balance": "%s"}`, name, accType, initialBalance)
	req, _ := http.NewRequest("POST", "/v1/accounts", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusCreated, recorder.Code)
	var resp dto.CreateAccountResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	return resp.Id
}

func getAccountBalance(t *testing.T, router *gin.Engine, token string, accountID int64) decimal.Decimal {
	req, _ := http.NewRequest("GET", fmt.Sprintf("/v1/accounts/%d", accountID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusOK, recorder.Code)
	var resp dto.AccountResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	fmt.Print(resp)
	return resp.Balance
}

func createTransaction(t *testing.T, router *gin.Engine, token string, accountId int64, description, txType, amount string) int64 {
	body := fmt.Sprintf(`{"description":"%s", "amount":"%s", "type":"%s", "account_id": %d, "date": "%s"}`, description, amount, txType, accountId, time.Now().Format(time.RFC3339))
	req, _ := http.NewRequest("POST", "/v1/transactions", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	fmt.Print(recorder.Body)
	require.Equal(t, http.StatusCreated, recorder.Code)
	var resp dto.TransactionResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	return resp.Id
}

func updateTransaction(t *testing.T, router *gin.Engine, token string, txID, accountId int64, description, txType, amount string) {
	updateDTO := dto.UpdateTransactionRequest{
		Description: description,
		Amount:      decimal.RequireFromString(amount),
		Date:        time.Now(),
		Type:        model.TransactionType(txType),
		AccountId:   accountId,
		CategoryId:  nil,
	}
	body, _ := json.Marshal(updateDTO)
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/v1/transactions/%d", txID), bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	fmt.Print(recorder.Body)
	require.Equal(t, http.StatusOK, recorder.Code)
}

func deleteTransaction(t *testing.T, router *gin.Engine, token string, txID int64) {
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/v1/transactions/%d", txID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusNoContent, recorder.Code)
}

func createTransfer(t *testing.T, router *gin.Engine, token, description, amount string, fromAccountID, toAccountID int64) int64 {
	body := fmt.Sprintf(`{"description":"%s", "amount":"%s", "type":"transfer", "account_id": %d, "destination_account_id": %d, "date": "%s"}`, description, amount, fromAccountID, toAccountID, time.Now().Format(time.RFC3339))
	req, _ := http.NewRequest("POST", "/v1/transactions", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusCreated, recorder.Code)
	var resp dto.TransactionResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	return resp.Id
}

func deleteAccount(t *testing.T, router *gin.Engine, token string, accountID int64) {
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/v1/accounts/%d", accountID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusNoContent, recorder.Code)
}

func assertAccountNotFound(t *testing.T, router *gin.Engine, token string, accountID int64) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("/v1/accounts/%d", accountID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func assertTransactionNotFound(t *testing.T, router *gin.Engine, token string, transactionID int64) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("/v1/transactions/%d", transactionID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func assertTransactionFound(t *testing.T, router *gin.Engine, token string, transactionID int64) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("/v1/transactions/%d", transactionID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusOK, recorder.Code)
}
