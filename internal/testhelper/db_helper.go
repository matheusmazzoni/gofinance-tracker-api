package testhelper

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/jmoiron/sqlx"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/db"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// GetTestDB é o ponto de entrada para qualquer teste que precise de um banco de dados.
// Ele garante que o container seja iniciado apenas uma vez e retorna a conexão.
func SetupTestDB() (*sqlx.DB, testcontainers.Container) {
	ctx := context.Background()
	logger := zerolog.Nop()

	waitStrategy := wait.ForSQL("5432/tcp", "postgres", func(host string, port nat.Port) string {
		return fmt.Sprintf("postgres://user:password@%s:%s/test-db?sslmode=disable", host, port.Port())
	}).WithStartupTimeout(20 * time.Second)

	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("test-db"),
		postgres.WithUsername("user"),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(waitStrategy),
	)

	if err != nil {
		log.Fatalf("failed to start postgres container: %v", err)
	}
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("failed to get connection string: %v", err)
	}

	testDB, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		log.Fatalf("failed to connect to test database: %v", err)
	}

	migrationsPath, err := getMigrationsPath()
	if err != nil {
		log.Fatalf("failed to find migrations path: %v", err)
	}

	err = db.RunMigrations(connStr, migrationsPath, &logger)
	if err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	return testDB, pgContainer
}

// TruncateTables é um helper compartilhado para limpar o estado entre os testes.
func TruncateTableAccount(t *testing.T, db *sqlx.DB) {
	_, err := db.Exec("TRUNCATE TABLE transaction_tags, budgets, transactions, accounts, categories, tags, users RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}
func TruncateTables(t *testing.T, db *sqlx.DB) {
	_, err := db.Exec("TRUNCATE TABLE transaction_tags, budgets, transactions, accounts, categories, tags, users RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

// getMigrationsPath é um helper interno para encontrar o caminho das migrations.
func getMigrationsPath() (string, error) {
	_, b, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(b), "../..")
	return filepath.Join(projectRoot, "db/migrations"), nil
}
