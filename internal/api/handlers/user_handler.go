package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/api/dto"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/service"
)

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(s *service.UserService) *UserHandler {
	return &UserHandler{service: s}
}

// CreateUser godoc
//
//	@Summary		Registra um novo usuário
//	@Description	Cria um novo usuário no sistema.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			user	body		dto.CreateUserRequest	true	"Dados do Usuário para Registro"
//	@Success		201		{object}	dto.UserResponse
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		500		{object}	dto.ErrorResponse
//	@Router			/users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.SendErrorResponse(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	user := model.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	}

	id, err := h.service.CreateUser(c.Request.Context(), user)
	if err != nil {
		// Verifica se o erro é de chave duplicada no e-mail
		if strings.Contains(err.Error(), "unique constraint") {
			dto.SendErrorResponse(c, http.StatusBadRequest, "a user with this email already exists")
			return
		}
		dto.SendErrorResponse(c, http.StatusInternalServerError, "failed to create user")
		return
	}

	response := dto.UserResponse{
		Id:        id,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}

	dto.SendSuccessResponse(c, http.StatusCreated, response)
}

// GetProfile godoc
//
//	@Summary		Busca o perfil do usuário logado
//	@Description	Retorna os dados do usuário que está fazendo a requisição.
//	@Tags			users
//	@Produce		json
//	@Success		200	{object}	dto.UserResponse
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/users/me [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userId, ok := c.Get("userId")
	if !ok {
		dto.SendErrorResponse(c, http.StatusUnauthorized, "invalid user context")
		return
	}

	user, err := h.service.GetUserById(c.Request.Context(), userId.(int64))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			dto.SendErrorResponse(c, http.StatusNotFound, "user not found")
			return
		}
		dto.SendErrorResponse(c, http.StatusInternalServerError, "failed to retrieve user profile")
		return
	}

	// Mapeia o modelo de domínio para o DTO de resposta
	response := dto.UserResponse{
		Id:        user.Id,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}

	dto.SendSuccessResponse(c, http.StatusOK, response)
}
