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
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/repository"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/service"
)

type TransactionHandler struct {
	service *service.TransactionService
}

func NewTransactionHandler(s *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{service: s}
}

// CreateTransaction godoc
//
//	@Summary		Cria uma nova transação
//	@Description	Adiciona uma nova transação ao sistema. Para transferências, o campo destination_account_id é obrigatório.
//	@Tags			transactions
//	@Accept			json
//	@Produce		json
//	@Param			transaction	body		dto.CreateTransactionRequest	true	"Dados da Transação para Criar"
//	@Success		201			{object}	dto.TransactionResponse
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/transactions [post]
func (h *TransactionHandler) CreateTransaction(c *gin.Context) {
	var req dto.CreateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	userId := c.MustGet("userId").(int64)

	tx := model.Transaction{
		UserId:               userId,
		Description:          req.Description,
		Amount:               req.Amount,
		Date:                 req.Date,
		Type:                 req.Type,
		AccountId:            req.AccountId,
		CategoryId:           req.CategoryId,
		DestinationAccountId: req.DestinationAccountId,
	}

	id, err := h.service.CreateTransaction(c.Request.Context(), tx)
	if err != nil {
		// O serviço agora retorna erros de negócio específicos
		dto.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	dto.SendSuccessResponse(c, http.StatusCreated, dto.TransactionResponse{Id: id})
}

// ListTransactions godoc
//	@Summary		Lists and filters user transactions
//	@Description	Retrieves a list of transactions for the authenticated user, with optional filters.
//	@Tags			transactions
//	@Produce		json
//	@Param			description		query	string	false	"Search text in description (case-insensitive)"
//	@Param			type			query	string	false	"Filter by type (income, expense, transfer)"	Enums(income, expense, transfer)
//	@Param			account_id		query	int		false	"Filter by a specific account Id"
//	@Param			start_date		query	string	false	"Filter by start date (format: YYYY-MM-DD)"
//	@Param			end_date		query	string	false	"Filter by end date (format: YYYY-MM-DD)"
//	@Param			category_ids	query	string	false	"Filter by one or more category Ids (comma-separated, e.g., 1,5,8)"
//	@Success		200				{array}	dto.TransactionResponse
//	@Security		BearerAuth
//	@Router			/transactions [get]
func (h *TransactionHandler) ListTransactions(c *gin.Context) {
	userId := c.MustGet("userId").(int64)

	// Create the filters struct by parsing query parameters
	var filters repository.ListTransactionFilters

	if description := c.Query("description"); description != "" {
		filters.Description = &description
	}
	if txTypeStr := c.Query("type"); txTypeStr != "" {
		txType := model.TransactionType(txTypeStr)
		filters.Type = &txType
	}
	if accountIdStr := c.Query("account_id"); accountIdStr != "" {
		if accountId, err := strconv.ParseInt(accountIdStr, 10, 64); err == nil {
			filters.AccountId = &accountId
		}
	}
	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if startDate, err := time.Parse("2006-01-02", startDateStr); err == nil {
			filters.StartDate = &startDate
		}
	}
	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if endDate, err := time.Parse("2006-01-02", endDateStr); err == nil {
			// To include the whole day, we set the time to the end of the day.
			endOfDay := endDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			filters.EndDate = &endOfDay
		}
	}
	if categoryIdsStr := c.Query("category_ids"); categoryIdsStr != "" {
		idsStr := strings.Split(categoryIdsStr, ",")
		ids := make([]int64, 0, len(idsStr))
		for _, idStr := range idsStr {
			if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
				ids = append(ids, id)
			}
		}
		if len(ids) > 0 {
			filters.CategoryIds = ids
		}
	}

	transactions, err := h.service.ListTransactions(c.Request.Context(), userId, filters)
	if err != nil {
		dto.SendErrorResponse(c, http.StatusInternalServerError, "failed to list transactions")
		return
	}

	var responses []dto.TransactionResponse
	for _, tx := range transactions {
		responses = append(responses, dto.TransactionResponse{
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

	dto.SendSuccessResponse(c, http.StatusOK, responses)
}

// GetTransaction godoc
//
//	@Summary		Busca uma transação pelo Id
//	@Description	Retorna os detalhes de uma única transação
//	@Tags			transactions
//	@Produce		json
//	@Param			id	path		int	true	"Id da Transação"
//	@Success		200	{object}	dto.TransactionResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/transactions/{id} [get]
func (h *TransactionHandler) GetTransaction(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userId := c.MustGet("userId").(int64)

	tx, err := h.service.GetTransactionById(c.Request.Context(), id, userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			dto.SendErrorResponse(c, http.StatusNotFound, "transaction not found")
			return
		}
		dto.SendErrorResponse(c, http.StatusInternalServerError, "failed to get transaction")
		return
	}

	response := dto.TransactionResponse{
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
	}
	dto.SendSuccessResponse(c, http.StatusOK, response)
}

// UpdateTransaction godoc
//
//	@Summary		Atualiza uma transação existente
//	@Description	Atualiza os detalhes de uma transação com base no seu Id
//	@Tags			transactions
//	@Accept			json
//	@Produce		json
//	@Param			id			path		int								true	"Id da Transação"
//	@Param			transaction	body		dto.UpdateTransactionRequest	true	"Dados para Atualizar"
//	@Success		200			{object}	dto.TransactionResponse
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/transactions/{id} [put]
func (h *TransactionHandler) UpdateTransaction(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userId := c.MustGet("userId").(int64)

	var req dto.UpdateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	tx := model.Transaction{
		Id:          id,
		UserId:      userId,
		Description: req.Description,
		Amount:      req.Amount,
		Date:        req.Date,
		Type:        req.Type,
		AccountId:   req.AccountId,
		CategoryId:  req.CategoryId,
	}

	updatedTx, err := h.service.UpdateTransaction(c.Request.Context(), tx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			dto.SendErrorResponse(c, http.StatusNotFound, "transaction not found")
			return
		}
		dto.SendErrorResponse(c, http.StatusInternalServerError, "failed to update transaction")
		return
	}

	response := dto.TransactionResponse{
		Id:                   updatedTx.Id,
		Description:          updatedTx.Description,
		Amount:               updatedTx.Amount,
		Date:                 updatedTx.Date,
		Type:                 updatedTx.Type,
		AccountId:            updatedTx.AccountId,
		AccountName:          updatedTx.AccountName,
		CategoryId:           updatedTx.CategoryId,
		CategoryName:         updatedTx.CategoryName,
		DestinationAccountId: updatedTx.DestinationAccountId,
		CreatedAt:            updatedTx.CreatedAt,
	}

	dto.SendSuccessResponse(c, http.StatusOK, response)
}

