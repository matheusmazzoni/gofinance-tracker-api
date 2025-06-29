package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/testhelper"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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

// ListByUserId simula a listagem de contas de um usuário.
func (m *MockAccountRepository) ListByUserId(ctx context.Context, userId int64) ([]model.Account, error) {
	args := m.Called(ctx, userId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Account), args.Error(1)
}

// GetCurrentBalance simula o cálculo de saldo de uma conta.
func (m *MockAccountRepository) GetCurrentBalance(ctx context.Context, accountId, userId int64) (decimal.Decimal, error) {
	args := m.Called(ctx, accountId, userId)
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

func TestAccountServiceGetStatementDetails(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.Disabled)
	ctx := context.Background()

	const testUserID = int64(1)
	const testAccountID = int64(10)

	testCases := []struct {
		name                   string
		targetYear             int
		targetMonth            time.Month
		mockAccount            *model.Account
		mockTransactions       []model.Transaction
		mockAccountError       error
		mockTxError            error
		expectError            bool
		expectedErrorMsg       string
		expectedStartDate      time.Time
		expectedEndDate        time.Time
		expectedPaymentDueDate time.Time
	}{
		{
			name:        "should calculate period for a standard mid-year month",
			targetYear:  2025,
			targetMonth: time.July,
			mockAccount: &model.Account{
				Id:                  testAccountID,
				UserId:              testUserID,
				Type:                model.CreditCard,
				StatementClosingDay: testhelper.Ptr(20), // Closing day is 20th
				PaymentDueDay:       testhelper.Ptr(10),
			},
			// For a bill closing on July 20th, the period is June 20th to July 20th.
			expectedStartDate:      time.Date(2025, time.June, 20, 0, 0, 0, 0, time.UTC),
			expectedEndDate:        time.Date(2025, time.July, 20, 0, 0, 0, 0, time.UTC),
			expectedPaymentDueDate: time.Date(2025, time.July, 10, 0, 0, 0, 0, time.UTC),
		},
		{
			name:        "should calculate period correctly across year boundary",
			targetYear:  2025,
			targetMonth: time.January,
			mockAccount: &model.Account{
				Id:                  testAccountID,
				UserId:              testUserID,
				Type:                model.CreditCard,
				StatementClosingDay: testhelper.Ptr(20), // Closing day is 20th
				PaymentDueDay:       testhelper.Ptr(21),
			},
			// For a bill closing on Jan 20th, 2025, the period is Dec 20th, 2024 to Jan 20th, 2025.
			expectedStartDate:      time.Date(2024, time.December, 20, 0, 0, 0, 0, time.UTC),
			expectedEndDate:        time.Date(2025, time.January, 20, 0, 0, 0, 0, time.UTC),
			expectedPaymentDueDate: time.Date(2025, time.January, 21, 0, 0, 0, 0, time.UTC),
		},
		{
			name:        "should handle end-of-month logic for a 30-day month",
			targetYear:  2025,
			targetMonth: time.April,
			mockAccount: &model.Account{
				Id:                  testAccountID,
				UserId:              testUserID,
				Type:                model.CreditCard,
				StatementClosingDay: testhelper.Ptr(31), // Closing day set to 31
				PaymentDueDay:       testhelper.Ptr(5),
			},
			// Service should use GetLastDayOfMonth, resulting in April 30th.
			// The start date becomes March 31st.
			expectedStartDate:      time.Date(2025, time.March, 31, 0, 0, 0, 0, time.UTC),
			expectedEndDate:        time.Date(2025, time.April, 30, 0, 0, 0, 0, time.UTC),
			expectedPaymentDueDate: time.Date(2025, time.April, 5, 0, 0, 0, 0, time.UTC),
		},
		{
			name:        "should handle end-of-month logic for a leap year February",
			targetYear:  2024, // 2024 is a leap year
			targetMonth: time.February,
			mockAccount: &model.Account{
				Id:                  testAccountID,
				UserId:              testUserID,
				Type:                model.CreditCard,
				StatementClosingDay: testhelper.Ptr(31), // Closing day set to 31
				PaymentDueDay:       testhelper.Ptr(15),
			},
			expectedStartDate:      time.Date(2024, time.January, 31, 0, 0, 0, 0, time.UTC),
			expectedEndDate:        time.Date(2024, time.February, 29, 0, 0, 0, 0, time.UTC),
			expectedPaymentDueDate: time.Date(2024, time.February, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:             "should return error if account is not found",
			targetYear:       2025,
			targetMonth:      time.July,
			mockAccountError: errors.New("database error: account not found"),
			expectError:      true,
			expectedErrorMsg: "account not found",
		},
		{
			name:        "should return error if account is not a credit card",
			targetYear:  2025,
			targetMonth: time.July,
			mockAccount: &model.Account{
				Id:     testAccountID,
				UserId: testUserID,
				Type:   model.Checking, // Not a CreditCard
			},
			expectError:      true,
			expectedErrorMsg: "operation only valid for credit card accounts",
		},
		{
			name:        "should return error if closing day is not set",
			targetYear:  2025,
			targetMonth: time.July,
			mockAccount: &model.Account{
				Id:                  testAccountID,
				UserId:              testUserID,
				Type:                model.CreditCard,
				StatementClosingDay: nil, // Missing data
				PaymentDueDay:       testhelper.Ptr(10),
			},
			expectError:      true,
			expectedErrorMsg: "credit card account must have billing cycle data",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			mockAccountRepo := new(MockAccountRepository)
			mockTxRepo := new(MockTransactionRepository)
			accountService := NewAccountService(mockAccountRepo, mockTxRepo)

			// Setup mocks based on the test case
			if tc.mockAccountError != nil {
				mockAccountRepo.On("GetById", ctx, testAccountID, testUserID).Return(nil, tc.mockAccountError)
			} else if tc.mockAccount != nil {
				mockAccountRepo.On("GetById", ctx, testAccountID, testUserID).Return(tc.mockAccount, nil)
			}

			// Only set up the transaction mock if no error is expected before that call
			if !tc.expectError {
				mockTxRepo.On("ListByAccountAndDateRange", ctx, testUserID, testAccountID, tc.expectedStartDate, tc.expectedEndDate).Return(tc.mockTransactions, tc.mockTxError)
			}

			// Act
			statement, err := accountService.GetStatementDetails(ctx, testUserID, testAccountID, tc.targetYear, int(tc.targetMonth))

			// Assert
			if tc.expectError {
				assert.Error(t, err)
				if tc.expectedErrorMsg != "" {
					assert.Contains(t, err.Error(), tc.expectedErrorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, statement)
				assert.Equal(t, tc.expectedStartDate, statement.StatementPeriod.Start)
				assert.Equal(t, tc.expectedEndDate, statement.StatementPeriod.End)
				assert.Equal(t, tc.expectedPaymentDueDate, statement.PaymentDueDate)
			}

			// Verify that all expected calls to the mocks were made.
			mockAccountRepo.AssertExpectations(t)
			mockTxRepo.AssertExpectations(t)
		})
	}
}
