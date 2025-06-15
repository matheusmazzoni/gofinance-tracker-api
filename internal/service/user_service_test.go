package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user model.User) (int64, error) {
	args := m.Called(ctx, user)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUserRepository) GetById(ctx context.Context, id int64) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func TestUserService(t *testing.T) {
	// Disable logging for tests to keep output clean
	zerolog.SetGlobalLevel(zerolog.Disabled)
	ctx := context.Background()

	t.Run("CreateUser", func(t *testing.T) {
		t.Run("should hash password and create user successfully", func(t *testing.T) {
			// Arrange
			mockRepo := new(MockUserRepository)
			userService := NewUserService(mockRepo)

			plainTextPassword := "strong-password-123"
			userToCreate := model.User{
				Name:     "John Doe",
				Email:    "john.doe@example.com",
				Password: plainTextPassword,
			}

			// We need to capture the user object that is passed to the repo
			// to verify that the password was hashed.
			var capturedUser model.User
			mockRepo.On("Create", ctx, mock.AnythingOfType("model.User")).
				Run(func(args mock.Arguments) {
					capturedUser = args.Get(1).(model.User)
				}).
				Return(int64(1), nil).
				Once()

			// Act
			id, err := userService.CreateUser(ctx, userToCreate)

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

			mockRepo.AssertExpectations(t)
		})
	})

	t.Run("GetUserById", func(t *testing.T) {
		t.Run("should return user when found", func(t *testing.T) {
			// Arrange
			mockRepo := new(MockUserRepository)
			userService := NewUserService(mockRepo)

			expectedUser := &model.User{Id: 1, Name: "Jane Doe"}
			mockRepo.On("GetById", ctx, int64(1)).Return(expectedUser, nil).Once()

			// Act
			user, err := userService.GetUserById(ctx, 1)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, expectedUser, user)
			mockRepo.AssertExpectations(t)
		})

		t.Run("should return error when not found", func(t *testing.T) {
			// Arrange
			mockRepo := new(MockUserRepository)
			userService := NewUserService(mockRepo)

			mockRepo.On("GetById", ctx, int64(99)).Return(nil, sql.ErrNoRows).Once()

			// Act
			user, err := userService.GetUserById(ctx, 99)

			// Assert
			assert.Error(t, err)
			assert.Nil(t, user)
			assert.ErrorIs(t, err, sql.ErrNoRows)
			mockRepo.AssertExpectations(t)
		})
	})
}