// Em: internal/api/handlers/transaction_handler.go

// PatchTransaction godoc
//	@Summary		Atualiza uma transação parcialmente
//	@Description	Atualiza um ou mais campos de uma transação. Apenas os campos fornecidos no corpo serão alterados.
//	@Tags			transactions
//	@Accept			json
//	@Produce		json
//	@Param			id			path		int							true	"Transaction Id"
//	@Param			transaction	body		dto.PatchTransactionRequest	true	"Fields to Update"
//	@Success		200			{object}	dto.TransactionResponse
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/transactions/{id} [patch]
func (h *TransactionHandler) PatchTransaction(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userId := c.MustGet("userId").(int64)

	var req dto.PatchTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.SendErrorResponse(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	updatedTx, err := h.service.PatchTransaction(c.Request.Context(), id, userId, req)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			dto.SendErrorResponse(c, http.StatusNotFound, "transaction not found")
			return
		}
		dto.SendErrorResponse(c, http.StatusInternalServerError, "failed to update transaction")
		return
	}

	// Mapeia o model para o DTO de resposta e envia
	dto.SendSuccessResponse(c, http.StatusOK, dto.TransactionResponse{
		Id:                   updatedTx.Id,
		Description:          updatedTx.Description,
		Amount:               updatedTx.Amount,
		Date:                 updatedTx.Date,
		Type:                 updatedTx.Type,
		AccountId:            updatedTx.AccountId,
		AccountName:          updatedTx.AccountName,
		CategoryId:           updatedTx.CategoryId,
		CategoryName:         updatedTx.CategoryName,
		DestinationAccountId: updatedTx.DestinationAccountId,
		CreatedAt:            updatedTx.CreatedAt,
	})
}

// DeleteTransaction godoc
//
//	@Summary		Deleta uma transação
//	@Description	Remove uma transação do sistema pelo seu Id
//	@Tags			transactions
//	@Param			id	path	int	true	"Id da Transação"
//	@Success		204
//	@Failure		404	{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/transactions/{id} [delete]
func (h *TransactionHandler) DeleteTransaction(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userId := c.MustGet("userId").(int64)

	err := h.service.DeleteTransaction(c.Request.Context(), id, userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			dto.SendErrorResponse(c, http.StatusNotFound, "transaction not found")
			return
		}
		dto.SendErrorResponse(c, http.StatusInternalServerError, "failed to delete transaction")
		return
	}

	dto.SendSuccessResponse(c, http.StatusNoContent, nil)
}
