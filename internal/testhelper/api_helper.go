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

// MakeAPIRequest is a generic helper to perform an API call against the test router.
// It builds a request, sets the necessary headers, executes the request, and returns the response recorder.
func MakeAPIRequest(t *testing.T, router *gin.Engine, method, url, token string, body *bytes.Buffer) *httptest.ResponseRecorder {
	// If body is nil, create an empty buffer to avoid errors with http.NewRequest.
	if body == nil {
		body = &bytes.Buffer{}
	}

	req, err := http.NewRequest(method, url, body)
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

// CreateAccount is a test helper to create a new account via an API call.
// It asserts for a successful creation and returns the new account's Id.
func CreateAccount(t *testing.T, router *gin.Engine, token string, createAccountReq dto.AccountRequest) int64 {

	body, err := json.Marshal(createAccountReq)
	require.NoError(t, err)

	recorder := MakeAPIRequest(t, router, http.MethodPost, "/v1/accounts", token, bytes.NewBuffer(body))
	require.Equal(t, http.StatusCreated, recorder.Code)

	var resp dto.AccountResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	return resp.Id
}

// GetAccountBalance is a test helper to fetch a specific account and return its balance.
func GetAccountBalance(t *testing.T, router *gin.Engine, token string, accountId int64) decimal.Decimal {
	url := fmt.Sprintf("/v1/accounts/%d", accountId)
	recorder := MakeAPIRequest(t, router, http.MethodGet, url, token, nil)
	require.Equal(t, http.StatusOK, recorder.Code)

	var resp dto.AccountResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	return *resp.Balance
}

// CreateTransaction is a test helper to create a new transaction via an API call.
// It asserts for a successful creation and returns the new transaction's Id.
func CreateTransaction(t *testing.T, router *gin.Engine, token string, accountId int64, description, txType, amount string) int64 {
	createTransactionReq := dto.CreateTransactionRequest{
		Description: description,
		Amount:      decimal.RequireFromString(amount),
		Date:        time.Now().UTC(),
		Type:        model.TransactionType(txType),
		AccountId:   accountId,
		CategoryId:  nil,
	}
	body, err := json.Marshal(createTransactionReq)
	require.NoError(t, err)

	recorder := MakeAPIRequest(t, router, http.MethodPost, "/v1/transactions", token, bytes.NewBuffer(body))
	require.Equal(t, http.StatusCreated, recorder.Code)

	var resp dto.TransactionResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	return resp.Id
}

// UpdateTransaction is a test helper to update an existing transaction via an API call.
// It asserts that the update operation was successful.
func UpdateTransaction(t *testing.T, router *gin.Engine, token string, txId, accountId int64, description, txType, amount string) {
	updateDTO := dto.UpdateTransactionRequest{
		Description: description,
		Amount:      decimal.RequireFromString(amount),
		Date:        time.Now().UTC(),
		Type:        model.TransactionType(txType),
		AccountId:   accountId,
		CategoryId:  nil,
	}
	body, err := json.Marshal(updateDTO)
	require.NoError(t, err)

	url := fmt.Sprintf("/v1/transactions/%d", txId)
	recorder := MakeAPIRequest(t, router, http.MethodPut, url, token, bytes.NewBuffer(body))
	require.Equal(t, http.StatusOK, recorder.Code)
}

// DeleteTransaction is a test helper to delete a transaction via an API call.
// It asserts that the deletion was successful.
func DeleteTransaction(t *testing.T, router *gin.Engine, token string, txId int64) {
	url := fmt.Sprintf("/v1/transactions/%d", txId)
	recorder := MakeAPIRequest(t, router, http.MethodDelete, url, token, nil)
	require.Equal(t, http.StatusNoContent, recorder.Code)
}

// CreateTransfer is a test helper to create a transfer transaction via an API call.
// It asserts for a successful creation and returns the new transaction's Id.
func CreateTransfer(t *testing.T, router *gin.Engine, token, description, amount string, fromAccountId, toAccountId int64) int64 {
	createTransferReq := dto.CreateTransactionRequest{
		Description:          description,
		Amount:               decimal.RequireFromString(amount),
		Date:                 time.Now().UTC(),
		Type:                 model.Transfer,
		AccountId:            fromAccountId,
		DestinationAccountId: &toAccountId,
		CategoryId:           nil,
	}
	body, err := json.Marshal(createTransferReq)
	require.NoError(t, err)

	recorder := MakeAPIRequest(t, router, http.MethodPost, "/v1/transactions", token, bytes.NewBuffer(body))
	require.Equal(t, http.StatusCreated, recorder.Code)

	var resp dto.TransactionResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	return resp.Id
}

// DeleteAccount is a test helper to delete an account via an API call.
// It asserts that the deletion was successful.
func DeleteAccount(t *testing.T, router *gin.Engine, token string, accountId int64) {
	url := fmt.Sprintf("/v1/accounts/%d", accountId)
	recorder := MakeAPIRequest(t, router, http.MethodDelete, url, token, nil)
	require.Equal(t, http.StatusNoContent, recorder.Code)
}

// AssertAccountNotFound is a test helper to verify that a GET request for an account returns a 404 Not Found status.
func AssertAccountNotFound(t *testing.T, router *gin.Engine, token string, accountId int64) {
	url := fmt.Sprintf("/v1/accounts/%d", accountId)
	recorder := MakeAPIRequest(t, router, http.MethodGet, url, token, nil)
	require.Equal(t, http.StatusNotFound, recorder.Code)
}

// AssertTransactionNotFound is a test helper to verify that a GET request for a transaction returns a 404 Not Found status.
func AssertTransactionNotFound(t *testing.T, router *gin.Engine, token string, transactionId int64) {
	url := fmt.Sprintf("/v1/transactions/%d", transactionId)
	recorder := MakeAPIRequest(t, router, http.MethodGet, url, token, nil)
	require.Equal(t, http.StatusNotFound, recorder.Code)
}

// AssertTransactionFound is a test helper to verify that a GET request for a transaction returns a 200 OK status.
func AssertTransactionFound(t *testing.T, router *gin.Engine, token string, transactionId int64) {
	url := fmt.Sprintf("/v1/transactions/%d", transactionId)
	recorder := MakeAPIRequest(t, router, http.MethodGet, url, token, nil)
	require.Equal(t, http.StatusOK, recorder.Code)
}

// GenerateTestToken creates a valid JWT for testing purposes.
// It uses the provided user Id as the subject of the token.
func GenerateTestToken(t *testing.T, userId int64, secretKey string) string {
	claims := &jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Hour * 1)),
		Subject:   fmt.Sprintf("%d", userId),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	require.NoError(t, err)
	return tokenString
}
