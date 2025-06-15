package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

func New() *zerolog.Logger {
	// Use uma variável de ambiente para controlar o modo (ex: APP_ENV=development)
	env := os.Getenv("APP_ENV")

	var logger zerolog.Logger

	if env == "development" {
		// Logger com formato legível e colorido para desenvolvimento
		output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
		logger = zerolog.New(output).With().Timestamp().Logger()
	} else {
		// Logger com formato JSON para produção
		logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	}

	// Define o nível de log global (pode vir de uma config também)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if env == "development" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	return &logger
}
