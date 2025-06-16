package service

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

// MockUserRepository is a mock implementation of the UserRepository interface.
// It uses stretchr/testify's mock package to simulate database interactions for testing the UserService.
type MockUserRepository struct {
	mock.Mock
}

// Create simulates creating a user in the database.
func (m *MockUserRepository) Create(ctx context.Context, user model.User) (int64, error) {
	args := m.Called(ctx, user)
	return args.Get(0).(int64), args.Error(1)
}

// GetById simulates retrieving a user by their ID.
func (m *MockUserRepository) GetById(ctx context.Context, id int64) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

// GetByEmail simulates retrieving a user by their email address.
func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

// TestUserService contains all tests for the user service logic.
func TestUserService(t *testing.T) {
	// Disable logging for tests to keep output clean.
	zerolog.SetGlobalLevel(zerolog.Disabled)
	ctx := context.Background()

	// setup is a helper function to initialize dependencies for each test case.
	// This avoids code duplication within each t.Run block.
	setup := func() (*UserService, *MockUserRepository, *MockCategoryRepository) {
		mockUserRepo := new(MockUserRepository)
		mockCategoryRepo := new(MockCategoryRepository)
		userService := NewUserService(mockUserRepo, mockCategoryRepo)
		return userService, mockUserRepo, mockCategoryRepo
	}

	t.Run("CreateUser", func(t *testing.T) {
		t.Run("should hash password and create user successfully", func(t *testing.T) {
			// Arrange
			userService, mockUserRepo, mockCategoryRepo := setup()
			plainTextPassword := "strong-password-123"
			userToCreate := model.User{
				Name:     "John Doe",
				Email:    "john.doe@example.com",
				Password: plainTextPassword,
			}

			// We need a WaitGroup to synchronize with the goroutine that creates default categories.
			var wg sync.WaitGroup
			numDefaultCategories := len(DefaultCategories)
			wg.Add(numDefaultCategories) // Expect N calls to wg.Done().

			// We need to capture the user object that is passed to the repo
			// to verify that the password was hashed correctly.
			var capturedUser model.User
			mockUserRepo.On("Create", ctx, mock.AnythingOfType("model.User")).
				Run(func(args mock.Arguments) {
					capturedUser = args.Get(1).(model.User)
				}).
				Return(int64(1), nil).
				Once()

			// Mock the creation of default categories.
			mockCategoryRepo.On("Create", mock.Anything, mock.AnythingOfType("model.Category")).
				Return(int64(1), nil).
				// For each call, mark one task as done in the WaitGroup.
				Run(func(args mock.Arguments) {
					wg.Done()
				}).
				Times(numDefaultCategories)

			// Act
			id, err := userService.CreateUser(ctx, userToCreate)
			wg.Wait() // Wait for all category creation goroutines to finish.

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, int64(1), id)

			// Verify the core business logic:
			// 1. The plaintext password field should be cleared.
			assert.Empty(t, capturedUser.Password, "Plaintext password should be cleared")
			// 2. The password hash field should NOT be empty.
			assert.NotEmpty(t, capturedUser.PasswordHash, "PasswordHash should be populated")
			// 3. The generated hash should match the original plaintext password.
			err = bcrypt.CompareHashAndPassword([]byte(capturedUser.PasswordHash), []byte(plainTextPassword))
			assert.NoError(t, err, "Hashed password should match original password")

			mockUserRepo.AssertExpectations(t)
			mockCategoryRepo.AssertExpectations(t)
		})
	})

	t.Run("GetUserById", func(t *testing.T) {
		t.Run("should return user when found", func(t *testing.T) {
			// Arrange
			userService, mockUserRepo, _ := setup()
			expectedUser := &model.User{Id: 1, Name: "Jane Doe"}
			mockUserRepo.On("GetById", ctx, int64(1)).Return(expectedUser, nil).Once()

			// Act
			user, err := userService.GetUserById(ctx, 1)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, expectedUser, user)
			mockUserRepo.AssertExpectations(t)
		})

		t.Run("should return error when not found", func(t *testing.T) {
			// Arrange
			userService, mockUserRepo, _ := setup()
			mockUserRepo.On("GetById", ctx, int64(99)).Return(nil, sql.ErrNoRows).Once()

			// Act
			user, err := userService.GetUserById(ctx, 99)

			// Assert
			assert.Error(t, err)
			assert.Nil(t, user)
			assert.ErrorIs(t, err, sql.ErrNoRows, "Should return a standard sql.ErrNoRows")
			mockUserRepo.AssertExpectations(t)
		})
	})
}
