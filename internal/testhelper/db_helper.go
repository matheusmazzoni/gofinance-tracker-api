package testhelper

import (
	"context"
	"fmt"
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

// SetupTestDB initializes a Postgres database container for testing.
// It ensures the container is started and that migrations are applied.
// It returns the database connection (*sqlx.DB) and the container instance,
// which must be terminated by the caller at the end of the test (e.g., defer container.Terminate(ctx)).
func SetupTestDB() (*sqlx.DB, testcontainers.Container) {
	ctx := context.Background()
	// A Nop logger is used to avoid polluting test output.
	// For debugging migrations, a standard logger can be substituted.
	logger := zerolog.Nop()

	// Wait strategy that waits for the database to accept connections.
	waitStrategy := wait.ForSQL("5432/tcp", "postgres", func(host string, port nat.Port) string {
		return fmt.Sprintf("postgres://user:password@%s:%s/test-db?sslmode=disable", host, port.Port())
	}).WithStartupTimeout(20 * time.Second)

	// Run the Postgres container using the specified image, configuration, and wait strategy.
	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("test-db"),
		postgres.WithUsername("user"),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(waitStrategy),
	)
	if err != nil {
		logger.Err(err).Msg("failed to start postgres container")
	}

	// Get the dynamic connection string from the container.
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		logger.Err(err).Msg("failed to get connection string")
	}

	// Connect to the test database.
	testDB, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		logger.Err(err).Msg("failed to connect to test database")
	}

	// Find the path to the migration files.
	migrationsPath, err := getMigrationsPath()
	if err != nil {
		logger.Err(err).Msg("failed to find migrations path")
	}

	// Run migrations on the newly created database.
	err = db.RunMigrations(connStr, migrationsPath, &logger)
	if err != nil {
		logger.Err(err).Msg("failed to run migrations")
	}

	return testDB, pgContainer
}

// TruncateTables cleans all database tables to ensure tests run in a clean, isolated state.
// The `RESTART IDENTITY` clause resets primary key sequences, and `CASCADE` removes
// records in dependent tables.
func TruncateTables(t *testing.T, db *sqlx.DB) {
	_, err := db.Exec("TRUNCATE TABLE transaction_tags, budgets, transactions, accounts, categories, tags, users RESTART IDENTITY CASCADE")
	// require.NoError ensures the test fails if the database cleanup is unsuccessful.
	require.NoError(t, err)
}

// getMigrationsPath is an internal helper that finds the migrations directory
// by navigating up two levels from the current file's directory.
func getMigrationsPath() (string, error) {
	_, b, _, _ := runtime.Caller(0)
	// Navigate from the current directory (testhelper) to the project root.
	projectRoot := filepath.Join(filepath.Dir(b), "../..")
	return filepath.Join(projectRoot, "db/migrations"), nil
}
