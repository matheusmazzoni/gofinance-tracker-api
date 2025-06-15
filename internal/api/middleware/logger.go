package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// LoggerMiddleware é um middleware do Gin que loga cada requisição HTTP.
func LoggerMiddleware(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Gera um ID único para a requisição para facilitar o rastreamento
		requestID := uuid.New().String()
		c.Set("requestID", requestID)

		// Cria um logger filho com o ID da requisição
		ctxlog := logger.With().Str("request_id", requestID).Logger()
		c.Request = c.Request.WithContext(ctxlog.WithContext(c.Request.Context()))

		// Processa a requisição
		c.Next()

		duration := time.Since(start)
		statusCode := c.Writer.Status()

		log := zerolog.Ctx(c.Request.Context())

		// Cria o evento de log com base no status da resposta
		var logEvent *zerolog.Event
		if statusCode >= 500 {
			logEvent = log.Error()
		} else if statusCode >= 400 {
			logEvent = log.Warn()
		} else {
			logEvent = log.Info()
		}

		// Adiciona todos os campos relevantes e envia a mensagem de log
		logEvent.
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status_code", statusCode).
			Dur("duration_ms", duration).
			Str("client_ip", c.ClientIP()).
			Msg("request completed")
	}
}
