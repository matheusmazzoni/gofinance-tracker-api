package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/api/dto"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/api/middleware"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/config"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/repository"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/service"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/testhelper"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

var (
	testServer *Server
)

func TestMain(m *testing.M) {
	var pgContainer testcontainers.Container

	testLogger := zerolog.Nop()
	testCfg := config.Config{JWTSecretKey: "account_handler_test_key"}
	testDB, pgContainer := testhelper.SetupTestDB()
	testServer = NewServer(testCfg, testDB, &testLogger)

	exitCode := m.Run()

	if err := pgContainer.Terminate(context.Background()); err != nil {
		log.Fatalf("failed to terminate container: %v", err)
	}

	os.Exit(exitCode)
}

// Middleware
func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Cria uma instância do middleware com uma chave secreta de teste e um logger "mudo".
	authMiddleware := middleware.AuthMiddleware(testServer.config.JWTSecretKey, zerolog.Nop())

	// Cria uma rota protegida de exemplo.
	router.GET("/protected", authMiddleware, func(c *gin.Context) {
		dto.SendSuccessResponse(c, http.StatusOK, gin.H{"message": "access granted"})
	})

	t.Run("should return 401 Unauthorized when Authorization header is missing", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/protected", nil)

		// Act
		router.ServeHTTP(recorder, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, recorder.Code)

		var errorResponse dto.ErrorResponse
		err := json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
		require.NoError(t, err)
		assert.Equal(t, "authorization header required", errorResponse.Error)
	})

	t.Run("should return 401 Unauthorized when token is malformed", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/protected", nil)
		// Token sem o prefixo "Bearer "
		req.Header.Set("Authorization", "a-very-bad-token")

		// Act
		router.ServeHTTP(recorder, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "bearer token required")
	})

	t.Run("should return 401 Unauthorized when token is invalid or signed with wrong key", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		// Gera um token com uma chave secreta DIFERENTE
		invalidToken := testhelper.GenerateTestToken(t, 123, "this_is_the_wrong_key")

		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+invalidToken)

		// Act
		router.ServeHTTP(recorder, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "invalid token")
	})

	t.Run("should return 401 Unauthorized when token is expired", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		// Gera um token que expirou há uma hora.
		claims := &jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(-1 * time.Hour)),
			Subject:   "123",
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		expiredToken, _ := token.SignedString([]byte(testServer.config.JWTSecretKey))

		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+expiredToken)

		// Act
		router.ServeHTTP(recorder, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "invalid token")
	})

	t.Run("should grant access when token is valid", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		validToken := testhelper.GenerateTestToken(t, 123, testServer.config.JWTSecretKey)

		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)

		// Act
		router.ServeHTTP(recorder, req)

		// Assert
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "access granted")
	})
	t.Run("should return 401 Unauthorized when token is missing", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/protected", nil)
		recorder := httptest.NewRecorder()

		router.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	})

	t.Run("should return 401 Unauthorized when token is invalid", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid-token-string")
		recorder := httptest.NewRecorder()

		router.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	})

	t.Run("should allow access when token is valid", func(t *testing.T) {
		validToken := testhelper.GenerateTestToken(t, 123, testServer.config.JWTSecretKey)

		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)
		recorder := httptest.NewRecorder()

		router.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

// Routes
func TestUserRoutes(t *testing.T) {
	require := require.New(t)
	gin.SetMode(gin.TestMode)

	testhelper.TruncateTables(t, testServer.db)

	t.Run("create user successfuly", func(t *testing.T) {
		CreateUserRequestDTO := dto.CreateUserRequest{
			Name:     "Foo Bar",
			Email:    "foo.bar@example.com",
			Password: "password123",
		}
		data, _ := json.Marshal(CreateUserRequestDTO)

		req, _ := http.NewRequest("POST", "/v1/users", bytes.NewBuffer(data))
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		testServer.router.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusCreated, recorder.Code, "the user should be created successfully")
	})
	t.Run("should return 400 Bad Request when creating a user with a duplicate email", func(t *testing.T) {

		firstUserDTO := dto.CreateUserRequest{
			Name:     "John Doe",
			Email:    "john.doe@example.com",
			Password: "password123",
		}
		firstBody, _ := json.Marshal(firstUserDTO)

		req1, _ := http.NewRequest("POST", "/v1/users", bytes.NewBuffer(firstBody))
		req1.Header.Set("Content-Type", "application/json")

		recorder1 := httptest.NewRecorder()
		testServer.router.ServeHTTP(recorder1, req1)

		// Garante que o primeiro usuário foi criado com sucesso.
		assert.Equal(t, http.StatusCreated, recorder1.Code, "the first user should be created successfully")

		// Arrange (2): Prepara a requisição para o segundo usuário com o mesmo e-mail.
		secondUserDTO := dto.CreateUserRequest{
			Name:     "Jane Smith",
			Email:    "john.doe@example.com", // <-- E-mail duplicado
			Password: "password456",
		}
		secondBody, _ := json.Marshal(secondUserDTO)

		req2, _ := http.NewRequest("POST", "/v1/users", bytes.NewBuffer(secondBody))
		req2.Header.Set("Content-Type", "application/json")

		recorder2 := httptest.NewRecorder()

		testServer.router.ServeHTTP(recorder2, req2)

		assert.Equal(t, http.StatusConflict, recorder2.Code, "expected status 409 for duplicate email")

		var errorResponse dto.ErrorResponse
		err := json.Unmarshal(recorder2.Body.Bytes(), &errorResponse)
		require.NoError(err, "should be able to unmarshal error response")

		assert.Contains(t, errorResponse.Error, "a user with this email already exists")
	})
	t.Run("should automatically create default categories for a new user", func(t *testing.T) {
		// This test validates the entire user onboarding flow:
		// 1. A user is created via the API.
		// 2. The user logs in to get a token.
		// 3. We verify that the default categories have been created for them.

		// Arrange
		testhelper.TruncateTables(t, testServer.db)

		newUserEmail := "new.user@example.com"
		newUserPassword := "password123"

		createDTO := dto.CreateUserRequest{
			Name:     "New User With Categories",
			Email:    newUserEmail,
			Password: newUserPassword,
		}
		createBody, _ := json.Marshal(createDTO)

		// Act 1: Create the new user
		recorderCreate := testhelper.MakeAPIRequest(t, testServer.router, "POST", "/v1/users", "", bytes.NewBuffer(createBody))
		assert.Equal(t, http.StatusCreated, recorderCreate.Code)

		// Act 2: Login as the new user to get their token
		loginDTO := dto.LoginRequest{Email: newUserEmail, Password: newUserPassword}
		loginBody, _ := json.Marshal(loginDTO)
		recorderLogin := testhelper.MakeAPIRequest(t, testServer.router, "POST", "/v1/auth/login", "", bytes.NewBuffer(loginBody))
		assert.Equal(t, http.StatusOK, recorderLogin.Code)

		var loginResp dto.LoginResponse
		err := json.Unmarshal(recorderLogin.Body.Bytes(), &loginResp)
		require.NoError(err)
		userToken := loginResp.Token
		require.NotEmpty(t, userToken)

		// Assert: Use assert.Eventually to poll the categories endpoint
		// It will try for 2 seconds, checking every 100 milliseconds.
		require.Eventually(func() bool {
			// This function is the check that will be repeated.

			// Act 3: Get the categories for the new user
			recorderGetCats := testhelper.MakeAPIRequest(t, testServer.router, "GET", "/v1/categories", userToken, bytes.NewBuffer(nil))

			if recorderGetCats.Code != http.StatusOK {
				return false // If the request fails, try again.
			}

			var categories []dto.CategoryResponse
			if err := json.Unmarshal(recorderGetCats.Body.Bytes(), &categories); err != nil {
				return false // If JSON is invalid, try again.
			}

			// The actual assertion: check if the number of created categories matches our default list.
			// The number 11 comes from the defaultCategories slice we defined in the service.
			return assert.Len(t, categories, len(service.DefaultCategories), "should have the default number of categories")

		}, 2*time.Second, 100*time.Millisecond, "it should create the default categories within the time limit")
	})
}

