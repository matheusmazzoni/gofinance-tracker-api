package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	_ "github.com/matheusmazzoni/gofinance-tracker-api/api"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/config"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/db"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/logger"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/server"
)

// @title						API do Sistema Financeiro
// @version					1.0
// @description				Esta é uma API para gerenciamento de finanças pessoais.
// @host						localhost:8080
// @BasePath					/v1
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
func main() {
	// 1. Initialize Logger
	logger := logger.New()

	// 2. Load Configuration
	var cfg config.Config
	if err := cfg.Load(logger); err != nil {
		logger.Fatal().Err(err).Msg("could not load config")
	}

	// 3. Connect to Database
	database, err := sqlx.Connect("postgres", cfg.DatabaseURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("could not connect to the database")
	}
	defer func() {
		if err := database.Close(); err != nil {
			logger.Fatal().Err(err).Msg("Error closing database connection")
		}
	}()

	// 4. Run Migrations
	if err := db.RunMigrations(cfg.DatabaseURL, "db/migrations", logger); err != nil {
		logger.Fatal().Err(err).Msg("Failed to run database migrations")
	}

	// 5. Create and Configure the Server
	srv := server.NewServer(cfg, database, logger)

	// 6. Start Server in a Goroutine
	go func() {
		logger.Info().Msgf("Server is running on port %s", cfg.ServerPort)
		logger.Info().Msgf("Swagger UI available at http://%s:%s/swagger/index.html", cfg.ServerHostName, cfg.ServerPort)
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("failed to start server")
		}
	}()

	// 7. Wait for an interrupt signal (e.g., Ctrl+C)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // This blocks until a signal is received

	logger.Warn().Msg("Shutting down server...")

	// 8. Gracefully shut down the server with a 5-second timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Server forced to shutdown:")
	}

	logger.Info().Msg("Server exiting.")
}
