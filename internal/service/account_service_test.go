package service

import (
	"context"
	"errors"
	"testing"

	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAccountRepository é uma simulação do nosso repositório de contas.
// Ele implementa a interface repository.AccountRepository para ser usado nos testes de serviço.
type MockAccountRepository struct {
	mock.Mock
}

func (m *MockAccountRepository) GetById(ctx context.Context, id, userId int64) (*model.Account, error) {
	args := m.Called(ctx, id, userId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Account), args.Error(1)
}

// GetAccountByName retrieves an account by its name for a specific user.
func (m *MockAccountRepository) GetByName(ctx context.Context, name string, userId int64) (*model.Account, error) {
	args := m.Called(ctx, name, userId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Account), args.Error(1)
}

// Create simula a criação de uma nova conta.
func (m *MockAccountRepository) Create(ctx context.Context, acc model.Account) (int64, error) {
	args := m.Called(ctx, acc)
	// A conversão para int64 é necessária pois o mock retorna um tipo genérico.
	return args.Get(0).(int64), args.Error(1)
}

// Update simula a atualização de uma conta.
func (m *MockAccountRepository) Update(ctx context.Context, acc model.Account) error {
	args := m.Called(ctx, acc)
	return args.Error(0)
}

// Delete simula a exclusão de uma conta.
func (m *MockAccountRepository) Delete(ctx context.Context, id, userId int64) error {
	args := m.Called(ctx, id, userId)
	return args.Error(0)
}

// ListByUserID simula a listagem de contas de um usuário.
func (m *MockAccountRepository) ListByUserId(ctx context.Context, userId int64) ([]model.Account, error) {
	args := m.Called(ctx, userId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Account), args.Error(1)
}

// GetCurrentBalance simula o cálculo de saldo de uma conta.
func (m *MockAccountRepository) GetCurrentBalance(ctx context.Context, accountID, userId int64) (decimal.Decimal, error) {
	args := m.Called(ctx, accountID, userId)
	// A conversão para decimal.Decimal é necessária.
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

// TestAccountService tests the business logic of the AccountService.
func TestAccountService(t *testing.T) {
	// Disable logging for tests to keep output clean
	zerolog.SetGlobalLevel(zerolog.Disabled)

	// --- Tests for CreateAccount ---
	t.Run("CreateAccount", func(t *testing.T) {
		mockAccountRepo := new(MockAccountRepository)
		accountService := NewAccountService(mockAccountRepo, nil)
		ctx := context.Background()

		accountToCreate := model.Account{UserId: 1, Name: "New Savings"}

		// Arrange: Expect the Create method to be called and return a new Id
		mockAccountRepo.On("Create", ctx, accountToCreate).Return(int64(123), nil).Once()

		// Act
		id, err := accountService.CreateAccount(ctx, accountToCreate)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, int64(123), id)
		mockAccountRepo.AssertExpectations(t)
	})

	// --- Tests for GetAccountById ---
	t.Run("GetAccountById", func(t *testing.T) {
		mockAccountRepo := new(MockAccountRepository)
		accountService := NewAccountService(mockAccountRepo, nil)
		ctx := context.Background()

		// Arrange
		expectedAccount := &model.Account{Id: 10, UserId: 1, Name: "Test Account"}
		expectedBalance := decimal.NewFromInt(100)
		mockAccountRepo.On("GetById", ctx, int64(10), int64(1)).Return(expectedAccount, nil).Once()
		mockAccountRepo.On("GetCurrentBalance", ctx, int64(10), int64(1)).Return(expectedBalance, nil).Once()

		// Act
		resultAccount, err := accountService.GetAccountById(ctx, 10, 1)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "Test Account", resultAccount.Name)
		assert.True(t, expectedBalance.Equal(resultAccount.Balance))
		mockAccountRepo.AssertExpectations(t)
	})

	// --- Tests for ListAccountsByUserId ---
	t.Run("ListAccountsByUserId", func(t *testing.T) {
		t.Run("should list accounts and calculate all balances successfully", func(t *testing.T) {
			mockAccountRepo := new(MockAccountRepository)
			accountService := NewAccountService(mockAccountRepo, nil)
			ctx := context.Background()

			// Arrange
			accountsFromRepo := []model.Account{{Id: 1}, {Id: 2}}
			mockAccountRepo.On("ListByUserId", ctx, int64(1)).Return(accountsFromRepo, nil).Once()
			mockAccountRepo.On("GetCurrentBalance", ctx, int64(1), int64(1)).Return(decimal.NewFromInt(100), nil).Once()
			mockAccountRepo.On("GetCurrentBalance", ctx, int64(2), int64(1)).Return(decimal.NewFromInt(250), nil).Once()

			// Act
			resultAccounts, err := accountService.ListAccountsByUserId(ctx, 1)

			// Assert
			assert.NoError(t, err)
			assert.Len(t, resultAccounts, 2)
			assert.True(t, decimal.NewFromInt(100).Equal(resultAccounts[0].Balance))
			assert.True(t, decimal.NewFromInt(250).Equal(resultAccounts[1].Balance))
			mockAccountRepo.AssertExpectations(t)
		})

		t.Run("should return accounts even if one balance calculation fails", func(t *testing.T) {
			mockAccountRepo := new(MockAccountRepository)
			accountService := NewAccountService(mockAccountRepo, nil)
			ctx := context.Background()

			// Arrange
			accountsFromRepo := []model.Account{{Id: 1}, {Id: 2}}
			mockAccountRepo.On("ListByUserId", ctx, int64(1)).Return(accountsFromRepo, nil).Once()
			// First balance call succeeds
			mockAccountRepo.On("GetCurrentBalance", ctx, int64(1), int64(1)).Return(decimal.NewFromInt(100), nil).Once()
			// Second balance call fails
			mockAccountRepo.On("GetCurrentBalance", ctx, int64(2), int64(1)).Return(decimal.Decimal{}, errors.New("db error")).Once()

			// Act
			resultAccounts, err := accountService.ListAccountsByUserId(ctx, 1)

			// Assert
			assert.NoError(t, err, "service should not return error, only log a warning")
			assert.Len(t, resultAccounts, 2)
			assert.True(t, decimal.NewFromInt(100).Equal(resultAccounts[0].Balance))
			assert.True(t, decimal.Zero.Equal(resultAccounts[1].Balance), "balance should be zero on failure")
			mockAccountRepo.AssertExpectations(t)
		})
	})

	// --- Tests for UpdateAccount ---
	t.Run("UpdateAccount", func(t *testing.T) {
		mockAccountRepo := new(MockAccountRepository)
		accountService := NewAccountService(mockAccountRepo, nil)
		ctx := context.Background()

		// Arrange
		accountToUpdate := model.Account{Name: "Updated Name"}
		accountAfterUpdate := &model.Account{Id: 10, UserId: 1, Name: "Updated Name"}
		mockAccountRepo.On("Update", ctx, mock.Anything).Return(nil).Once()
		mockAccountRepo.On("GetById", ctx, int64(10), int64(1)).Return(accountAfterUpdate, nil).Once()
		mockAccountRepo.On("GetCurrentBalance", ctx, int64(10), int64(1)).Return(decimal.NewFromInt(100), nil).Once()

		// Act
		resultAccount, err := accountService.UpdateAccount(ctx, 10, 1, accountToUpdate)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "Updated Name", resultAccount.Name)
		mockAccountRepo.AssertExpectations(t)
	})

	// --- Tests for DeleteAccount ---
	t.Run("DeleteAccount", func(t *testing.T) {
		mockAccountRepo := new(MockAccountRepository)
		mockTransactionRepo := new(MockTransactionRepository)
		accountService := NewAccountService(mockAccountRepo, mockTransactionRepo)
		ctx := context.Background()

		// Arrange
		mockAccountRepo.On("GetById", ctx, int64(10), int64(1)).Return(&model.Account{Id: 10}, nil).Once()
		mockTransactionRepo.On("DeleteByAccountId", ctx, int64(1), int64(10)).Return(nil).Once()
		mockAccountRepo.On("Delete", ctx, int64(10), int64(1)).Return(nil).Once()

		// Act
		err := accountService.DeleteAccount(ctx, 10, 1)

		// Assert
		assert.NoError(t, err)
		mockAccountRepo.AssertExpectations(t)
		mockTransactionRepo.AssertExpectations(t)
	})
}