func TestLoginRoutes(t *testing.T) {
	require := require.New(t)
	gin.SetMode(gin.TestMode)

	testhelper.TruncateTables(t, testServer.db)

	userRepo := repository.NewUserRepository(testServer.db)
	categoryRepo := repository.NewCategoryRepository(testServer.db)
	userService := service.NewUserService(userRepo, categoryRepo)

	email := "login@test.com"
	password := "login123"
	_, err := userService.CreateUser(context.Background(), model.User{Name: "Login Routes Test User", Email: email, Password: password})
	require.NoError(err)

	t.Run("successfuly login", func(t *testing.T) {
		loginRequest := dto.LoginRequest{
			Email:    email,
			Password: password,
		}
		data, _ := json.Marshal(loginRequest)

		req, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(data))
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		testServer.router.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)

		var loginResponse dto.LoginResponse
		_ = json.Unmarshal(recorder.Body.Bytes(), &loginResponse)
		require.NotZero(loginResponse.Token)
	})
	t.Run("wrong password login", func(t *testing.T) {
		loginRequest := dto.LoginRequest{
			Email:    email,
			Password: "wrongpassword",
		}
		data, _ := json.Marshal(loginRequest)

		req, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(data))
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		testServer.router.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusUnauthorized, recorder.Code)

		var errorResponse dto.ErrorResponse
		_ = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
		assert.Equal(t, errorResponse.Error, "invalid credentials")
	})
	t.Run("inexistet email login", func(t *testing.T) {
		loginRequest := dto.LoginRequest{
			Email:    "wrongemail@test.com",
			Password: password,
		}
		data, _ := json.Marshal(loginRequest)

		req, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(data))
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		testServer.router.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusUnauthorized, recorder.Code)

		var errorResponse dto.ErrorResponse
		_ = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
		assert.Equal(t, errorResponse.Error, "invalid credentials")
	})
}

