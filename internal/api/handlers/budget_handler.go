package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/api/dto"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/service"
)

type BudgetHandler struct {
	service *service.BudgetService
}

func NewBudgetHandler(s *service.BudgetService) *BudgetHandler {
	return &BudgetHandler{service: s}
}

// ListBudgets godoc
// @Summary      Lists budgets for a given period
// @Description  Retrieves a list of all budgets for the authenticated user for a specific month and year. If no period is provided, the current month and year are used.
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
	userID := c.MustGet("userId").(int64)

	// Default to current month and year if not provided
	now := time.Now()
	monthStr := c.DefaultQuery("month", strconv.Itoa(int(now.Month())))
	yearStr := c.DefaultQuery("year", strconv.Itoa(now.Year()))

	month, errMonth := strconv.Atoi(monthStr)
	year, errYear := strconv.Atoi(yearStr)

	if errMonth != nil || errYear != nil {
		dto.SendErrorResponse(c, http.StatusBadRequest, "invalid month or year format")
		return
	}

	// The service returns the enriched budget data
	enrichedBudgets, err := h.service.ListEnrichedBudgetsByPeriod(c.Request.Context(), userID, month, year)
	if err != nil {
		dto.SendErrorResponse(c, http.StatusInternalServerError, "failed to list budgets")
		return
	}

	// Map the service layer's enriched struct to the API's DTO
	var responses []dto.BudgetResponse
	for _, eb := range enrichedBudgets {
		responses = append(responses, dto.BudgetResponse{
			ID:           eb.Id,
			CategoryID:   eb.CategoryId,
			CategoryName: eb.CategoryName,
			Amount:       eb.Amount,
			SpentAmount:  eb.SpentAmount,
			Balance:      eb.Balance,
			Month:        eb.Month,
			Year:         eb.Year,
			CreatedAt:    eb.CreatedAt,
		})
	}

	dto.SendSuccessResponse(c, http.StatusOK, responses)
}

// (You would add handlers for Create, Update, Delete here as well)
