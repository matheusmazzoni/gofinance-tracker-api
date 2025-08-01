package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/api/dto"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/service"
	"github.com/rs/zerolog"
)

type AccountHandler struct {
	service *service.AccountService
}

func NewAccountHandler(s *service.AccountService) *AccountHandler {
	return &AccountHandler{service: s}
}

// CreateAccount godoc
//
//	@Summary		Create a new account
//	@Description	Adds a new financial account to the logged-in user's system
//	@Tags			accounts
//	@Accept			json
//	@Produce		json
//	@Param			account	body		dto.AccountRequest	true	"Account Data for Creation"
//	@Success		201		{object}	dto.AccountResponse
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		500		{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/accounts [post]
func (h *AccountHandler) CreateAccount(c *gin.Context) {
	var req dto.AccountRequest
	if !dto.BindAndValidate(c, &req) {
		return
	}

	userId := c.MustGet("userId").(int64)
	account := model.Account{
		UserId:              userId,
		Name:                req.Name,
		Type:                req.Type,
		InitialBalance:      *req.InitialBalance,
		CreditLimit:         req.CreditLimit,
		StatementClosingDay: req.StatementClosingDay,
		PaymentDueDay:       req.PaymentDueDay,
	}

	id, err := h.service.CreateAccount(c.Request.Context(), account)
	if err != nil {
		// Check if the error is for a duplicate key (account name)
		if strings.Contains(err.Error(), "unique constraint") {
			dto.SendError(c, http.StatusConflict, "account with this name already exists", nil)
			return
		}
		dto.SendError(c, http.StatusInternalServerError, "failed to create account", map[string]string{
			"message": err.Error(),
		})
		return
	}

	response := dto.AccountResponse{
		Id: id,
	}
	dto.SendSuccessResponse(c, http.StatusCreated, response)
}

// ListAccounts godoc
//
//	@Summary		List all user accounts
//	@Description	Returns an array with all of the logged-in user's accounts
//	@Tags			accounts
//	@Produce		json
//	@Success		200	{array}		dto.AccountResponse
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		500	{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/accounts [get]
func (h *AccountHandler) ListAccounts(c *gin.Context) {
	userId := c.MustGet("userId").(int64)
	accounts, err := h.service.ListAccountsByUserId(c.Request.Context(), userId)
	if err != nil {
		dto.SendError(c, http.StatusInternalServerError, "failed to list accounts", map[string]string{
			"message": err.Error(),
		})
		return
	}

	var responses []dto.AccountResponse
	for _, acc := range accounts {
		responses = append(responses, dto.AccountResponse{
			Id:                  acc.Id,
			Name:                acc.Name,
			Type:                acc.Type,
			Balance:             &acc.Balance,
			StatementClosingDay: acc.StatementClosingDay,
			PaymentDueDay:       acc.PaymentDueDay,
		})
	}
	dto.SendSuccessResponse(c, http.StatusOK, responses)
}

// GetAccount godoc
//
//	@Summary		Get an account by ID
//	@Description	Returns the details of a single account belonging to the logged-in user
//	@Tags			accounts
//	@Produce		json
//	@Param			id	path		int	true	"Account Id"
//	@Success		200	{object}	dto.AccountResponse
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/accounts/{id} [get]
func (h *AccountHandler) GetAccount(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userId := c.MustGet("userId").(int64)
	account, err := h.service.GetAccountById(c.Request.Context(), id, userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			dto.SendError(c, http.StatusNotFound, "account not found", nil)
			return
		}
		dto.SendError(c, http.StatusInternalServerError, "failed to get account", map[string]string{
			"message": err.Error(),
		})
		return
	}

	dto.SendSuccessResponse(c, http.StatusOK, dto.AccountResponse{
		Id:                  account.Id,
		Name:                account.Name,
		Type:                account.Type,
		InitialBalance:      &account.InitialBalance,
		Balance:             &account.Balance,
		StatementClosingDay: account.StatementClosingDay,
		PaymentDueDay:       account.PaymentDueDay,
	})
}

// UpdateAccount godoc
//
//	@Summary		Update an account
//	@Description	Updates the details of an existing account
//	@Tags			accounts
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int					true	"Account ID"
//	@Param			account	body		dto.AccountRequest	true	"Data for Update"
//	@Success		200		{object}	dto.AccountResponse
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		404		{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/accounts/{id} [put]
func (h *AccountHandler) UpdateAccount(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req dto.AccountRequest
	if !dto.BindAndValidate(c, &req) {
		return
	}

	userId := c.MustGet("userId").(int64)

	updatedAcc, err := h.service.UpdateAccount(c.Request.Context(), model.Account{
		Id:     id,
		UserId: userId,
		Name:   req.Name,
		Type:   req.Type,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			dto.SendError(c, http.StatusNotFound, "account not found", nil)
			return
		}
		dto.SendError(c, http.StatusInternalServerError, "failed to update account", map[string]string{
			"message": err.Error(),
		})
		return
	}

	dto.SendSuccessResponse(c, http.StatusOK, dto.AccountResponse{
		Id:                  updatedAcc.Id,
		Name:                updatedAcc.Name,
		Type:                updatedAcc.Type,
		InitialBalance:      &updatedAcc.InitialBalance,
		Balance:             &updatedAcc.Balance,
		StatementClosingDay: updatedAcc.StatementClosingDay,
		PaymentDueDay:       updatedAcc.PaymentDueDay,
	})
}

// DeleteAccount godoc
//
//	@Summary		Delete an account
//	@Description	Removes an account from the system. It will fail if the account has associated transactions.
//	@Tags			accounts
//	@Param			id	path	int	true	"Account ID"
//	@Success		204
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Failure		409	{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/accounts/{id} [delete]
func (h *AccountHandler) DeleteAccount(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userId := c.MustGet("userId").(int64)

	err := h.service.DeleteAccount(c.Request.Context(), id, userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			dto.SendError(c, http.StatusNotFound, "account not found", nil)
			return
		}
		// Check for a foreign key error, indicating the account is in use.
		if strings.Contains(err.Error(), "violates foreign key constraint") {
			dto.SendError(c, http.StatusConflict, "cannot delete account with associated transactions", nil)
			return
		}
		dto.SendError(c, http.StatusInternalServerError, "failed to delete account", map[string]string{
			"message": err.Error(),
		})
		return
	}
	c.Status(http.StatusNoContent)
}

// GetAccountStatement godoc
//
//	@Summary		Get a credit card statement
//	@Description	Retrieves all transactions and balance details for a specific credit card billing cycle. Defaults to the current statement if month/year are not provided.
//	@Tags			accounts
//	@Produce		json
//	@Param			id		path		int	true	"Account Id"
//	@Param			month	query		int	false	"The month of the statement's due date (1-12)"
//	@Param			year	query		int	false	"The year of the statement's due date (e.g., 2025)"
//	@Success		200		{object}	dto.StatementResponse
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		404		{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/accounts/{id}/statement [get]
func (h *AccountHandler) GetAccountStatement(c *gin.Context) {
	logger := zerolog.Ctx(c.Request.Context())

	accountId, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.SendError(c, http.StatusBadRequest, "invalid account Id format", nil)
		return
	}
	userId := c.MustGet("userId").(int64)

	// Default to the current month and year if not provided in the query.
	// This will fetch the statement that is currently open or due next.
	now := time.Now()
	month, _ := strconv.Atoi(c.DefaultQuery("month", strconv.Itoa(int(now.Month()))))
	year, _ := strconv.Atoi(c.DefaultQuery("year", strconv.Itoa(now.Year())))

	// Call the service method that contains all the complex logic
	statementDetails, err := h.service.GetStatementDetails(c.Request.Context(), userId, accountId, year, month)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			dto.SendError(c, http.StatusNotFound, err.Error(), nil)
			return
		}
		if strings.Contains(err.Error(), "only valid for credit card") {
			dto.SendError(c, http.StatusBadRequest, err.Error(), nil)
			return
		}
		logger.Error().Err(err).Msg("failed to get statement details")
		dto.SendError(c, http.StatusInternalServerError, "failed to generate statement report", map[string]string{
			"message": err.Error(),
		})
		return
	}

	// Map the internal service struct to the public API DTOs.
	// This is the "translation" step.
	transactions := []dto.TransactionResponse{}
	for _, tx := range statementDetails.Transactions {
		transactions = append(transactions, dto.TransactionResponse{
			Id:                   tx.Id,
			Description:          tx.Description,
			Amount:               tx.Amount,
			Date:                 tx.Date,
			Type:                 tx.Type,
			AccountId:            tx.AccountId,
			AccountName:          tx.AccountName,
			CategoryId:           tx.CategoryId,
			CategoryName:         tx.CategoryName,
			DestinationAccountId: tx.DestinationAccountId,
			CreatedAt:            tx.CreatedAt,
		})
	}

	response := dto.StatementResponse{
		AccountName:    statementDetails.AccountName,
		StatementTotal: statementDetails.StatementTotal,
		PaymentDueDate: statementDetails.PaymentDueDate,
		Period: dto.StatementPeriod{
			Start: statementDetails.StatementPeriod.Start,
			End:   statementDetails.StatementPeriod.End,
		},
		Transactions: transactions,
	}

	dto.SendSuccessResponse(c, http.StatusOK, response)
}
