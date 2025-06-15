package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockTransactionRepository struct {
	mock.Mock
}

func (m *MockTransactionRepository) Create(ctx context.Context, tx model.Transaction) (int64, error) {
	args := m.Called(ctx, tx)
	return args.Get(0).(int64), args.Error(1)
}

// Implement other TransactionRepository methods with panic
func (m *MockTransactionRepository) GetById(ctx context.Context, id, userId int64) (*model.Transaction, error) {
	args := m.Called(ctx, id, userId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Transaction), args.Error(1)
}
func (m *MockTransactionRepository) ListByUserId(ctx context.Context, userId int64) ([]model.Transaction, error) {
	panic("not implemented")
}
func (m *MockTransactionRepository) Update(ctx context.Context, tx model.Transaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}
func (m *MockTransactionRepository) Delete(ctx context.Context, id, userId int64) error {
	args := m.Called(ctx, userId, userId)
	return args.Error(0)
}
func (m *MockTransactionRepository) DeleteByAccountId(ctx context.Context, userId, accountId int64) error {
	args := m.Called(ctx, userId, accountId)
	return args.Error(0)
}

func TestTransactionService_CreateTransaction(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.Disabled)
	ctx := context.Background()

	baseTx := model.Transaction{
		UserId:    1,
		AccountId: 10,
		Amount:    decimal.NewFromInt(100),
		Type:      model.Expense,
		Date:      time.Now(),
	}

	destAccountId := int64(20)

	t.Run("success: should create a simple income/expense transaction", func(t *testing.T) {
		// Arrange
		mockAccountRepo := new(MockAccountRepository)
		mockTxRepo := new(MockTransactionRepository)
		txService := NewTransactionService(mockTxRepo, mockAccountRepo)

		// Expect GetById to be called for the source account, and it should succeed.
		mockAccountRepo.On("GetById", ctx, baseTx.AccountId, baseTx.UserId).Return(&model.Account{}, nil).Once()
		// Expect Create to be called on the transaction repo, and it should succeed.
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
		mockAccountRepo := new(MockAccountRepository)
		mockTxRepo := new(MockTransactionRepository)
		txService := NewTransactionService(mockTxRepo, mockAccountRepo)

		transferTx := baseTx
		transferTx.Type = model.Transfer
		transferTx.DestinationAccountId = &destAccountId

		// Expect GetById for source account
		mockAccountRepo.On("GetById", ctx, transferTx.AccountId, transferTx.UserId).Return(&model.Account{}, nil).Once()
		// Expect GetById for destination account
		mockAccountRepo.On("GetById", ctx, *transferTx.DestinationAccountId, transferTx.UserId).Return(&model.Account{}, nil).Once()
		// Expect Create to be called
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
		mockAccountRepo := new(MockAccountRepository)
		mockTxRepo := new(MockTransactionRepository)
		txService := NewTransactionService(mockTxRepo, mockAccountRepo)

		txWithInvalidAmount := baseTx
		txWithInvalidAmount.Amount = decimal.NewFromInt(0)

		// Act
		_, err := txService.CreateTransaction(ctx, txWithInvalidAmount)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, "transaction amount must be positive", err.Error())
		// Ensure no repository methods were ever called
		mockAccountRepo.AssertNotCalled(t, "GetById")
		mockTxRepo.AssertNotCalled(t, "Create")
	})

	t.Run("failure: should return error if source account does not exist", func(t *testing.T) {
		// Arrange
		mockAccountRepo := new(MockAccountRepository)
		mockTxRepo := new(MockTransactionRepository)
		txService := NewTransactionService(mockTxRepo, mockAccountRepo)

		// Expect GetById for the source account to fail
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
		mockAccountRepo := new(MockAccountRepository)
		mockTxRepo := new(MockTransactionRepository)
		txService := NewTransactionService(mockTxRepo, mockAccountRepo)

		transferTx := baseTx
		transferTx.Type = model.Transfer
		transferTx.DestinationAccountId = &destAccountId

		// Expect GetById for source account to succeed
		mockAccountRepo.On("GetById", ctx, transferTx.AccountId, transferTx.UserId).Return(&model.Account{}, nil).Once()
		// Expect GetById for destination account to fail
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
		mockAccountRepo := new(MockAccountRepository)
		mockTxRepo := new(MockTransactionRepository)
		txService := NewTransactionService(mockTxRepo, mockAccountRepo)

		sameAccountId := int64(10)
		transferTx := baseTx
		transferTx.Type = model.Transfer
		transferTx.AccountId = sameAccountId
		transferTx.DestinationAccountId = &sameAccountId

		// Expect GetById for source account to succeed
		mockAccountRepo.On("GetById", ctx, transferTx.AccountId, transferTx.UserId).Return(&model.Account{}, nil).Once()

		// Act
		_, err := txService.CreateTransaction(ctx, transferTx)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, "source and destination accounts cannot be the same", err.Error())
		mockAccountRepo.AssertExpectations(t)
		mockTxRepo.AssertNotCalled(t, "Create")
	})
}

// You can add more tests for Update, Delete, List, etc. following the same pattern.
