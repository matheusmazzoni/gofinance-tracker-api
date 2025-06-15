package main

import (
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	_ "github.com/matheusmazzoni/gofinance-tracker-api/docs"
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
	logger := logger.New()

	var cfg config.Config
	if err := cfg.Load(logger); err != nil {
		logger.Fatal().Err(err).Msg("could not load config")
	}

	// 3. Conectar ao Banco de Dados
	database, err := sqlx.Connect("postgres", cfg.DatabaseURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("could not connect to the database")
	}
	defer database.Close()

	// 4. Rodar Migrations
	if err := db.RunMigrations(cfg.DatabaseURL, "db/migrations", logger); err != nil {
		logger.Fatal().Err(err).Msg("Failed to run database migrations")
	}

	// 5. Criar e Configurar o Servidor
	srv := server.NewServer(cfg, database, logger)

	// 6. Iniciar o Servidor
	logger.Info().Msgf("Server is running on port %s", cfg.ServerPort)
	logger.Info().Msgf("Swagger UI available at http://%s:%s/swagger/index.html", cfg.ServerHostName, cfg.ServerPort)
	if err := srv.Start(); err != nil {
		logger.Fatal().Err(err).Msg("failed to start server")
	}
}