func TestAccountRoutes(t *testing.T) {
	require := require.New(t)
	gin.SetMode(gin.TestMode)

	testhelper.TruncateTables(t, testServer.db)

	ctx := context.Background()
	accountRepo := repository.NewAccountRepository(testServer.db)
	userRepo := repository.NewUserRepository(testServer.db)

	t.Run("validate account requests", func(t *testing.T) {

		userId, _ := userRepo.Create(ctx, model.User{Name: "validation", Email: "validation@test.com", PasswordHash: "validation_test_key"})
		token := testhelper.GenerateTestToken(t, userId, testServer.config.JWTSecretKey)

		tests := []struct {
			name          string
			request       dto.AccountRequest
			wantError     bool
			expectedError dto.ErrorResponse
		}{
			{
				name: "valid checking account",
				request: dto.AccountRequest{
					Name:           "Valid Checking Account",
					Type:           "checking",
					InitialBalance: testhelper.Ptr(decimal.NewFromFloat(1000.0)),
				},
				wantError: false,
			},
			{
				name: "valid credit card account",
				request: dto.AccountRequest{
					Name:                "Valid Credit Card",
					Type:                "credit_card",
					InitialBalance:      testhelper.Ptr(decimal.NewFromFloat(0.0)),
					CreditLimit:         testhelper.Ptr(decimal.NewFromFloat(5000.0)),
					PaymentDueDay:       testhelper.Ptr(15),
					StatementClosingDay: testhelper.Ptr(10),
				},
				wantError: false,
			},
			{
				name: "credit card without credit limit",
				request: dto.AccountRequest{
					Name:                "Credit Card",
					Type:                "credit_card",
					InitialBalance:      testhelper.Ptr(decimal.NewFromInt(0)),
					PaymentDueDay:       testhelper.Ptr(15),
					StatementClosingDay: testhelper.Ptr(10),
				},
				wantError:     true,
				expectedError: dto.ErrorResponse{Error: "Validation failed", Details: map[string]string{"credit_limit": "The 'credit_limit' field is required for credit card accounts."}},
			},
			{
				name: "credit card without due day",
				request: dto.AccountRequest{
					Name:                "Credit Card",
					Type:                "credit_card",
					InitialBalance:      testhelper.Ptr(decimal.NewFromInt(0)),
					CreditLimit:         testhelper.Ptr(decimal.NewFromFloat(5000.0)),
					StatementClosingDay: testhelper.Ptr(10),
				},
				wantError:     true,
				expectedError: dto.ErrorResponse{Error: "Validation failed", Details: map[string]string{"payment_due_day": "The 'payment_due_day' field is required for credit card accounts."}},
			},
			{
				name: "credit card without closing day",
				request: dto.AccountRequest{
					Name:           "Credit Card",
					Type:           "credit_card",
					InitialBalance: testhelper.Ptr(decimal.NewFromInt(0)),
					CreditLimit:    testhelper.Ptr(decimal.NewFromFloat(5000.0)),
					PaymentDueDay:  testhelper.Ptr(15),
				},
				wantError:     true,
				expectedError: dto.ErrorResponse{Error: "Validation failed", Details: map[string]string{"statement_closing_day": "The 'statement_closing_day' field is required for credit card accounts."}},
			},
			{
				name: "credit card with zero credit limit",
				request: dto.AccountRequest{
					Name:                "Credit Card",
					Type:                "credit_card",
					InitialBalance:      testhelper.Ptr(decimal.NewFromInt(0)),
					CreditLimit:         testhelper.Ptr(decimal.NewFromFloat(0.0)),
					PaymentDueDay:       testhelper.Ptr(15),
					StatementClosingDay: testhelper.Ptr(10),
				},
				wantError: true,
				expectedError: dto.ErrorResponse{
					Error: "Validation failed",
					Details: map[string]string{
						"credit_limit": "The 'credit_limit' field must be greater than 0 for credit card accounts.",
					},
				},
			},
			{
				name: "credit card with negative credit limit",
				request: dto.AccountRequest{
					Name:                "Credit Card",
					Type:                "credit_card",
					InitialBalance:      testhelper.Ptr(decimal.NewFromInt(0)),
					CreditLimit:         testhelper.Ptr(decimal.NewFromFloat(-1000.0)),
					PaymentDueDay:       testhelper.Ptr(15),
					StatementClosingDay: testhelper.Ptr(10),
				},
				wantError: true,
				expectedError: dto.ErrorResponse{
					Error: "Validation failed",
					Details: map[string]string{
						"credit_limit": "The 'credit_limit' field must be greater than 0 for credit card accounts.",
					},
				},
			},
			{
				name: "checking account with credit card fields",
				request: dto.AccountRequest{
					Name:                "Checking",
					Type:                "checking",
					InitialBalance:      testhelper.Ptr(decimal.NewFromInt(0)),
					CreditLimit:         testhelper.Ptr(decimal.NewFromFloat(5000.0)),
					PaymentDueDay:       testhelper.Ptr(15),
					StatementClosingDay: testhelper.Ptr(10),
				},
				wantError: true,
				expectedError: dto.ErrorResponse{
					Error: "Validation failed",
					Details: map[string]string{
						"credit_limit":          "The 'credit_limit' field is not allowed for this account type.",
						"payment_due_day":       "The 'payment_due_day' field is not allowed for this account type.",
						"statement_closing_day": "The 'statement_closing_day' field is not allowed for this account type.",
					},
				},
			},
			{
				name: "invalid account type",
				request: dto.AccountRequest{
					Name: "Invalid Account",
					Type: "invalid",
				},
				wantError:     true,
				expectedError: dto.ErrorResponse{Error: "Validation failed", Details: map[string]string{"initial_balance": "The 'initial_balance' field is required.", "type": "The 'type' field must be one of: [checking savings credit_card other]."}},
			},
			{
				name: "empty name",
				request: dto.AccountRequest{
					Name: "",
					Type: "checking",
				},
				wantError:     true,
				expectedError: dto.ErrorResponse{Error: "Validation failed", Details: map[string]string{"initial_balance": "The 'initial_balance' field is required.", "name": "The 'name' field is required."}},
			},
			{
				name: "name too long",
				request: dto.AccountRequest{
					Name:           "This is a very long account name that exceeds the maximum length limit of 100 characters and should fail validation",
					Type:           "checking",
					InitialBalance: testhelper.Ptr(decimal.NewFromInt(0)),
				},
				wantError:     true,
				expectedError: dto.ErrorResponse{Error: "Validation failed", Details: map[string]string{"name": "The 'name' field must not exceed 100 characters."}},
			},
			{
				name: "invalid due day - too low",
				request: dto.AccountRequest{
					Name:                "Credit Card",
					Type:                "credit_card",
					InitialBalance:      testhelper.Ptr(decimal.NewFromInt(0)),
					CreditLimit:         testhelper.Ptr(decimal.NewFromFloat(5000.0)),
					PaymentDueDay:       testhelper.Ptr(0),
					StatementClosingDay: testhelper.Ptr(10),
				},
				wantError:     true,
				expectedError: dto.ErrorResponse{Error: "Validation failed", Details: map[string]string{"payment_due_day": "The 'payment_due_day' field must be between 1 and 31. '0' values is not a valid."}},
			},
			{
				name: "invalid due day - too high",
				request: dto.AccountRequest{
					Name:                "Credit Card",
					Type:                "credit_card",
					InitialBalance:      testhelper.Ptr(decimal.NewFromInt(0)),
					CreditLimit:         testhelper.Ptr(decimal.NewFromFloat(5000.0)),
					PaymentDueDay:       testhelper.Ptr(32),
					StatementClosingDay: testhelper.Ptr(10),
				},
				wantError:     true,
				expectedError: dto.ErrorResponse{Error: "Validation failed", Details: map[string]string{"payment_due_day": "The 'payment_due_day' field must be between 1 and 31. '32' values is not a valid."}},
			},
			{
				name: "invalid closing day - too low",
				request: dto.AccountRequest{
					Name:                "Credit Card",
					Type:                "credit_card",
					InitialBalance:      testhelper.Ptr(decimal.NewFromInt(0)),
					CreditLimit:         testhelper.Ptr(decimal.NewFromFloat(5000.0)),
					PaymentDueDay:       testhelper.Ptr(15),
					StatementClosingDay: testhelper.Ptr(0),
				},
				wantError:     true,
				expectedError: dto.ErrorResponse{Error: "Validation failed", Details: map[string]string{"statement_closing_day": "The 'statement_closing_day' field must be between 1 and 31. '0' values is not a valid."}},
			},
			{
				name: "invalid closing day - too high",
				request: dto.AccountRequest{
					Name:                "Credit Card",
					Type:                "credit_card",
					InitialBalance:      testhelper.Ptr(decimal.NewFromInt(0)),
					CreditLimit:         testhelper.Ptr(decimal.NewFromFloat(5000.0)),
					PaymentDueDay:       testhelper.Ptr(15),
					StatementClosingDay: testhelper.Ptr(32),
				},
				wantError: true,
				expectedError: dto.ErrorResponse{
					Error: "Validation failed",
					Details: map[string]string{
						"statement_closing_day": "The 'statement_closing_day' field must be between 1 and 31. '32' values is not a valid.",
					},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				body, err := json.Marshal(tt.request)
				require.NoError(err)
				recorder := testhelper.MakeAPIRequest(t, testServer.router, "POST", "/v1/accounts", token, bytes.NewBuffer(body))

				if tt.wantError {
					var errorBody dto.ErrorResponse
					err = json.Unmarshal(recorder.Body.Bytes(), &errorBody)
					require.NoError(err)
					require.Equal(tt.expectedError, errorBody)

				} else {
					var responseBody dto.AccountResponse
					err = json.Unmarshal(recorder.Body.Bytes(), &responseBody)
					require.NoError(err)
					require.NotZero(responseBody.Id)
				}

			})
		}

	})
	t.Run("create an account successfully", func(t *testing.T) {
		userId, _ := userRepo.Create(ctx, model.User{Name: "UserSuccessfullyCreated", Email: "success@test.com", PasswordHash: "hash"})
		token := testhelper.GenerateTestToken(t, userId, testServer.config.JWTSecretKey)

		accountDTO := dto.AccountRequest{
			Name:           "My API Test Account",
			Type:           model.Checking,
			InitialBalance: testhelper.Ptr(decimal.NewFromInt(100)),
		}
		body, _ := json.Marshal(accountDTO)

		req, _ := http.NewRequest("POST", "/v1/accounts", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		recorder := httptest.NewRecorder()

		testServer.router.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusCreated, recorder.Code)

		var responseBody dto.AccountResponse
		err := json.Unmarshal(recorder.Body.Bytes(), &responseBody)
		require.NoError(err)
		require.NotZero(responseBody.Id)
	})
	t.Run("create multiple accounts of different types", func(t *testing.T) {
		userId, _ := userRepo.Create(ctx, model.User{Name: "MultipleAccountsDiffTypes", Email: "multipleAccountsDiffTypes@test.com", PasswordHash: "hash"})
		token := testhelper.GenerateTestToken(t, userId, testServer.config.JWTSecretKey)

		checkingAccountDTO := dto.AccountRequest{
			Name:           "Checking Account",
			Type:           model.Checking,
			InitialBalance: testhelper.Ptr(decimal.NewFromInt(1000)),
		}
		checkingBody, _ := json.Marshal(checkingAccountDTO)
		reqChecking, _ := http.NewRequest("POST", "/v1/accounts", bytes.NewBuffer(checkingBody))
		reqChecking.Header.Set("Content-Type", "application/json")
		reqChecking.Header.Set("Authorization", "Bearer "+token)
		recorderChecking := httptest.NewRecorder()

		// --- ACT 1: Create the "Checking Account" ---
		testServer.router.ServeHTTP(recorderChecking, reqChecking)

		// --- ASSERT 1: Verify the first creation was successful ---
		assert.Equal(t, http.StatusCreated, recorderChecking.Code, "should create checking account")

		// --- ARRANGE 2: Prepare the "Credit Card" request ---
		creditCardDTO := dto.AccountRequest{
			Name:                "Primary Credit Card",
			Type:                model.CreditCard,
			InitialBalance:      testhelper.Ptr(decimal.NewFromInt(0)),
			CreditLimit:         testhelper.Ptr(decimal.NewFromFloat(1000.00)),
			StatementClosingDay: testhelper.Ptr(20),
			PaymentDueDay:       testhelper.Ptr(27),
		}
		creditCardBody, _ := json.Marshal(creditCardDTO)
		reqCreditCard, _ := http.NewRequest("POST", "/v1/accounts", bytes.NewBuffer(creditCardBody))
		reqCreditCard.Header.Set("Content-Type", "application/json")
		reqCreditCard.Header.Set("Authorization", "Bearer "+token)
		recorderCreditCard := httptest.NewRecorder()

		// --- ACT 2: Create the "Credit Card" ---
		testServer.router.ServeHTTP(recorderCreditCard, reqCreditCard)

		// --- ASSERT 2: Verify the second creation was successful ---
		assert.Equal(t, http.StatusCreated, recorderCreditCard.Code, "should create credit card")

		// Arrange
		reqList, _ := http.NewRequest("GET", "/v1/accounts", nil)
		reqList.Header.Set("Authorization", "Bearer "+token)
		recorderList := httptest.NewRecorder()

		// Act
		testServer.router.ServeHTTP(recorderList, reqList)

		// Assert
		assert.Equal(t, http.StatusOK, recorderList.Code)

		var responseBody []dto.AccountResponse
		err := json.Unmarshal(recorderList.Body.Bytes(), &responseBody)
		require.NoError(err)

		// We expect to find exactly the two accounts we created.
		require.Len(responseBody, 2, "should find two accounts for the user")

		// Check the details of the accounts to be sure.
		// Note: The order is not guaranteed, so we check for presence.
		var foundChecking, foundCreditCard bool
		for _, AccountResponse := range responseBody {
			if AccountResponse.Name == "Checking Account" && AccountResponse.Type == model.Checking {
				foundChecking = true
				assert.True(t, decimal.NewFromInt(1000).Equal(*AccountResponse.Balance))
			}
			if AccountResponse.Name == "Primary Credit Card" && AccountResponse.Type == model.CreditCard {
				foundCreditCard = true
				assert.True(t, decimal.NewFromInt(0).Equal(*AccountResponse.Balance))
			}
		}

		assert.True(t, foundChecking, "the checking account should have been found in the list")
		assert.True(t, foundCreditCard, "the credit card account should have been found in the list")
	})
	t.Run("create account with bad request body", func(t *testing.T) {
		userId, _ := userRepo.Create(ctx, model.User{Name: "AccountBadRequest", Email: "accountBadRequest@test.com", PasswordHash: "hash"})
		token := testhelper.GenerateTestToken(t, userId, testServer.config.JWTSecretKey)

		badRequestBody := `{"type": "checking", "initial_balance": 100}`

		req, _ := http.NewRequest("POST", "/v1/accounts", bytes.NewBuffer([]byte(badRequestBody)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		recorder := httptest.NewRecorder()

		// Act
		testServer.router.ServeHTTP(recorder, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})
	t.Run("delete account without transactions", func(t *testing.T) {
		userId, _ := userRepo.Create(ctx, model.User{Name: "DeleteAccountWitoutTransactions", Email: "deleteAccountWitoutTransactions@test.com", PasswordHash: "hash"})
		token := testhelper.GenerateTestToken(t, userId, testServer.config.JWTSecretKey)

		createAccountBody := `{"name": "DeleteAccountWithoutTransactions", "type": "checking", "initial_balance": 1000}`
		req, _ := http.NewRequest("POST", "/v1/accounts", bytes.NewBuffer([]byte(createAccountBody)))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		testServer.router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusCreated, recorder.Code)

		var createAccountResp dto.AccountResponse
		_ = json.Unmarshal(recorder.Body.Bytes(), &createAccountResp)
		accountId := createAccountResp.Id
		require.NotZero(accountId)

		// Delete Account
		req, _ = http.NewRequest("DELETE", fmt.Sprintf("/v1/accounts/%d", accountId), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder = httptest.NewRecorder()
		testServer.router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusNoContent, recorder.Code)

	})
	t.Run("delete account with transactions", func(t *testing.T) {
		userId, _ := userRepo.Create(ctx, model.User{Name: "DeleteAccountWithTransactions", Email: "deleteAccountWithTransactions@test.com", PasswordHash: "hash"})
		token := testhelper.GenerateTestToken(t, userId, testServer.config.JWTSecretKey)

		createAccountBody := `{"name": "AccountDeleteScenario", "type": "checking", "initial_balance": 1000}`
		req, _ := http.NewRequest("POST", "/v1/accounts", bytes.NewBuffer([]byte(createAccountBody)))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		testServer.router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusCreated, recorder.Code)

		var createAccountResp dto.AccountResponse
		_ = json.Unmarshal(recorder.Body.Bytes(), &createAccountResp)
		accountId := createAccountResp.Id
		require.NotZero(accountId)

		// Create 3 transactions in the account
		var transactionIds []int64
		transactionsToCreate := []string{
			`{"description": "Salário", "amount": "2000.00", "type": "income", "account_id": %d, "date": "2025-06-15T10:00:00Z"}`,
			`{"description": "Aluguel", "amount": "1200.00", "type": "expense", "account_id": %d, "date": "2025-06-15T11:00:00Z"}`,
			`{"description": "Cinema", "amount": "50.00", "type": "expense", "account_id": %d, "date": "2025-06-15T12:00:00Z"}`,
		}
		for _, txBody := range transactionsToCreate {
			req, _ = http.NewRequest("POST", "/v1/transactions", bytes.NewBuffer([]byte(fmt.Sprintf(txBody, accountId))))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			recorder = httptest.NewRecorder()
			testServer.router.ServeHTTP(recorder, req)
			assert.Equal(t, http.StatusCreated, recorder.Code)

			var txResp map[string]int64
			_ = json.Unmarshal(recorder.Body.Bytes(), &txResp)
			transactionIds = append(transactionIds, txResp["id"])
		}
		require.Len(transactionIds, 3)

		// Delete Account
		req, _ = http.NewRequest("DELETE", fmt.Sprintf("/v1/accounts/%d", accountId), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder = httptest.NewRecorder()
		testServer.router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusNoContent, recorder.Code)

	})
	t.Run("update an account name successfully", func(t *testing.T) {
		userId, _ := userRepo.Create(ctx, model.User{Name: "MultipleAccounts", Email: "multiAccounts@test.com", PasswordHash: "hash"})
		token := testhelper.GenerateTestToken(t, userId, testServer.config.JWTSecretKey)

		accountDTO := dto.AccountRequest{
			Name:           "My API Test Account",
			Type:           model.Checking,
			InitialBalance: testhelper.Ptr(decimal.NewFromInt(100)),
		}
		body, _ := json.Marshal(accountDTO)

		req, _ := http.NewRequest("POST", "/v1/accounts", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		recorder := httptest.NewRecorder()

		testServer.router.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusCreated, recorder.Code)

		var responseBody dto.AccountResponse
		err := json.Unmarshal(recorder.Body.Bytes(), &responseBody)
		require.NoError(err)

		accountId := responseBody.Id
		require.NotZero(accountId)

		updateDTO := dto.AccountRequest{
			Name:           "User A Main Account",
			Type:           model.Checking,
			InitialBalance: testhelper.Ptr(decimal.NewFromInt(1000)),
		}
		body, _ = json.Marshal(updateDTO)
		req, _ = http.NewRequest("PUT", fmt.Sprintf("/v1/accounts/%d", accountId), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		recorder = httptest.NewRecorder()

		testServer.router.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		var resp dto.AccountResponse
		_ = json.Unmarshal(recorder.Body.Bytes(), &resp)
		assert.Equal(t, "User A Main Account", resp.Name)
	})
	t.Run("accounts belonging to the authenticated user", func(t *testing.T) {
		userId, _ := userRepo.Create(ctx, model.User{Name: "Authenticated", Email: "autheticated@test.com", PasswordHash: "hash"})
		token := testhelper.GenerateTestToken(t, userId, testServer.config.JWTSecretKey)

		_, _ = accountRepo.Create(ctx, model.Account{UserId: userId, Name: "My API Test Account"})

		req, _ := http.NewRequest("GET", "/v1/accounts", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()

		testServer.router.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		var resp []dto.AccountResponse
		_ = json.Unmarshal(recorder.Body.Bytes(), &resp)

		// User A has only one account at this point: "User A Updated Main Account"
		require.Len(resp, 1)
		assert.Equal(t, "My API Test Account", resp[0].Name)
	})
	t.Run("duplicate account name for the same user", func(t *testing.T) {
		name := "Same Account"
		userId, _ := userRepo.Create(ctx, model.User{Name: "DuplicateAccountSameUser", Email: "duplicateAccountSameUser@test.com", PasswordHash: "hash"})
		token := testhelper.GenerateTestToken(t, userId, testServer.config.JWTSecretKey)

		firstAccount := dto.AccountRequest{
			Name:           name,
			Type:           model.Checking,
			InitialBalance: testhelper.Ptr(decimal.NewFromInt(1000)),
		}
		firstAccountBody, _ := json.Marshal(firstAccount)
		requeFirstAccount, _ := http.NewRequest("POST", "/v1/accounts", bytes.NewBuffer(firstAccountBody))
		requeFirstAccount.Header.Set("Content-Type", "application/json")
		requeFirstAccount.Header.Set("Authorization", "Bearer "+token)
		recorderFisrtAccount := httptest.NewRecorder()

		testServer.router.ServeHTTP(recorderFisrtAccount, requeFirstAccount)

		assert.Equal(t, http.StatusCreated, recorderFisrtAccount.Code)

		secondAccount := dto.AccountRequest{
			Name:                name,
			Type:                model.CreditCard,
			InitialBalance:      testhelper.Ptr(decimal.NewFromInt(0)),
			CreditLimit:         testhelper.Ptr(decimal.NewFromFloat(1000.00)),
			StatementClosingDay: testhelper.Ptr(20),
			PaymentDueDay:       testhelper.Ptr(27),
		}
		secondAccountBody, _ := json.Marshal(secondAccount)
		reqSecondAccount, _ := http.NewRequest("POST", "/v1/accounts", bytes.NewBuffer(secondAccountBody))
		reqSecondAccount.Header.Set("Content-Type", "application/json")
		reqSecondAccount.Header.Set("Authorization", "Bearer "+token)
		recorderSecondAccount := httptest.NewRecorder()

		testServer.router.ServeHTTP(recorderSecondAccount, reqSecondAccount)

		assert.Equal(t, http.StatusConflict, recorderSecondAccount.Code)

		var errorResponse dto.ErrorResponse
		err := json.Unmarshal(recorderSecondAccount.Body.Bytes(), &errorResponse)
		require.NoError(err)

		require.Contains(errorResponse.Error, "account with this name already exists")
	})
	t.Run("get credit card statement", func(t *testing.T) {
		userId, _ := userRepo.Create(ctx, model.User{Name: "CC Statement API User", Email: "cc-statement-api@test.com", PasswordHash: "hash"})
		token := testhelper.GenerateTestToken(t, userId, testServer.config.JWTSecretKey)

		// Create a credit card account with a closing day of the 20th
		accountRepo := repository.NewAccountRepository(testServer.db)
		closingDay, dueDate := 20, 10
		creditCardId, _ := accountRepo.Create(ctx, model.Account{
			UserId:              userId,
			Name:                "My Test Credit Card",
			Type:                model.CreditCard,
			StatementClosingDay: &closingDay,
			PaymentDueDay:       &dueDate,
		})

		// Create transactions in different billing cycles
		txRepo := repository.NewTransactionRepository(testServer.db)
		// This transaction belongs to the PREVIOUS statement (due in May)
		_, _ = txRepo.Create(ctx, model.Transaction{UserId: userId, AccountId: creditCardId, Description: "Old Purchase", Amount: decimal.NewFromInt(1000), Type: "expense", Date: time.Date(2025, 5, 15, 12, 0, 0, 0, time.UTC)})
		// These two transactions belong to the CURRENT statement (due in June)
		_, _ = txRepo.Create(ctx, model.Transaction{UserId: userId, AccountId: creditCardId, Description: "Grocery Shopping", Amount: decimal.NewFromFloat(250.50), Type: "expense", Date: time.Date(2025, 5, 25, 12, 0, 0, 0, time.UTC)})
		_, _ = txRepo.Create(ctx, model.Transaction{UserId: userId, AccountId: creditCardId, Description: "Dinner Out", Amount: decimal.NewFromFloat(150.00), Type: "expense", Date: time.Date(2025, 6, 10, 12, 0, 0, 0, time.UTC)})
		// This transaction belongs to the NEXT statement (due in July)
		_, _ = txRepo.Create(ctx, model.Transaction{UserId: userId, AccountId: creditCardId, Description: "Future Purchase", Amount: decimal.NewFromInt(50), Type: "expense", Date: time.Date(2025, 6, 25, 12, 0, 0, 0, time.UTC)})

		// --- ACT ---
		// We ask for the statement for June (month 6) 2025.
		// The backend should calculate the period from May 20st to June 20th.
		url := fmt.Sprintf("/v1/accounts/%d/statement?month=6&year=2025", creditCardId)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()

		testServer.router.ServeHTTP(recorder, req)

		// --- ASSERT ---
		assert.Equal(t, http.StatusOK, recorder.Code)

		var resp dto.StatementResponse
		err := json.Unmarshal(recorder.Body.Bytes(), &resp)
		require.NoError(err)

		// Check the calculated total
		expectedTotal := decimal.NewFromFloat(400.50) // 250.50 + 150.00
		assert.True(t, expectedTotal.Equal(resp.StatementTotal), "statement total should be calculated correctly")

		// Check that only the correct transactions were included
		assert.Len(t, resp.Transactions, 2, "should only be two transactions in this statement")

		// Check the details of the included transactions
		var descriptions []string
		for _, tx := range resp.Transactions {
			descriptions = append(descriptions, tx.Description)
		}
		assert.Contains(t, descriptions, "Grocery Shopping")
		assert.Contains(t, descriptions, "Dinner Out")
		assert.NotContains(t, descriptions, "Old Purchase")
		assert.NotContains(t, descriptions, "Future Purchase")

	})
}

func TestTransactionRoutes(t *testing.T) {
	require := require.New(t)
	gin.SetMode(gin.TestMode)

	testhelper.TruncateTables(t, testServer.db)

	ctx := context.Background()
	accountRepo := repository.NewAccountRepository(testServer.db)
	userRepo := repository.NewUserRepository(testServer.db)
	categoryRepo := repository.NewCategoryRepository(testServer.db)

	// Create a test user and generate a token
	userId, _ := userRepo.Create(ctx, model.User{Name: "Transaction User", Email: "tx.user@example.com", PasswordHash: "hash"})
	token := testhelper.GenerateTestToken(t, userId, testServer.config.JWTSecretKey)

	// Create prerequisite accounts and a category for the user
	checkingAccountId, _ := accountRepo.Create(ctx, model.Account{UserId: userId, Name: "Checking Account", Type: model.Checking})
	savingsAccountId, _ := accountRepo.Create(ctx, model.Account{UserId: userId, Name: "Savings Account", Type: model.Savings})
	foodCategoryId, _ := categoryRepo.Create(ctx, model.Category{UserId: userId, Name: "Food"})

	var createdExpense model.Transaction

	t.Run("CreateTransactions", func(t *testing.T) {
		t.Run("should create an expense transaction successfully", func(t *testing.T) {
			// Arrange
			expenseDTO := dto.CreateTransactionRequest{
				Description: "Groceries",
				Amount:      decimal.NewFromFloat(75.50),
				Date:        time.Now().UTC(),
				Type:        model.Expense,
				AccountId:   checkingAccountId,
				CategoryId:  &foodCategoryId,
			}
			body, _ := json.Marshal(expenseDTO)
			req, _ := http.NewRequest("POST", "/v1/transactions", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			recorder := httptest.NewRecorder()

			// Act
			testServer.router.ServeHTTP(recorder, req)

			// Assert
			assert.Equal(t, http.StatusCreated, recorder.Code)
			var resp dto.TransactionResponse
			err := json.Unmarshal(recorder.Body.Bytes(), &resp)
			require.NoError(err)
			require.NotZero(resp.Id)

			// Save for later tests
			createdExpense.Id = resp.Id
			createdExpense.Description = expenseDTO.Description
		})

		t.Run("should create a transfer transaction successfully", func(t *testing.T) {
			// Arrange
			transferDTO := dto.CreateTransactionRequest{
				Description:          "Move to savings",
				Amount:               decimal.NewFromInt(500),
				Date:                 time.Now().UTC(),
				Type:                 model.Transfer,
				AccountId:            checkingAccountId,
				DestinationAccountId: &savingsAccountId,
			}
			body, _ := json.Marshal(transferDTO)
			req, _ := http.NewRequest("POST", "/v1/transactions", bytes.NewBuffer(body))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()

			// Act
			testServer.router.ServeHTTP(recorder, req)

			// Assert
			assert.Equal(t, http.StatusCreated, recorder.Code)
		})

		t.Run("should fail with 400 Bad Request if destination account is missing for a transfer", func(t *testing.T) {
			// Arrange
			invalidTransferDTO := dto.CreateTransactionRequest{
				Description: "Invalid Transfer",
				Amount:      decimal.NewFromInt(100),
				Date:        time.Now().UTC(),
				Type:        model.Transfer,
				AccountId:   checkingAccountId,
				// DestinationAccountId is missing
			}
			body, _ := json.Marshal(invalidTransferDTO)
			req, _ := http.NewRequest("POST", "/v1/transactions", bytes.NewBuffer(body))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()

			// Act
			testServer.router.ServeHTTP(recorder, req)

			// Assert
			assert.Equal(t, http.StatusBadRequest, recorder.Code)
			assert.Contains(t, recorder.Body.String(), "destination account is required for a transfer")
		})
	})
	t.Run("ListTransactions", func(t *testing.T) {
		require.NotZero(t, createdExpense.Id, "Create test must run first")
		t.Run("should get a transaction by filter successfully", func(t *testing.T) {
			url := "/v1/transactions?description=" + createdExpense.Description
			respRecorder := testhelper.MakeAPIRequest(t, testServer.router, "GET", url, token, nil)
			var transactions []dto.TransactionResponse
			err := json.Unmarshal(respRecorder.Body.Bytes(), &transactions)
			require.NoError(err)

			// Assert
			assert.Equal(t, http.StatusOK, respRecorder.Code)
			assert.Len(t, transactions, 1, "should have the correct number of transactions")
		})
		t.Run("should get no transaction by wrong filter value", func(t *testing.T) {
			url := "/v1/transactions?description=test"
			respRecorder := testhelper.MakeAPIRequest(t, testServer.router, "GET", url, token, nil)
			var transactions []dto.TransactionResponse
			err := json.Unmarshal(respRecorder.Body.Bytes(), &transactions)
			require.NoError(err)

			// Assert
			assert.Equal(t, http.StatusOK, respRecorder.Code)
			assert.Len(t, transactions, 0, "should have the correct number of transactions")
		})
		t.Run("should get transactions ignoring wrong filter key", func(t *testing.T) {
			url := "/v1/transactions?unknowfilter=" + createdExpense.Description
			respRecorder := testhelper.MakeAPIRequest(t, testServer.router, "GET", url, token, nil)
			var transactions []dto.TransactionResponse
			err := json.Unmarshal(respRecorder.Body.Bytes(), &transactions)
			require.NoError(err)

			// Assert
			assert.Equal(t, http.StatusOK, respRecorder.Code)
			assert.Len(t, transactions, 2, "should have the correct number of transactions")
		})

	})
	t.Run("GetTransaction", func(t *testing.T) {
		require.NotZero(t, createdExpense.Id, "Create test must run first")

		t.Run("should get a transaction by Id successfully", func(t *testing.T) {
			// Arrange
			url := fmt.Sprintf("/v1/transactions/%d", createdExpense.Id)
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			recorder := httptest.NewRecorder()

			// Act
			testServer.router.ServeHTTP(recorder, req)

			// Assert
			assert.Equal(t, http.StatusOK, recorder.Code)
			var resp dto.TransactionResponse
			err := json.Unmarshal(recorder.Body.Bytes(), &resp)
			require.NoError(err)
			assert.Equal(t, "Groceries", resp.Description)
			assert.Equal(t, "Checking Account", resp.AccountName)
			assert.Equal(t, "Food", *resp.CategoryName)
		})
	})
	t.Run("PatchTransaction", func(t *testing.T) {
		require.NotZero(t, createdExpense.Id, "Create test must run first")

		t.Run("should partially update a transaction", func(t *testing.T) {
			// Arrange
			description := "Expensive Groceries"

			patchDTO := dto.PatchTransactionRequest{
				Description: &description,
			}
			body, _ := json.Marshal(patchDTO)
			url := fmt.Sprintf("/v1/transactions/%d", createdExpense.Id)
			req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			recorder := httptest.NewRecorder()

			// Act
			testServer.router.ServeHTTP(recorder, req)

			// Assert
			assert.Equal(t, http.StatusOK, recorder.Code)
			var resp dto.TransactionResponse
			err := json.Unmarshal(recorder.Body.Bytes(), &resp)
			require.NoError(err)
			// Check that the description changed
			assert.Equal(t, "Expensive Groceries", resp.Description)
			// Check that the amount remained the same
			assert.True(t, decimal.NewFromFloat(75.50).Equal(resp.Amount))
		})
	})
	t.Run("Update Transaction", func(t *testing.T) {

		// Create a new user and account for this test
		userId, _ := userRepo.Create(ctx, model.User{Name: "Update User", Email: "update@test.com", PasswordHash: "hash"})
		token := testhelper.GenerateTestToken(t, userId, testServer.config.JWTSecretKey)
		accountId := testhelper.CreateAccount(t, testServer.router, token, dto.AccountRequest{
			Name:           "Account for Update Test",
			Type:           model.Checking,
			InitialBalance: testhelper.Ptr(decimal.NewFromInt(1000)),
		})

		// Create the initial transaction that we are going to update
		initialDescription := "Initial Dinner"
		initialAmount := "50.00"
		txId := testhelper.CreateTransaction(t, testServer.router, token, accountId, initialDescription, "expense", initialAmount)

		// Prepare the update request with new data
		updateDTO := dto.UpdateTransactionRequest{
			Description: "Updated Dinner with Friends",
			Amount:      decimal.NewFromFloat(65.50),
			Date:        time.Now().UTC(),
			Type:        model.Expense,
			AccountId:   accountId,
			CategoryId:  nil, // Not changing the category
		}
		body, _ := json.Marshal(updateDTO)

		// Create the HTTP request for the PUT endpoint
		url := fmt.Sprintf("/v1/transactions/%d", txId)
		req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		recorder := httptest.NewRecorder()

		// ACT: Execute the update request
		testServer.router.ServeHTTP(recorder, req)

		// ASSERT 1: Check the immediate response from the PUT request
		assert.Equal(t, http.StatusOK, recorder.Code, "the update request should succeed")

		var updateResponse dto.TransactionResponse
		err := json.Unmarshal(recorder.Body.Bytes(), &updateResponse)
		require.NoError(err)

		// Verify the response body contains the updated data
		assert.Equal(t, "Updated Dinner with Friends", updateResponse.Description)
		require.True(decimal.NewFromFloat(65.50).Equal(updateResponse.Amount), "the amount in the response should be updated")

		// FINAL VERIFICATION: Fetch the resource again to ensure the change was persisted
		// Arrange for GET
		getReq, _ := http.NewRequest("GET", url, nil)
		getReq.Header.Set("Authorization", "Bearer "+token)
		getRecorder := httptest.NewRecorder()

		// Act for GET
		testServer.router.ServeHTTP(getRecorder, getReq)

		// Assert for GET
		assert.Equal(t, http.StatusOK, getRecorder.Code)
		var finalResponse dto.TransactionResponse
		err = json.Unmarshal(getRecorder.Body.Bytes(), &finalResponse)
		require.NoError(err)

		assert.Equal(t, "Updated Dinner with Friends", finalResponse.Description, "the persisted description should be updated")
		require.True(decimal.NewFromFloat(65.50).Equal(finalResponse.Amount), "the persisted amount should be updated")
	})
	t.Run("Creation Transaction", func(t *testing.T) {
		t.Run("should correctly update account balance after income and expense", func(t *testing.T) {
			userId, err := userRepo.Create(ctx, model.User{Name: "Test User", Email: "transaction@test.com", PasswordHash: "hash"})
			require.NoError(err)
			token := testhelper.GenerateTestToken(t, userId, testServer.config.JWTSecretKey)

			// Arrange: Create an account with an initial balance of $1000
			accountId := testhelper.CreateAccount(t, testServer.router, token, dto.AccountRequest{
				Name:           "Account for Balance Test",
				Type:           model.Checking,
				InitialBalance: testhelper.Ptr(decimal.NewFromInt(1000)),
			})

			// Act 1: Add an income of $5000
			testhelper.CreateTransaction(t, testServer.router, token, accountId, "Monthly Salary", "income", "5000.00")
			// Assert 1: Balance should be 1000 + 5000 = 6000
			balance := testhelper.GetAccountBalance(t, testServer.router, token, accountId)
			require.True(decimal.NewFromInt(6000).Equal(balance), "Balance should increase after income")

			// Act 2: Add an expense of $150
			testhelper.CreateTransaction(t, testServer.router, token, accountId, "Dinner", "expense", "150.00")
			// Assert 2: Balance should be 6000 - 150 = 5850
			balance = testhelper.GetAccountBalance(t, testServer.router, token, accountId)
			require.True(decimal.NewFromInt(5850).Equal(balance), "Balance should decrease after expense")
		})
		t.Run("invalid amount", func(t *testing.T) {
			userId, err := userRepo.Create(ctx, model.User{Name: "Test User", Email: "invalidamount@test.com", PasswordHash: "hash"})
			require.NoError(err)
			token := testhelper.GenerateTestToken(t, userId, testServer.config.JWTSecretKey)
			accountId := testhelper.CreateAccount(t, testServer.router, token, dto.AccountRequest{
				Name:           "Invalid Amount Account",
				Type:           model.Checking,
				InitialBalance: testhelper.Ptr(decimal.NewFromInt(100)),
			})

			createTransactionReq := dto.CreateTransactionRequest{
				Description: "Zero Amount",
				Amount:      decimal.NewFromInt(0),
				Type:        model.Expense,
				AccountId:   accountId,
				Date:        time.Now().UTC(),
			}
			body, _ := json.Marshal(createTransactionReq)
			respRecorder := testhelper.MakeAPIRequest(t, testServer.router, "POST", "/v1/transactions", token, bytes.NewBuffer(body))

			assert.Equal(t, http.StatusBadRequest, respRecorder.Code)
			assert.Contains(t, respRecorder.Body.String(), "transaction amount must be positive")
		})
		t.Run("non-existent source account", func(t *testing.T) {
			userId, err := userRepo.Create(ctx, model.User{Name: "Test User", Email: "noaccount@test.com", PasswordHash: "hash"})
			require.NoError(err)
			token := testhelper.GenerateTestToken(t, userId, testServer.config.JWTSecretKey)

			createTransactionReq := dto.CreateTransactionRequest{
				Description: "Ghost Transaction",
				Amount:      decimal.NewFromInt(100),
				Type:        model.Expense,
				AccountId:   9999999,
				Date:        time.Now().UTC(),
			}
			body, _ := json.Marshal(createTransactionReq)
			respRecorder := testhelper.MakeAPIRequest(t, testServer.router, "POST", "/v1/transactions", token, bytes.NewBuffer(body))

			assert.Equal(t, http.StatusBadRequest, respRecorder.Code)
			assert.Contains(t, respRecorder.Body.String(), "source account not found")
		})
	})
	t.Run("Transfer Scenarios", func(t *testing.T) {
		t.Run("update balances of both accounts after a valid transfer", func(t *testing.T) {
			userId, err := userRepo.Create(ctx, model.User{Name: "Test User", Email: "transfer@test.com", PasswordHash: "hash"})
			require.NoError(err)
			token := testhelper.GenerateTestToken(t, userId, testServer.config.JWTSecretKey)

			// Arrange
			sourceAccountId := testhelper.CreateAccount(t, testServer.router, token, dto.AccountRequest{
				Name:           "Source Account",
				Type:           model.Checking,
				InitialBalance: testhelper.Ptr(decimal.NewFromInt(1000.00)),
			})
			destAccountId := testhelper.CreateAccount(t, testServer.router, token, dto.AccountRequest{
				Name:           "Destination Account",
				Type:           model.Savings,
				InitialBalance: testhelper.Ptr(decimal.NewFromInt(500.00)),
			})

			// Act: Transfer $200 from source to destination
			testhelper.CreateTransfer(t, testServer.router, token, "Saving money", "200.00", sourceAccountId, destAccountId)

			// Assert
			sourceBalance := testhelper.GetAccountBalance(t, testServer.router, token, sourceAccountId)
			destBalance := testhelper.GetAccountBalance(t, testServer.router, token, destAccountId)

			require.True(decimal.NewFromInt(800).Equal(sourceBalance), "Source account balance should decrease by 200")
			require.True(decimal.NewFromInt(700).Equal(destBalance), "Destination account balance should increase by 200")
		})
		t.Run("transfer to the same account", func(t *testing.T) {
			userId, err := userRepo.Create(ctx, model.User{Name: "Test User", Email: "sametransfer@test.com", PasswordHash: "hash"})
			require.NoError(err)
			token := testhelper.GenerateTestToken(t, userId, testServer.config.JWTSecretKey)

			accountId := testhelper.CreateAccount(t, testServer.router, token, dto.AccountRequest{
				Name:           "Same-Transfer Account",
				Type:           model.Checking,
				InitialBalance: testhelper.Ptr(decimal.NewFromInt(1000)),
			})

			createTransactionReq := dto.CreateTransactionRequest{
				Description:          "Invalid",
				Amount:               decimal.NewFromInt(100),
				Type:                 model.Transfer,
				AccountId:            accountId,
				DestinationAccountId: &accountId,
				Date:                 time.Now().UTC(),
			}
			body, _ := json.Marshal(createTransactionReq)
			respRecorder := testhelper.MakeAPIRequest(t, testServer.router, "POST", "/v1/transactions", token, bytes.NewBuffer(body))

			assert.Equal(t, http.StatusBadRequest, respRecorder.Code)
			assert.Contains(t, respRecorder.Body.String(), "source and destination accounts cannot be the same")
		})
		t.Run("non-existent destination account", func(t *testing.T) {
			userId, err := userRepo.Create(ctx, model.User{Name: "Test User", Email: "nodest@test.com", PasswordHash: "hash"})
			require.NoError(err)
			token := testhelper.GenerateTestToken(t, userId, testServer.config.JWTSecretKey)

			sourceAccountId := testhelper.CreateAccount(t, testServer.router, token, dto.AccountRequest{
				Name:           "No-Dest Source",
				Type:           model.Checking,
				InitialBalance: testhelper.Ptr(decimal.NewFromInt(1000)),
			})
			invalidDestinationAccountId := int64(9999999)

			createTransactionReq := dto.CreateTransactionRequest{
				Description:          "Ghost Transaction",
				Amount:               decimal.NewFromInt(100),
				Type:                 model.Transfer,
				DestinationAccountId: &invalidDestinationAccountId,
				AccountId:            sourceAccountId,
				Date:                 time.Now().UTC(),
			}
			body, _ := json.Marshal(createTransactionReq)
			respRecorder := testhelper.MakeAPIRequest(t, testServer.router, "POST", "/v1/transactions", token, bytes.NewBuffer(body))

			assert.Equal(t, http.StatusBadRequest, respRecorder.Code)
			assert.Contains(t, respRecorder.Body.String(), "destination account not found")
		})
		t.Run("destination account not informed", func(t *testing.T) {
			userId, err := userRepo.Create(ctx, model.User{Name: "Test User", Email: "dontdestinf@test.com", PasswordHash: "hash"})
			require.NoError(err)
			token := testhelper.GenerateTestToken(t, userId, testServer.config.JWTSecretKey)

			sourceAccountId := testhelper.CreateAccount(t, testServer.router, token, dto.AccountRequest{
				Name:           "No-Dest Source",
				Type:           model.Checking,
				InitialBalance: testhelper.Ptr(decimal.NewFromInt(1000)),
			})

			createTransactionReq := dto.CreateTransactionRequest{
				Description:          "Ghost Transaction",
				Amount:               decimal.NewFromInt(100),
				Type:                 model.Transfer,
				AccountId:            sourceAccountId,
				Date:                 time.Now().UTC(),
				DestinationAccountId: nil,
			}
			body, _ := json.Marshal(createTransactionReq)
			respRecorder := testhelper.MakeAPIRequest(t, testServer.router, "POST", "/v1/transactions", token, bytes.NewBuffer(body))

			assert.Equal(t, http.StatusBadRequest, respRecorder.Code)
			assert.Contains(t, respRecorder.Body.String(), "destination account is required for a transfer")
		})
	})
}

func TestBudgetRoutes(t *testing.T) {
	// --- GENERAL ARRANGE ---
	require := require.New(t)
	gin.SetMode(gin.TestMode)

	testhelper.TruncateTables(t, testServer.db)

	ctx := context.Background()
	categoryRepo := repository.NewCategoryRepository(testServer.db)

	t.Run("should create and list budgets with calculated spending", func(t *testing.T) {
		testhelper.TruncateTables(t, testServer.db)

		// Arrange: Create user, token, accounts, and categories
		userRepo := repository.NewUserRepository(testServer.db)
		userId, _ := userRepo.Create(ctx, model.User{Name: "Budget API User", Email: "budgetapi@test.com", PasswordHash: "hash"})
		token := testhelper.GenerateTestToken(t, userId, testServer.config.JWTSecretKey)

		checkingId := testhelper.CreateAccount(t, testServer.router, token, dto.AccountRequest{
			Name:           "Checking Account",
			Type:           model.Checking,
			InitialBalance: testhelper.Ptr(decimal.NewFromInt(0)),
		})
		foodCatId, err := categoryRepo.Create(ctx, model.Category{UserId: userId, Name: "Food", Type: "expense"})
		assert.NoError(t, err)

		// Act 1: Create a budget for Food for the current month
		now := time.Now().UTC()
		createDTO := dto.CreateBudgetRequest{
			CategoryId: foodCatId,
			Amount:     decimal.NewFromInt(500),
			Month:      int(now.Month()),
			Year:       now.Year(),
		}
		createBody, _ := json.Marshal(createDTO)
		recorderCreate := testhelper.MakeAPIRequest(t, testServer.router, "POST", "/v1/budgets", token, bytes.NewBuffer(createBody))
		assert.Equal(t, http.StatusCreated, recorderCreate.Code)

		// Arrange 2: Create some expense transactions in that category for the period
		txRepo := repository.NewTransactionRepository(testServer.db)
		_, _ = txRepo.Create(ctx, model.Transaction{UserId: userId, AccountId: checkingId, CategoryId: &foodCatId, Description: "Groceries", Amount: decimal.NewFromFloat(120.50), Type: "expense", Date: now})
		_, _ = txRepo.Create(ctx, model.Transaction{UserId: userId, AccountId: checkingId, CategoryId: &foodCatId, Description: "Restaurant", Amount: decimal.NewFromFloat(80.00), Type: "expense", Date: now})

		// Act 2: Call the List Budgets endpoint for the current period
		recorderList := testhelper.MakeAPIRequest(t, testServer.router, "GET", "/v1/budgets", token, nil)

		// Assert
		assert.Equal(t, http.StatusOK, recorderList.Code)

		var resp []dto.BudgetResponse
		err = json.Unmarshal(recorderList.Body.Bytes(), &resp)
		require.NoError(err)

		require.Len(resp, 1, "should be one budget")

		foodBudget := resp[0]
		expectedSpent := decimal.NewFromFloat(200.50)   // 120.50 + 80.00
		expectedBalance := decimal.NewFromFloat(299.50) // 500.00 - 200.50

		assert.Equal(t, "Food", foodBudget.CategoryName)
		require.True(decimal.NewFromInt(500).Equal(foodBudget.Amount), "budgeted amount should be correct")
		require.True(expectedSpent.Equal(foodBudget.SpentAmount), "spent amount should be calculated correctly")
		require.True(expectedBalance.Equal(foodBudget.Balance), "remaining balance should be calculated correctly")
	})
}

// TestBusinessScenarios validates complex, multi-step user workflows.
func TestBusinessScenarios(t *testing.T) {
	require := require.New(t)
	gin.SetMode(gin.TestMode)

	ctx := context.Background()
	userRepo := repository.NewUserRepository(testServer.db)

	t.Run("Scenario: 'I Messed Up My Entry' Correction Flow", func(t *testing.T) {
		// This test simulates a user creating, editing, and deleting a transaction
		// and verifies the account balance is correctly recalculated at each step.
		testhelper.TruncateTables(t, testServer.db)

		// Arrange 1: Create a user and an account with an initial balance.
		userId, _ := userRepo.Create(ctx, model.User{Name: "Correction User", Email: "correction@test.com", PasswordHash: "hash"})
		token := testhelper.GenerateTestToken(t, userId, testServer.config.JWTSecretKey)
		accountId := testhelper.CreateAccount(t, testServer.router, token, dto.AccountRequest{
			Name:           "Checking Account",
			Type:           model.Checking,
			InitialBalance: testhelper.Ptr(decimal.NewFromInt(500)),
		})

		// Assert 1: Initial balance is correct.
		balance := testhelper.GetAccountBalance(t, testServer.router, token, accountId)
		require.True(decimal.NewFromInt(500).Equal(balance))

		// Act 1: Create an incorrect expense of $100
		expenseId := testhelper.CreateTransaction(t, testServer.router, token, accountId, "Dinner", "expense", "100.00")

		// Assert 2: Balance is updated correctly after creation.
		balance = testhelper.GetAccountBalance(t, testServer.router, token, accountId)
		assert.Equal(t, decimal.NewFromInt(400), balance, "Balance should be 400 after $100 expense")

		// Act 2: Edit the expense to the correct amount of $80.
		testhelper.UpdateTransaction(t, testServer.router, token, expenseId, accountId, "Dinner (Corrected)", "expense", "80.00")

		// Assert 3: Balance is recalculated correctly after update.
		balance = testhelper.GetAccountBalance(t, testServer.router, token, accountId)
		assert.Equal(t, decimal.NewFromInt(420), balance, "Balance should be $420 after correction to $80")

		// Act 3: Delete the expense entirely.
		testhelper.DeleteTransaction(t, testServer.router, token, expenseId)

		// Assert 4: Balance returns to its original state.
		balance = testhelper.GetAccountBalance(t, testServer.router, token, accountId)
		require.True(decimal.NewFromInt(500).Equal(balance), "Balance should return to 500 after deletion")
	})

	t.Run("Scenario: Credit Card Payment Workflow", func(t *testing.T) {
		// This test simulates paying off a credit card balance from a checking account.
		testhelper.TruncateTables(t, testServer.db)

		// Arrange: Create user, a checking account with $2000, and a credit card with $0.
		userId, _ := userRepo.Create(ctx, model.User{Name: "Payment User", Email: "payment@test.com", PasswordHash: "hash"})
		token := testhelper.GenerateTestToken(t, userId, testServer.config.JWTSecretKey)
		checkingId := testhelper.CreateAccount(t, testServer.router, token, dto.AccountRequest{
			Name:           "My Checking",
			Type:           model.Checking,
			InitialBalance: testhelper.Ptr(decimal.NewFromInt(2000)),
		})
		creditCardId := testhelper.CreateAccount(t, testServer.router, token, dto.AccountRequest{
			Name:                "My Credit Card",
			Type:                model.CreditCard,
			InitialBalance:      testhelper.Ptr(decimal.NewFromInt(0)),
			CreditLimit:         testhelper.Ptr(decimal.NewFromInt(1000)),
			StatementClosingDay: testhelper.Ptr(28),
			PaymentDueDay:       testhelper.Ptr(5),
		})

		// Arrange: Add $450 in expenses to the credit card.
		testhelper.CreateTransaction(t, testServer.router, token, creditCardId, "Groceries", "expense", "150.00")
		testhelper.CreateTransaction(t, testServer.router, token, creditCardId, "Online Shopping", "expense", "300.00")

		// Assert 1: Verify pre-payment balances.
		require.True(decimal.NewFromInt(2000).Equal(testhelper.GetAccountBalance(t, testServer.router, token, checkingId)))
		require.True(decimal.NewFromInt(-450).Equal(testhelper.GetAccountBalance(t, testServer.router, token, creditCardId)), "Credit card balance should be negative after expenses")

		// Act: Pay the credit card bill by transferring $450 from Checking to Credit Card.
		testhelper.CreateTransfer(t, testServer.router, token, "CC Payment", "450.00", checkingId, creditCardId)

		// Assert 2: Verify final balances.
		assert.True(t, decimal.NewFromInt(1550).Equal(testhelper.GetAccountBalance(t, testServer.router, token, checkingId)), "Checking account balance should decrease")
		assert.True(t, decimal.Zero.Equal(testhelper.GetAccountBalance(t, testServer.router, token, creditCardId)), "Credit card balance should be zero after payment")
	})

	t.Run("Scenario: Application-Level Cascade Delete", func(t *testing.T) {
		// This test verifies that deleting an account successfully deletes all associated transactions.
		testhelper.TruncateTables(t, testServer.db)

		// Arrange: Create user and multiple accounts/transactions.
		userId, _ := userRepo.Create(ctx, model.User{Name: "Cascade User", Email: "cascade@test.com", PasswordHash: "hash"})
		token := testhelper.GenerateTestToken(t, userId, testServer.config.JWTSecretKey)

		accountToDeleteId := testhelper.CreateAccount(t, testServer.router, token, dto.AccountRequest{
			Name:           "Account To Delete",
			Type:           model.Checking,
			InitialBalance: testhelper.Ptr(decimal.NewFromInt(100)),
		})
		otherAccountId := testhelper.CreateAccount(t, testServer.router, token, dto.AccountRequest{
			Name:           "Other Account",
			Type:           model.Savings,
			InitialBalance: testhelper.Ptr(decimal.NewFromInt(200)),
		})

		tx1Id := testhelper.CreateTransaction(t, testServer.router, token, accountToDeleteId, "Expense from deleted account", "expense", "10")
		tx2Id := testhelper.CreateTransfer(t, testServer.router, token, "Transfer from deleted account", "20", accountToDeleteId, otherAccountId)
		tx3Id := testhelper.CreateTransaction(t, testServer.router, token, otherAccountId, "Expense from other account", "expense", "30")

		// Act: Delete the account that has history.
		testhelper.DeleteAccount(t, testServer.router, token, accountToDeleteId)

		// Assert: Verify resources were deleted or preserved correctly.
		// 1. The account itself is gone.
		testhelper.AssertAccountNotFound(t, testServer.router, token, accountToDeleteId)
		// 2. Transactions linked to the deleted account are gone.
		testhelper.AssertTransactionNotFound(t, testServer.router, token, tx1Id)
		testhelper.AssertTransactionNotFound(t, testServer.router, token, tx2Id)
		// 3. The unrelated transaction still exists.
		testhelper.AssertTransactionFound(t, testServer.router, token, tx3Id)
	})
}
