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
	"github.com/jmoiron/sqlx"
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
	testLogger = zerolog.Nop()
	testCfg    = config.Config{JWTSecretKey: "account_handler_test_key"}
	testDB     *sqlx.DB
)

func TestMain(m *testing.M) {
	var pgContainer testcontainers.Container
	testDB, pgContainer = testhelper.SetupTestDB()
	exitCode := m.Run()

	if err := pgContainer.Terminate(context.Background()); err != nil {
		log.Fatalf("failed to terminate container: %v", err)
	}

	os.Exit(exitCode)
}

// generateTestToken é um helper para criar um token JWT válido para nossos testes.
func generateTestToken(t *testing.T, userId int64, secretKey string) string {
	claims := &jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 1)),
		Subject:   fmt.Sprintf("%d", userId),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	require.NoError(t, err)
	return tokenString
}

// Middleware
func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Cria uma instância do middleware com uma chave secreta de teste e um logger "mudo".
	authMiddleware := middleware.AuthMiddleware(testCfg.JWTSecretKey, zerolog.Nop())

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
		require.Equal(t, http.StatusUnauthorized, recorder.Code)

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
		require.Equal(t, http.StatusUnauthorized, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "bearer token required")
	})

	t.Run("should return 401 Unauthorized when token is invalid or signed with wrong key", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		// Gera um token com uma chave secreta DIFERENTE
		invalidToken := generateTestToken(t, 123, "this_is_the_wrong_key")

		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+invalidToken)

		// Act
		router.ServeHTTP(recorder, req)

		// Assert
		require.Equal(t, http.StatusUnauthorized, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "invalid token")
	})

	t.Run("should return 401 Unauthorized when token is expired", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		// Gera um token que expirou há uma hora.
		claims := &jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			Subject:   "123",
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		expiredToken, _ := token.SignedString([]byte(testCfg.JWTSecretKey))

		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+expiredToken)

		// Act
		router.ServeHTTP(recorder, req)

		// Assert
		require.Equal(t, http.StatusUnauthorized, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "invalid token")
	})

	t.Run("should grant access when token is valid", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		validToken := generateTestToken(t, 123, testCfg.JWTSecretKey)

		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)

		// Act
		router.ServeHTTP(recorder, req)

		// Assert
		require.Equal(t, http.StatusOK, recorder.Code)
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
		validToken := generateTestToken(t, 123, testCfg.JWTSecretKey)

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

	testhelper.TruncateTables(t, testDB)

	server := NewServer(testCfg, testDB, &testLogger)

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
		server.router.ServeHTTP(recorder, req)

		require.Equal(http.StatusCreated, recorder.Code, "the user should be created successfully")
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
		server.router.ServeHTTP(recorder1, req1)

		// Garante que o primeiro usuário foi criado com sucesso.
		require.Equal(http.StatusCreated, recorder1.Code, "the first user should be created successfully")

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

		// Act: Faz a segunda requisição, que deve falhar.
		server.router.ServeHTTP(recorder2, req2)

		// Assert: Verifica se a resposta foi a esperada.
		require.Equal(http.StatusBadRequest, recorder2.Code, "expected status 400 for duplicate email")

		var errorResponse dto.ErrorResponse
		err := json.Unmarshal(recorder2.Body.Bytes(), &errorResponse)
		require.NoError(err, "should be able to unmarshal error response")

		assert.Contains(t, errorResponse.Error, "a user with this email already exists")
	})
}
func TestLoginRoutes(t *testing.T) {
	require := require.New(t)
	gin.SetMode(gin.TestMode)

	testhelper.TruncateTables(t, testDB)

	userRepo := repository.NewUserRepository(testDB)
	userService := service.NewUserService(userRepo)

	email := "login@test.com"
	password := "login123"
	_, err := userService.CreateUser(context.Background(), model.User{Name: "Login Routes Test User", Email: email, Password: password})
	require.NoError(err)

	server := NewServer(testCfg, testDB, &testLogger)

	t.Run("successfuly login", func(t *testing.T) {
		loginRequest := dto.LoginRequest{
			Email:    email,
			Password: password,
		}
		data, _ := json.Marshal(loginRequest)

		req, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(data))
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		server.router.ServeHTTP(recorder, req)

		require.Equal(http.StatusOK, recorder.Code)

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
		server.router.ServeHTTP(recorder, req)

		require.Equal(http.StatusUnauthorized, recorder.Code)

		var errorResponse dto.ErrorResponse
		_ = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
		require.Equal(errorResponse.Error, "invalid credentials")
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
		server.router.ServeHTTP(recorder, req)

		require.Equal(http.StatusUnauthorized, recorder.Code)

		var errorResponse dto.ErrorResponse
		_ = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
		require.Equal(errorResponse.Error, "invalid credentials")
	})
}
func TestAccountRoutes(t *testing.T) {
	require := require.New(t)
	gin.SetMode(gin.TestMode)

	testhelper.TruncateTables(t, testDB)

	server := NewServer(testCfg, testDB, &testLogger)

	ctx := context.Background()
	accountRepo := repository.NewAccountRepository(testDB)
	userRepo := repository.NewUserRepository(testDB)

	t.Run("create an account successfully", func(t *testing.T) {
		userId, _ := userRepo.Create(ctx, model.User{Name: "UserSuccessfullyCreated", Email: "success@test.com", PasswordHash: "hash"})
		token := generateTestToken(t, userId, testCfg.JWTSecretKey)

		accountDTO := dto.CreateAccountRequest{
			Name:           "My API Test Account",
			Type:           model.Checking,
			InitialBalance: decimal.NewFromInt(100),
		}
		body, _ := json.Marshal(accountDTO)

		req, _ := http.NewRequest("POST", "/v1/accounts", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		recorder := httptest.NewRecorder()

		server.router.ServeHTTP(recorder, req)

		require.Equal(http.StatusCreated, recorder.Code)

		var responseBody dto.CreateAccountResponse
		err := json.Unmarshal(recorder.Body.Bytes(), &responseBody)
		require.NoError(err)
		require.NotZero(responseBody.Id)
	})
	t.Run("create multiple accounts of different types", func(t *testing.T) {
		userId, _ := userRepo.Create(ctx, model.User{Name: "MultipleAccountsDiffTypes", Email: "multipleAccountsDiffTypes@test.com", PasswordHash: "hash"})
		token := generateTestToken(t, userId, testCfg.JWTSecretKey)

		checkingAccountDTO := dto.CreateAccountRequest{
			Name:           "Checking Account",
			Type:           model.Checking,
			InitialBalance: decimal.NewFromInt(1000),
		}
		checkingBody, _ := json.Marshal(checkingAccountDTO)
		reqChecking, _ := http.NewRequest("POST", "/v1/accounts", bytes.NewBuffer(checkingBody))
		reqChecking.Header.Set("Content-Type", "application/json")
		reqChecking.Header.Set("Authorization", "Bearer "+token)
		recorderChecking := httptest.NewRecorder()

		// --- ACT 1: Create the "Checking Account" ---
		server.router.ServeHTTP(recorderChecking, reqChecking)

		// --- ASSERT 1: Verify the first creation was successful ---
		require.Equal(http.StatusCreated, recorderChecking.Code, "should create checking account")

		// --- ARRANGE 2: Prepare the "Credit Card" request ---
		creditCardDTO := dto.CreateAccountRequest{
			Name:           "Primary Credit Card",
			Type:           model.CreditCard,
			InitialBalance: decimal.NewFromInt(0),
		}
		creditCardBody, _ := json.Marshal(creditCardDTO)
		reqCreditCard, _ := http.NewRequest("POST", "/v1/accounts", bytes.NewBuffer(creditCardBody))
		reqCreditCard.Header.Set("Content-Type", "application/json")
		reqCreditCard.Header.Set("Authorization", "Bearer "+token)
		recorderCreditCard := httptest.NewRecorder()

		// --- ACT 2: Create the "Credit Card" ---
		server.router.ServeHTTP(recorderCreditCard, reqCreditCard)

		// --- ASSERT 2: Verify the second creation was successful ---
		require.Equal(http.StatusCreated, recorderCreditCard.Code, "should create credit card")

		// Arrange
		reqList, _ := http.NewRequest("GET", "/v1/accounts", nil)
		reqList.Header.Set("Authorization", "Bearer "+token)
		recorderList := httptest.NewRecorder()

		// Act
		server.router.ServeHTTP(recorderList, reqList)

		// Assert
		require.Equal(http.StatusOK, recorderList.Code)

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
				assert.True(t, decimal.NewFromInt(1000).Equal(AccountResponse.Balance))
			}
			if AccountResponse.Name == "Primary Credit Card" && AccountResponse.Type == model.CreditCard {
				foundCreditCard = true
				assert.True(t, decimal.NewFromInt(0).Equal(AccountResponse.Balance))
			}
		}

		assert.True(t, foundChecking, "the checking account should have been found in the list")
		assert.True(t, foundCreditCard, "the credit card account should have been found in the list")
	})
	t.Run("create account with bad request body", func(t *testing.T) {
		userId, _ := userRepo.Create(ctx, model.User{Name: "AccountBadRequest", Email: "accountBadRequest@test.com", PasswordHash: "hash"})
		token := generateTestToken(t, userId, testCfg.JWTSecretKey)

		badRequestBody := `{"type": "checking", "initial_balance": 100}`

		req, _ := http.NewRequest("POST", "/v1/accounts", bytes.NewBuffer([]byte(badRequestBody)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		recorder := httptest.NewRecorder()

		// Act
		server.router.ServeHTTP(recorder, req)

		// Assert
		require.Equal(http.StatusBadRequest, recorder.Code)
	})
	t.Run("delete account without transactions", func(t *testing.T) {
		userId, _ := userRepo.Create(ctx, model.User{Name: "DeleteAccountWitoutTransactions", Email: "deleteAccountWitoutTransactions@test.com", PasswordHash: "hash"})
		token := generateTestToken(t, userId, testCfg.JWTSecretKey)

		createAccountBody := `{"name": "DeleteAccountWithoutTransactions", "type": "checking", "initial_balance": 1000}`
		req, _ := http.NewRequest("POST", "/v1/accounts", bytes.NewBuffer([]byte(createAccountBody)))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		server.router.ServeHTTP(recorder, req)
		require.Equal(http.StatusCreated, recorder.Code)

		var createAccountResp dto.CreateAccountResponse
		_ = json.Unmarshal(recorder.Body.Bytes(), &createAccountResp)
		accountId := createAccountResp.Id
		require.NotZero(accountId)

		// Delete Account
		req, _ = http.NewRequest("DELETE", fmt.Sprintf("/v1/accounts/%d", accountId), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder = httptest.NewRecorder()
		server.router.ServeHTTP(recorder, req)
		require.Equal(http.StatusNoContent, recorder.Code)

	})
	t.Run("delete account with transactions", func(t *testing.T) {
		userId, _ := userRepo.Create(ctx, model.User{Name: "DeleteAccountWithTransactions", Email: "deleteAccountWithTransactions@test.com", PasswordHash: "hash"})
		token := generateTestToken(t, userId, testCfg.JWTSecretKey)

		createAccountBody := `{"name": "AccountDeleteScenario", "type": "checking", "initial_balance": 1000}`
		req, _ := http.NewRequest("POST", "/v1/accounts", bytes.NewBuffer([]byte(createAccountBody)))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		server.router.ServeHTTP(recorder, req)
		require.Equal(http.StatusCreated, recorder.Code)

		var createAccountResp dto.CreateAccountResponse
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
			server.router.ServeHTTP(recorder, req)
			require.Equal(http.StatusCreated, recorder.Code)

			var txResp map[string]int64
			_ = json.Unmarshal(recorder.Body.Bytes(), &txResp)
			transactionIds = append(transactionIds, txResp["id"])
		}
		require.Len(transactionIds, 3)

		// Delete Account
		req, _ = http.NewRequest("DELETE", fmt.Sprintf("/v1/accounts/%d", accountId), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder = httptest.NewRecorder()
		server.router.ServeHTTP(recorder, req)
		require.Equal(http.StatusNoContent, recorder.Code)

	})
	t.Run("update an account name successfully", func(t *testing.T) {
		userId, _ := userRepo.Create(ctx, model.User{Name: "MultipleAccounts", Email: "multiAccounts@test.com", PasswordHash: "hash"})
		token := generateTestToken(t, userId, testCfg.JWTSecretKey)

		accountDTO := dto.CreateAccountRequest{
			Name:           "My API Test Account",
			Type:           model.Checking,
			InitialBalance: decimal.NewFromInt(100),
		}
		body, _ := json.Marshal(accountDTO)

		req, _ := http.NewRequest("POST", "/v1/accounts", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		recorder := httptest.NewRecorder()

		server.router.ServeHTTP(recorder, req)

		require.Equal(http.StatusCreated, recorder.Code)

		var responseBody dto.CreateAccountResponse
		err := json.Unmarshal(recorder.Body.Bytes(), &responseBody)
		require.NoError(err)

		accountId := responseBody.Id
		require.NotZero(accountId)

		updateDTO := dto.UpdateAccountRequest{
			Name:           "User A Main Account",
			Type:           model.Checking,
			InitialBalance: decimal.NewFromInt(0),
		}
		body, _ = json.Marshal(updateDTO)
		req, _ = http.NewRequest("PUT", fmt.Sprintf("/v1/accounts/%d", accountId), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		recorder = httptest.NewRecorder()

		server.router.ServeHTTP(recorder, req)

		require.Equal(http.StatusOK, recorder.Code)
		var resp dto.AccountResponse
		_ = json.Unmarshal(recorder.Body.Bytes(), &resp)
		assert.Equal(t, "User A Main Account", resp.Name)
	})
	t.Run("accounts belonging to the authenticated user", func(t *testing.T) {
		userId, _ := userRepo.Create(ctx, model.User{Name: "Authenticated", Email: "autheticated@test.com", PasswordHash: "hash"})
		token := generateTestToken(t, userId, testCfg.JWTSecretKey)

		_, _ = accountRepo.Create(ctx, model.Account{UserId: userId, Name: "My API Test Account"})

		req, _ := http.NewRequest("GET", "/v1/accounts", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()

		server.router.ServeHTTP(recorder, req)

		require.Equal(http.StatusOK, recorder.Code)
		var resp []dto.AccountResponse
		_ = json.Unmarshal(recorder.Body.Bytes(), &resp)

		// User A has only one account at this point: "User A Updated Main Account"
		require.Len(resp, 1)
		assert.Equal(t, "My API Test Account", resp[0].Name)
	})
	t.Run("duplicate account name for the same user", func(t *testing.T) {
		name := "Same Account"
		userId, _ := userRepo.Create(ctx, model.User{Name: "DuplicateAccountSameUser", Email: "duplicateAccountSameUser@test.com", PasswordHash: "hash"})
		token := generateTestToken(t, userId, testCfg.JWTSecretKey)

		firstAccount := dto.CreateAccountRequest{
			Name:           name,
			Type:           model.Checking,
			InitialBalance: decimal.NewFromInt(1000),
		}
		firstAccountBody, _ := json.Marshal(firstAccount)
		requeFirstAccount, _ := http.NewRequest("POST", "/v1/accounts", bytes.NewBuffer(firstAccountBody))
		requeFirstAccount.Header.Set("Content-Type", "application/json")
		requeFirstAccount.Header.Set("Authorization", "Bearer "+token)
		recorderFisrtAccount := httptest.NewRecorder()

		server.router.ServeHTTP(recorderFisrtAccount, requeFirstAccount)

		require.Equal(http.StatusCreated, recorderFisrtAccount.Code)

		secondAccount := dto.CreateAccountRequest{
			Name:           name,
			Type:           model.CreditCard,
			InitialBalance: decimal.NewFromInt(0),
		}
		secondAccountBody, _ := json.Marshal(secondAccount)
		reqSecondAccount, _ := http.NewRequest("POST", "/v1/accounts", bytes.NewBuffer(secondAccountBody))
		reqSecondAccount.Header.Set("Content-Type", "application/json")
		reqSecondAccount.Header.Set("Authorization", "Bearer "+token)
		recorderSecondAccount := httptest.NewRecorder()

		server.router.ServeHTTP(recorderSecondAccount, reqSecondAccount)

		// --- ASSERT 2: Verify the second creation was successful ---
		require.Equal(http.StatusBadRequest, recorderSecondAccount.Code)

		var errorResponse dto.ErrorResponse
		err := json.Unmarshal(recorderSecondAccount.Body.Bytes(), &errorResponse)
		require.NoError(err)

		require.Contains(errorResponse.Error, "an account with this name already exists")
	})
}
func TestTransactionRoutes(t *testing.T) {
	require := require.New(t)
	gin.SetMode(gin.TestMode)

	testhelper.TruncateTables(t, testDB)

	server := NewServer(testCfg, testDB, &testLogger)

	ctx := context.Background()
	accountRepo := repository.NewAccountRepository(testDB)
	userRepo := repository.NewUserRepository(testDB)
	categoryRepo := repository.NewCategoryRepository(testDB)

	// Create a test user and generate a token
	userId, _ := userRepo.Create(ctx, model.User{Name: "Transaction User", Email: "tx.user@example.com", PasswordHash: "hash"})
	token := generateTestToken(t, userId, testCfg.JWTSecretKey)

	// Create prerequisite accounts and a category for the user
	checkingAccountId, _ := accountRepo.Create(ctx, model.Account{UserId: userId, Name: "Checking Account", Type: model.Checking})
	savingsAccountId, _ := accountRepo.Create(ctx, model.Account{UserId: userId, Name: "Savings Account", Type: model.Savings})
	foodCategoryId, _ := categoryRepo.Create(ctx, model.Category{UserId: userId, Name: "Food"})

	var createdExpenseId int64

	t.Run("CreateTransactions", func(t *testing.T) {
		t.Run("should create an expense transaction successfully", func(t *testing.T) {
			// Arrange
			expenseDTO := dto.CreateTransactionRequest{
				Description: "Groceries",
				Amount:      decimal.NewFromFloat(75.50),
				Date:        time.Now(),
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
			server.router.ServeHTTP(recorder, req)

			// Assert
			require.Equal(http.StatusCreated, recorder.Code)
			var resp dto.TransactionResponse
			err := json.Unmarshal(recorder.Body.Bytes(), &resp)
			require.NoError(err)
			require.NotZero(resp.Id)
			createdExpenseId = resp.Id // Save for later tests
		})

		t.Run("should create a transfer transaction successfully", func(t *testing.T) {
			// Arrange
			transferDTO := dto.CreateTransactionRequest{
				Description:          "Move to savings",
				Amount:               decimal.NewFromInt(500),
				Date:                 time.Now(),
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
			server.router.ServeHTTP(recorder, req)

			// Assert
			require.Equal(http.StatusCreated, recorder.Code)
		})

		t.Run("should fail with 400 Bad Request if destination account is missing for a transfer", func(t *testing.T) {
			// Arrange
			invalidTransferDTO := dto.CreateTransactionRequest{
				Description: "Invalid Transfer",
				Amount:      decimal.NewFromInt(100),
				Date:        time.Now(),
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
			server.router.ServeHTTP(recorder, req)

			// Assert
			require.Equal(http.StatusBadRequest, recorder.Code)
			assert.Contains(t, recorder.Body.String(), "destination account is required for a transfer")
		})
	})
	t.Run("GetTransaction", func(t *testing.T) {
		require.NotZero(t, createdExpenseId, "Create test must run first")

		t.Run("should get a transaction by Id successfully", func(t *testing.T) {
			// Arrange
			url := fmt.Sprintf("/v1/transactions/%d", createdExpenseId)
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			recorder := httptest.NewRecorder()

			// Act
			server.router.ServeHTTP(recorder, req)

			// Assert
			require.Equal(http.StatusOK, recorder.Code)
			var resp dto.TransactionResponse
			err := json.Unmarshal(recorder.Body.Bytes(), &resp)
			require.NoError(err)
			assert.Equal(t, "Groceries", resp.Description)
			assert.Equal(t, "Checking Account", resp.AccountName)
			assert.Equal(t, "Food", *resp.CategoryName)
		})
	})
	t.Run("PatchTransaction", func(t *testing.T) {
		require.NotZero(t, createdExpenseId, "Create test must run first")

		t.Run("should partially update a transaction", func(t *testing.T) {
			// Arrange
			patchDTO := dto.PatchTransactionRequest{
				Description: ptr("Expensive Groceries"),
			}
			body, _ := json.Marshal(patchDTO)
			url := fmt.Sprintf("/v1/transactions/%d", createdExpenseId)
			req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			recorder := httptest.NewRecorder()

			// Act
			server.router.ServeHTTP(recorder, req)

			// Assert
			require.Equal(http.StatusOK, recorder.Code)
			var resp dto.TransactionResponse
			err := json.Unmarshal(recorder.Body.Bytes(), &resp)
			require.NoError(err)
			// Check that the description changed
			assert.Equal(t, "Expensive Groceries", resp.Description)
			// Check that the amount remained the same
			assert.True(t, decimal.NewFromFloat(75.50).Equal(resp.Amount))
		})
	})
	t.Run("UpdateTransaction", func(t *testing.T) {

		// Create a new user and account for this test
		userId, _ := userRepo.Create(ctx, model.User{Name: "Update User", Email: "update@test.com", PasswordHash: "hash"})
		token := generateTestToken(t, userId, testCfg.JWTSecretKey)
		accountId := createAccount(t, server.router, token, "Account for Update Test", "checking", "1000")

		// Create the initial transaction that we are going to update
		initialDescription := "Initial Dinner"
		initialAmount := "50.00"
		txId := createTransaction(t, server.router, token, accountId, initialDescription, "expense", initialAmount)

		// Prepare the update request with new data
		updateDTO := dto.UpdateTransactionRequest{
			Description: "Updated Dinner with Friends",
			Amount:      decimal.NewFromFloat(65.50),
			Date:        time.Now(),
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
		server.router.ServeHTTP(recorder, req)
		fmt.Print(recorder)

		// ASSERT 1: Check the immediate response from the PUT request
		require.Equal(http.StatusOK, recorder.Code, "the update request should succeed")

		var updateResponse dto.TransactionResponse
		err := json.Unmarshal(recorder.Body.Bytes(), &updateResponse)
		require.NoError(err)

		// Verify the response body contains the updated data
		require.Equal("Updated Dinner with Friends", updateResponse.Description)
		require.True(decimal.NewFromFloat(65.50).Equal(updateResponse.Amount), "the amount in the response should be updated")

		// FINAL VERIFICATION: Fetch the resource again to ensure the change was persisted
		// Arrange for GET
		getReq, _ := http.NewRequest("GET", url, nil)
		getReq.Header.Set("Authorization", "Bearer "+token)
		getRecorder := httptest.NewRecorder()

		// Act for GET
		server.router.ServeHTTP(getRecorder, getReq)

		// Assert for GET
		require.Equal(http.StatusOK, getRecorder.Code)
		var finalResponse dto.TransactionResponse
		err = json.Unmarshal(getRecorder.Body.Bytes(), &finalResponse)
		require.NoError(err)

		require.Equal("Updated Dinner with Friends", finalResponse.Description, "the persisted description should be updated")
		require.True(decimal.NewFromFloat(65.50).Equal(finalResponse.Amount), "the persisted amount should be updated")
	})
}

// ptr is a simple helper function to get a pointer to a value, useful for DTOs.
func ptr[T any](v T) *T {
	return &v
}

// TODO: Fazer para outros Routes
