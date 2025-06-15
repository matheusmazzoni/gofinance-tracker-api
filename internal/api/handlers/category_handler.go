package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/api/dto"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/service"
)

type CategoryHandler struct {
	service *service.CategoryService
}

func NewCategoryHandler(s *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{service: s}
}

// CreateCategory godoc
//
//	@Summary		Cria uma nova categoria
//	@Description	Adiciona uma nova categoria financeira ao sistema
//	@Tags			categories
//	@Accept			json
//	@Produce		json
//	@Param			account	body		dto.CategoryRequest	true	"Dados da Conta"
//	@Success		201		{object}	dto.CategoryResponse
//	@Security		BearerAuth
//	@Router			/categories [post]
func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var req dto.CategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	category := model.Category{
		UserId: c.MustGet("userId").(int64),
		Name:   req.Name,
	}

	id, err := h.service.CreateCategory(c.Request.Context(), category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create account"})
		return
	}

	c.JSON(http.StatusCreated, dto.CategoryResponse{
		Name: req.Name,
		Id:   id,
	})
}

// ListCategorys godoc
//
//	@Summary		Lista todas as categorias do usuário
//	@Description	Retorna um array com todas as categorias do usuário logado
//	@Tags			categories
//	@Produce		json
//	@Success		200	{array}	dto.CategoryResponse
//	@Security		BearerAuth
//	@Router			/categories [get]
func (h *CategoryHandler) ListCategories(c *gin.Context) {
	userId := c.MustGet("userId").(int64)
	categories, err := h.service.ListCategoriesByUserId(c.Request.Context(), userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list categories"})
		return
	}

	var responses []dto.CategoryResponse
	for _, ct := range categories {
		responses = append(responses, dto.CategoryResponse{
			Id:   ct.Id,
			Name: ct.Name,
		})
	}
	c.JSON(http.StatusOK, responses)
}

// GetCategory godoc
//
//	@Summary		Busca uma categoria pelo ID
//	@Description	Retorna os detalhes de uma única categoria
//	@Tags			categories
//	@Produce		json
//	@Param			id	path		int	true	"Id da Conta"
//	@Success		200	{object}	dto.CategoryResponse
//	@Security		BearerAuth
//	@Router			/categories/{id} [get]
func (h *CategoryHandler) GetCategory(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userId := c.MustGet("userId").(int64)
	category, err := h.service.GetCategoryById(c.Request.Context(), id, userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get account"})
		return
	}

	response := dto.CategoryResponse{
		Name: category.Name,
		Id:   id,
	}

	dto.SendSuccessResponse(c, http.StatusOK, response)
}

// UpdateCategory godoc
//
//	@Summary		Atualiza uma categoria
//	@Description	Atualiza os detalhes de uma categoria existente
//	@Tags			categories
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int						true	"Id da Conta"
//	@Param			account	body		dto.CategoryResponse	true	"Dados para Atualizar"
//	@Success		200		{object}	dto.CategoryResponse
//	@Security		BearerAuth
//	@Router			/categories/{id} [put]
func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userId := c.MustGet("userId").(int64)
	var acc model.Category
	if err := c.ShouldBindJSON(&acc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updatedAcc, err := h.service.UpdateCategory(c.Request.Context(), id, userId, acc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update account"})
		return
	}

	response := dto.CategoryResponse{
		Id:   updatedAcc.Id,
		Name: updatedAcc.Name,
	}

	dto.SendSuccessResponse(c, http.StatusOK, response)
}

// DeleteCategory godoc
//
//	@Summary		Deleta uma categoria
//	@Description	Remove uma categoria do sistema
//	@Tags			categories
//	@Param			id	path	int	true	"Id da Conta"
//	@Success		204
//	@Security		BearerAuth
//	@Router			/categories/{id} [delete]
func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userId := c.MustGet("userId").(int64)
	err := h.service.DeleteCategory(c.Request.Context(), id, userId)
	if err != nil {
		// Erro pode ser por 'not found' ou pela constraint 'RESTRICT'
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete account. It may have associated transactions."})
		return
	}
	dto.SendSuccessResponse(c, http.StatusNoContent, nil)
}
