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
)

type BudgetHandler struct {
	service *service.BudgetService
}

func NewBudgetHandler(s *service.BudgetService) *BudgetHandler {
	return &BudgetHandler{service: s}
}

// CreateBudget godoc
// @Summary      Creates a new budget
// @Description  Adds a new monthly budget for a specific category.
// @Tags         budgets
// @Accept       json
// @Produce      json
// @Param        budget body dto.CreateBudgetRequest true "Budget Creation Data"
// @Success      201 {object} dto.BudgetResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      409 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /budgets [post]
func (h *BudgetHandler) CreateBudget(c *gin.Context) {
	var req dto.CreateBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.SendErrorResponse(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	userId := c.MustGet("userId").(int64)
	budget := model.Budget{
		UserId:     userId,
		CategoryId: req.CategoryId,
		Amount:     req.Amount,
		Month:      req.Month,
		Year:       req.Year,
	}

	newBudget, err := h.service.CreateBudget(c.Request.Context(), budget)
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			dto.SendErrorResponse(c, http.StatusConflict, "a budget for this category and period already exists")
			return
		}
		dto.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// For the response, we can immediately return the enriched version
	enrichedBudget, err := h.service.GetEnrichedBudgetById(c.Request.Context(), newBudget.Id, userId)
	if err != nil {
		dto.SendErrorResponse(c, http.StatusInternalServerError, "failed to retrieve newly created budget details")
		return
	}

	dto.SendSuccessResponse(c, http.StatusCreated, toBudgetResponseDTO(enrichedBudget))
}

// ListBudgets godoc
// @Summary      Lists budgets for a given period
// @Description  Retrieves all budgets for the user for a specific month and year. Defaults to the current month/year.
// @Tags         budgets
// @Produce      json
// @Param        month query int false "Month to filter (1-12)"
// @Param        year  query int false "Year to filter (e.g., 2025)"
// @Success      200 {array} dto.BudgetResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /budgets [get]
func (h *BudgetHandler) ListBudgets(c *gin.Context) {
	userId := c.MustGet("userId").(int64)

	now := time.Now()
	month, errMonth := strconv.Atoi(c.DefaultQuery("month", strconv.Itoa(int(now.Month()))))
	year, errYear := strconv.Atoi(c.DefaultQuery("year", strconv.Itoa(now.Year())))

	if errMonth != nil || errYear != nil {
		dto.SendErrorResponse(c, http.StatusBadRequest, "invalid month or year format")
		return
	}
	enrichedBudgets, err := h.service.ListEnrichedBudgetsByPeriod(c.Request.Context(), userId, month, year)
	if err != nil {
		dto.SendErrorResponse(c, http.StatusInternalServerError, "failed to list budgets")
		return
	}

	var responses []dto.BudgetResponse
	for _, eb := range enrichedBudgets {
		responses = append(responses, toBudgetResponseDTO(&eb))
	}

	dto.SendSuccessResponse(c, http.StatusOK, responses)
}

// UpdateBudget godoc
// @Summary      Updates a budget's amount
// @Description  Changes the amount for an existing budget.
// @Tags         budgets
// @Accept       json
// @Produce      json
// @Param        id     path int true "Budget Id"
// @Param        budget body dto.UpdateBudgetRequest true "New Budget Amount"
// @Success      200 {object} dto.BudgetResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /budgets/{id} [put]
func (h *BudgetHandler) UpdateBudget(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.SendErrorResponse(c, http.StatusBadRequest, "invalid budget Id format")
		return
	}
	userId := c.MustGet("userId").(int64)

	var req dto.UpdateBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.SendErrorResponse(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	budget := model.Budget{Amount: req.Amount}
	updatedBudget, err := h.service.UpdateBudget(c.Request.Context(), id, userId, budget)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			dto.SendErrorResponse(c, http.StatusNotFound, "budget not found")
			return
		}
		dto.SendErrorResponse(c, http.StatusInternalServerError, "failed to update budget")
		return
	}

	enrichedBudget, err := h.service.GetEnrichedBudgetById(c.Request.Context(), updatedBudget.Id, userId)
	if err != nil {
		dto.SendErrorResponse(c, http.StatusInternalServerError, "failed to retrieve updated budget details")
		return
	}

	dto.SendSuccessResponse(c, http.StatusOK, toBudgetResponseDTO(enrichedBudget))
}

// DeleteBudget godoc
// @Summary      Deletes a budget
// @Description  Removes a budget for a specific category and period.
// @Tags         budgets
// @Param        id path int true "Budget Id"
// @Success      204
// @Failure      404 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /budgets/{id} [delete]
func (h *BudgetHandler) DeleteBudget(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.SendErrorResponse(c, http.StatusBadRequest, "invalid budget Id format")
		return
	}
	userId := c.MustGet("userId").(int64)

	if err := h.service.DeleteBudget(c.Request.Context(), id, userId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			dto.SendErrorResponse(c, http.StatusNotFound, "budget not found")
			return
		}
		dto.SendErrorResponse(c, http.StatusInternalServerError, "failed to delete budget")
		return
	}

	c.Status(http.StatusNoContent)
}

// toBudgetResponseDTO is a helper function to map the internal enriched struct to the public DTO.
func toBudgetResponseDTO(eb *service.EnrichedBudget) dto.BudgetResponse {
	return dto.BudgetResponse{
		Id:           eb.Id,
		CategoryId:   eb.CategoryId,
		CategoryName: eb.CategoryName,
		Amount:       eb.Amount,
		SpentAmount:  eb.SpentAmount,
		Balance:      eb.Balance,
		Month:        eb.Month,
		Year:         eb.Year,
		CreatedAt:    eb.CreatedAt,
	}
}
