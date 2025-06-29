package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/repository"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTransactionRepository is a mock implementation of the TransactionRepository interface,
// used for testing the TransactionService in isolation.
type MockTransactionRepository struct {
	mock.Mock
}

// Create simulates creating a transaction in the database.
func (m *MockTransactionRepository) Create(ctx context.Context, tx model.Transaction) (int64, error) {
	args := m.Called(ctx, tx)
	return args.Get(0).(int64), args.Error(1)
}

// GetById simulates retrieving a single transaction by its Id and user Id.
func (m *MockTransactionRepository) GetById(ctx context.Context, id, userId int64) (*model.Transaction, error) {
	args := m.Called(ctx, id, userId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Transaction), args.Error(1)
}

// List simulates listing all transactions for a given user.
func (m *MockTransactionRepository) List(ctx context.Context, userId int64, filters repository.ListTransactionFilters) ([]model.Transaction, error) {
	args := m.Called(ctx, userId, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Transaction), args.Error(1)
}
func (m *MockTransactionRepository) ListByAccountAndDateRange(ctx context.Context, userId, accountId int64, startDate, endDate time.Time) ([]model.Transaction, error) {
	args := m.Called(ctx, userId, accountId, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Transaction), args.Error(1)
}

// Update simulates updating a transaction in the database.
func (m *MockTransactionRepository) Update(ctx context.Context, tx model.Transaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

// Delete simulates deleting a transaction by its Id and user Id.
func (m *MockTransactionRepository) Delete(ctx context.Context, id, userId int64) error {
	// Note: Fixed a bug here. Original was m.Called(ctx, userId, userId).
	args := m.Called(ctx, id, userId)
	return args.Error(0)
}

// DeleteByAccountId simulates deleting all transactions associated with a specific account.
func (m *MockTransactionRepository) DeleteByAccountId(ctx context.Context, userId, accountId int64) error {
	args := m.Called(ctx, userId, accountId)
	return args.Error(0)
}

func (m *MockTransactionRepository) SumExpensesByCategoryAndPeriod(ctx context.Context, userId, categoryId int64, startDate, endDate time.Time) (decimal.Decimal, error) {
	args := m.Called(ctx, userId, categoryId, startDate, endDate)
	// Get the first return argument and assert it's a decimal.Decimal
	data := args.Get(0)
	if data == nil {
		return decimal.Zero, args.Error(1)
	}
	return data.(decimal.Decimal), args.Error(1)

}

// TestTransactionService contains all tests for transaction service business logic .
func TestTransactionService(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.Disabled)
	ctx := context.Background()

	// setup is a helper to initialize dependencies for each test case, avoiding code duplication.
	setup := func() (*TransactionService, *MockAccountRepository, *MockTransactionRepository) {
		mockAccountRepo := new(MockAccountRepository)
		mockTxRepo := new(MockTransactionRepository)
		txService := NewTransactionService(mockTxRepo, mockAccountRepo)
		return txService, mockAccountRepo, mockTxRepo
	}

	baseTx := model.Transaction{
		UserId:    1,
		AccountId: 10,
		Amount:    decimal.NewFromInt(100),
		Type:      model.Expense,
		Date:      time.Now(),
	}
	destAccountId := int64(20)

	t.Run("CreateTransaction", func(t *testing.T) {

		t.Run("success: should create a simple income/expense transaction", func(t *testing.T) {
			// Arrange
			txService, mockAccountRepo, mockTxRepo := setup()

			mockAccountRepo.On("GetById", ctx, baseTx.AccountId, baseTx.UserId).Return(&model.Account{}, nil).Once()
			mockTxRepo.On("Create", ctx, mock.AnythingOfType("model.Transaction")).Return(int64(123), nil).Once()

			// Act
			id, err := txService.CreateTransaction(ctx, baseTx)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, int64(123), id)
			mockAccountRepo.AssertExpectations(t)
			mockTxRepo.AssertExpectations(t)
		})

		t.Run("success: should create a transfer transaction", func(t *testing.T) {
			// Arrange
			txService, mockAccountRepo, mockTxRepo := setup()
			transferTx := baseTx
			transferTx.Type = model.Transfer
			transferTx.DestinationAccountId = &destAccountId

			mockAccountRepo.On("GetById", ctx, transferTx.AccountId, transferTx.UserId).Return(&model.Account{}, nil).Once()
			mockAccountRepo.On("GetById", ctx, *transferTx.DestinationAccountId, transferTx.UserId).Return(&model.Account{}, nil).Once()
			mockTxRepo.On("Create", ctx, mock.AnythingOfType("model.Transaction")).Return(int64(124), nil).Once()

			// Act
			id, err := txService.CreateTransaction(ctx, transferTx)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, int64(124), id)
			mockAccountRepo.AssertExpectations(t)
			mockTxRepo.AssertExpectations(t)
		})

		t.Run("failure: should return error for invalid amount", func(t *testing.T) {
			// Arrange
			txService, mockAccountRepo, mockTxRepo := setup()
			txWithInvalidAmount := baseTx
			txWithInvalidAmount.Amount = decimal.NewFromInt(0)

			// Act
			_, err := txService.CreateTransaction(ctx, txWithInvalidAmount)

			// Assert
			assert.Error(t, err)
			assert.Equal(t, "transaction amount must be positive", err.Error())
			mockAccountRepo.AssertNotCalled(t, "GetById")
			mockTxRepo.AssertNotCalled(t, "Create")
		})

		t.Run("failure: should return error if source account does not exist", func(t *testing.T) {
			// Arrange
			txService, mockAccountRepo, mockTxRepo := setup()

			mockAccountRepo.On("GetById", ctx, baseTx.AccountId, baseTx.UserId).Return(nil, sql.ErrNoRows).Once()

			// Act
			_, err := txService.CreateTransaction(ctx, baseTx)

			// Assert
			assert.Error(t, err)
			assert.Equal(t, "source account not found or does not belong to the user", err.Error())
			mockAccountRepo.AssertExpectations(t)
			mockTxRepo.AssertNotCalled(t, "Create")
		})

		t.Run("failure: should return error for transfer if destination account does not exist", func(t *testing.T) {
			// Arrange
			txService, mockAccountRepo, mockTxRepo := setup()
			transferTx := baseTx
			transferTx.Type = model.Transfer
			transferTx.DestinationAccountId = &destAccountId

			mockAccountRepo.On("GetById", ctx, transferTx.AccountId, transferTx.UserId).Return(&model.Account{}, nil).Once()
			mockAccountRepo.On("GetById", ctx, *transferTx.DestinationAccountId, transferTx.UserId).Return(nil, sql.ErrNoRows).Once()

			// Act
			_, err := txService.CreateTransaction(ctx, transferTx)

			// Assert
			assert.Error(t, err)
			assert.Equal(t, "destination account not found or does not belong to the user", err.Error())
			mockAccountRepo.AssertExpectations(t)
			mockTxRepo.AssertNotCalled(t, "Create")
		})

		t.Run("failure: should return error for transfer if source and destination are the same", func(t *testing.T) {
			// Arrange
			txService, mockAccountRepo, _ := setup()
			sameAccountId := int64(10)
			transferTx := baseTx
			transferTx.Type = model.Transfer
			transferTx.AccountId = sameAccountId
			transferTx.DestinationAccountId = &sameAccountId

			mockAccountRepo.On("GetById", ctx, transferTx.AccountId, transferTx.UserId).Return(&model.Account{}, nil).Once()

			// Act
			_, err := txService.CreateTransaction(ctx, transferTx)

			// Assert
			assert.Error(t, err)
			assert.Equal(t, "source and destination accounts cannot be the same", err.Error())
			mockAccountRepo.AssertExpectations(t)
		})
	})
	// TODO: Further tests for Update, Delete, and List methods can be added following the same pattern.
}
