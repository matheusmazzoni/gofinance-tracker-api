package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/api/dto"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/service"
)

type AccountHandler struct {
	service *service.AccountService
}

func NewAccountHandler(s *service.AccountService) *AccountHandler {
	return &AccountHandler{service: s}
}

// CreateAccount godoc
//
//	@Summary		Cria uma nova conta
//	@Description	Adiciona uma nova conta financeira ao sistema do usuário logado
//	@Tags			accounts
//	@Accept			json
//	@Produce		json
//	@Param			account	body		dto.CreateAccountRequest	true	"Dados da Conta para Criação"
//	@Success		201		{object}	dto.CreateAccountResponse
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		500		{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/accounts [post]
func (h *AccountHandler) CreateAccount(c *gin.Context) {
	var req dto.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.SendErrorResponse(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	userId := c.MustGet("userId").(int64)
	account := model.Account{
		UserId:         userId,
		Name:           req.Name,
		Type:           req.Type,
		InitialBalance: req.InitialBalance,
	}

	id, err := h.service.CreateAccount(c.Request.Context(), account)
	if err != nil {
		// Verifica se o erro é de chave duplicada (nome da conta)
		if strings.Contains(err.Error(), "unique constraint") {
			dto.SendErrorResponse(c, http.StatusBadRequest, "an account with this name already exists")
			return
		}
		dto.SendErrorResponse(c, http.StatusInternalServerError, "failed to create account")
		return
	}

	response := dto.CreateAccountResponse{Id: id}
	dto.SendSuccessResponse(c, http.StatusCreated, response)
}

// ListAccounts godoc
//
//	@Summary		Lista todas as contas do usuário
//	@Description	Retorna um array com todas as contas do usuário logado
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
		dto.SendErrorResponse(c, http.StatusInternalServerError, "failed to list accounts")
		return
	}

	var responses []dto.AccountResponse
	for _, acc := range accounts {
		responses = append(responses, dto.AccountResponse{
			Id:                  acc.Id,
			Name:                acc.Name,
			Type:                acc.Type,
			Balance:             acc.Balance,
			CreatedAt:           acc.CreatedAt,
			UpdatedAt:           acc.UpdatedAt,
			StatementClosingDay: acc.StatementClosingDay,
			PaymentDueDay:       acc.PaymentDueDay,
		})
	}
	dto.SendSuccessResponse(c, http.StatusOK, responses)
}

// GetAccount godoc
//
//	@Summary		Busca uma conta pelo ID
//	@Description	Retorna os detalhes de uma única conta que pertença ao usuário logado
//	@Tags			accounts
//	@Produce		json
//	@Param			id	path		int	true	"Id da Conta"
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
			dto.SendErrorResponse(c, http.StatusNotFound, "account not found")
			return
		}
		dto.SendErrorResponse(c, http.StatusInternalServerError, "failed to get account")
		return
	}

	dto.SendSuccessResponse(c, http.StatusOK, dto.AccountResponse{
		Id:                  account.Id,
		Name:                account.Name,
		Type:                account.Type,
		InitialBalance:      account.InitialBalance,
		Balance:             account.Balance,
		CreatedAt:           account.CreatedAt,
		UpdatedAt:           account.UpdatedAt,
		StatementClosingDay: account.StatementClosingDay,
		PaymentDueDay:       account.PaymentDueDay,
	})
}

// UpdateAccount godoc
//
//	@Summary		Atualiza uma conta
//	@Description	Atualiza os detalhes de uma conta existente
//	@Tags			accounts
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int							true	"Id da Conta"
//	@Param			account	body		dto.UpdateAccountRequest	true	"Dados para Atualizar"
//	@Success		200		{object}	dto.AccountResponse
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		404		{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/accounts/{id} [put]
func (h *AccountHandler) UpdateAccount(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userId := c.MustGet("userId").(int64)

	var req dto.UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.SendErrorResponse(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	account := model.Account{Name: req.Name, Type: req.Type}
	updatedAcc, err := h.service.UpdateAccount(c.Request.Context(), id, userId, account)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			dto.SendErrorResponse(c, http.StatusNotFound, "account not found")
			return
		}
		dto.SendErrorResponse(c, http.StatusInternalServerError, "failed to update account")
		return
	}

	dto.SendSuccessResponse(c, http.StatusOK, dto.AccountResponse{
		Id:                  updatedAcc.Id,
		Name:                updatedAcc.Name,
		Type:                updatedAcc.Type,
		InitialBalance:      updatedAcc.InitialBalance,
		Balance:             updatedAcc.Balance,
		CreatedAt:           updatedAcc.CreatedAt,
		UpdatedAt:           updatedAcc.UpdatedAt,
		StatementClosingDay: updatedAcc.StatementClosingDay,
		PaymentDueDay:       updatedAcc.PaymentDueDay,
	})
}

// DeleteAccount godoc
//
//	@Summary		Deleta uma conta
//	@Description	Remove uma conta do sistema. Falhará se a conta tiver transações associadas.
//	@Tags			accounts
//	@Param			id	path	int	true	"Id da Conta"
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
			dto.SendErrorResponse(c, http.StatusNotFound, "account not found")
			return
		}
		// Verifica erro de chave estrangeira, indicando que a conta está em uso.
		if strings.Contains(err.Error(), "violates foreign key constraint") {
			dto.SendErrorResponse(c, http.StatusConflict, "cannot delete account with associated transactions")
			return
		}
		dto.SendErrorResponse(c, http.StatusInternalServerError, "failed to delete account")
		return
	}
	c.Status(http.StatusNoContent)
}
