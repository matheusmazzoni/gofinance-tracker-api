package testhelper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/api/dto"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// TODO: trocar referencias do body para dtos

// makeAPIRequest is a generic helper to perform an API call.
func MakeAPIRequest(t *testing.T, router *gin.Engine, method, url, token string, body *bytes.Buffer) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, url, body)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func CreateUser(t *testing.T, router *gin.Engine, name, email, password string) int64 {
	createUserReq := dto.CreateUserRequest{
		Name:     name,
		Email:    email,
		Password: password,
	}
	body, _ := json.Marshal(createUserReq)

	recorder := MakeAPIRequest(t, router, "POST", "/v1/users", "", bytes.NewBuffer(body))
	require.Equal(t, http.StatusCreated, recorder.Code)

	var resp dto.UserResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	return resp.Id
}

func CreateAccount(t *testing.T, router *gin.Engine, token, name string, accType model.AccountType, initialBalance decimal.Decimal) int64 {
	createAccountReq := dto.CreateAccountRequest{
		Name:           name,
		Type:           accType,
		InitialBalance: initialBalance,
	}
	body, _ := json.Marshal(createAccountReq)

	recorder := MakeAPIRequest(t, router, "POST", "/v1/accounts", token, bytes.NewBuffer(body))
	require.Equal(t, http.StatusCreated, recorder.Code)

	var resp dto.CreateAccountResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	return resp.Id
}

func GetAccountBalance(t *testing.T, router *gin.Engine, token string, accountID int64) decimal.Decimal {
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

func CreateTransaction(t *testing.T, router *gin.Engine, token string, accountId int64, description, txType, amount string) int64 {
	createTransactionReq := dto.CreateTransactionRequest{
		Description: description,
		Amount:      decimal.RequireFromString(amount),
		Date:        time.Now(),
		Type:        model.TransactionType(txType),
		AccountId:   accountId,
		CategoryId:  nil,
	}
	body, _ := json.Marshal(createTransactionReq)
	req, _ := http.NewRequest("POST", "/v1/transactions", bytes.NewBuffer(body))
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

func UpdateTransaction(t *testing.T, router *gin.Engine, token string, txID, accountId int64, description, txType, amount string) {
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
	require.Equal(t, http.StatusOK, recorder.Code)
}

func DeleteTransaction(t *testing.T, router *gin.Engine, token string, txID int64) {
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/v1/transactions/%d", txID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusNoContent, recorder.Code)
}

func CreateTransfer(t *testing.T, router *gin.Engine, token, description, amount string, fromAccountID, toAccountID int64) int64 {
	createTransferReq := dto.CreateTransactionRequest{
		Description:          description,
		Amount:               decimal.RequireFromString(amount),
		Date:                 time.Now(),
		Type:                 model.Transfer,
		AccountId:            fromAccountID,
		DestinationAccountId: &toAccountID,
		CategoryId:           nil,
	}
	body, _ := json.Marshal(createTransferReq)

	req, _ := http.NewRequest("POST", "/v1/transactions", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusCreated, recorder.Code)

	var resp dto.TransactionResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	return resp.Id
}

func DeleteAccount(t *testing.T, router *gin.Engine, token string, accountID int64) {
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/v1/accounts/%d", accountID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusNoContent, recorder.Code)
}

func AssertAccountNotFound(t *testing.T, router *gin.Engine, token string, accountID int64) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("/v1/accounts/%d", accountID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func AssertTransactionNotFound(t *testing.T, router *gin.Engine, token string, transactionID int64) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("/v1/transactions/%d", transactionID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func AssertTransactionFound(t *testing.T, router *gin.Engine, token string, transactionID int64) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("/v1/transactions/%d", transactionID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusOK, recorder.Code)
}

// GenerateTestToken é um helper para criar um token JWT válido para nossos testes.
func GenerateTestToken(t *testing.T, userId int64, secretKey string) string {
	claims := &jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 1)),
		Subject:   fmt.Sprintf("%d", userId),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	require.NoError(t, err)
	return tokenString
}
