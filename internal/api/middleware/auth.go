package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/api/dto"
	"github.com/rs/zerolog"
)

func AuthMiddleware(jwtKey string, logger zerolog.Logger) gin.HandlerFunc {
	key := []byte(jwtKey)
	logger = logger.With().Str("middleware", "AuthMiddleware").Logger()

	return func(c *gin.Context) {

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.Warn().Msg("Authorization header is missing")
			dto.SendErrorResponse(c, http.StatusUnauthorized, "authorization header required")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			logger.Warn().Str("header", authHeader).Msg("Bearer token is missing or malformed")
			dto.SendErrorResponse(c, http.StatusUnauthorized, "bearer token required")
			return
		}

		claims := &jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return key, nil
		})

		if err != nil || !token.Valid {
			logger.Warn().Err(err).Msg("Token is invalid")
			dto.SendErrorResponse(c, http.StatusUnauthorized, "invalid token")
			return
		}

		userId, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil {
			logger.Error().Err(err).Msg("Could not parse userID from token subject")
			dto.SendErrorResponse(c, http.StatusUnauthorized, "invalid user ID in token")
			return
		}

		logger.Debug().Int64("userId", userId).Msg("Token is valid. Setting userId in context.")
		c.Set("userId", userId)
		c.Next()
	}
}
