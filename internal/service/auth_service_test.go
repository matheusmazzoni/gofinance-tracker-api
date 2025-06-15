package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthService_Login(t *testing.T) {
	ctx := context.Background()
	testJwtKey := "test_secret_key"
	validPassword := "password123"

	// Geramos um hash real para usar nos nossos testes.
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(validPassword), bcrypt.DefaultCost)

	mockUser := &model.User{
		Id:           1,
		Email:        "test@example.com",
		PasswordHash: string(hashedPassword),
	}

	// Usamos uma estrutura de "table-driven test" para cobrir todos os cenários.
	testCases := []struct {
		name          string
		email         string
		password      string
		setupMock     func(mockRepo *MockUserRepository)
		expectToken   bool
		expectError   bool
		expectedError string
	}{
		{
			name:     "Success: valid credentials should return a JWT",
			email:    "test@example.com",
			password: validPassword,
			setupMock: func(mockRepo *MockUserRepository) {
				mockRepo.On("GetByEmail", ctx, "test@example.com").Return(mockUser, nil).Once()
			},
			expectToken: true,
			expectError: false,
		},
		{
			name:     "Failure: user email does not exist",
			email:    "notfound@example.com",
			password: validPassword,
			setupMock: func(mockRepo *MockUserRepository) {
				mockRepo.On("GetByEmail", ctx, "notfound@example.com").Return(nil, sql.ErrNoRows).Once()
			},
			expectToken:   false,
			expectError:   true,
			expectedError: "invalid credentials",
		},
		{
			name:     "Failure: incorrect password",
			email:    "test@example.com",
			password: "wrong_password",
			setupMock: func(mockRepo *MockUserRepository) {
				mockRepo.On("GetByEmail", ctx, "test@example.com").Return(mockUser, nil).Once()
			},
			expectToken:   false,
			expectError:   true,
			expectedError: "invalid credentials",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			mockRepo := new(MockUserRepository)
			authService := NewAuthService(mockRepo, testJwtKey)
			tc.setupMock(mockRepo) // Configura o mock para este cenário específico

			// Act
			token, err := authService.Login(ctx, tc.email, tc.password)

			// Assert
			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
			}

			if tc.expectToken {
				assert.NotEmpty(t, token)
				// (Opcional) Podemos validar o conteúdo do token gerado
				parsedToken, _ := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) { return []byte(testJwtKey), nil })
				assert.True(t, parsedToken.Valid)
			} else {
				assert.Empty(t, token)
			}

			// Garante que as expectativas do mock foram cumpridas
			mockRepo.AssertExpectations(t)
		})
	}
}
