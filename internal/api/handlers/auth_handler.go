package handlers

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/api/dto"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/service"
	"github.com/rs/zerolog"
)

type AuthHandler struct {
	service *service.AuthService
}

func NewAuthHandler(s *service.AuthService) *AuthHandler {
	return &AuthHandler{service: s}
}

// Login godoc
//
//	@Summary		Realiza o login do usuário
//	@Description	Autentica o usuário e retorna um token JWT
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			credentials	body		dto.LoginRequest	true	"Credenciais de Login"
//	@Success		200			{object}	dto.LoginResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		400			{object}	dto.ErrorResponse
//	@Router			/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	logger := zerolog.Ctx(c.Request.Context()).With().Str("handler", "AuthHandlerLogin").Logger()

	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	logger.Debug().Str("email", req.Email).Msg("AuthHandler: Received login request")

	token, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		logger.Warn().Err(err)
		if errors.Is(err, sql.ErrNoRows) {
			dto.SendErrorResponse(c, http.StatusNotFound, "user not found")
			return
		}
		dto.SendErrorResponse(c, http.StatusUnauthorized, err.Error())
		return
	}

	logger.Info().Msg("Login successful, returning token")
	dto.SendSuccessResponse(c, http.StatusOK, dto.LoginResponse{Token: token})
}
