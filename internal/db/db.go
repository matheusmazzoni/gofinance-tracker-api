package db

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // DB driver for migrate
	_ "github.com/golang-migrate/migrate/v4/source/file"       // Source driver for migrate
	"github.com/rs/zerolog"
)

// RunMigrations executes pending database migrations from a given path and URL.
// It returns an error if anything fails, allowing the caller to decide how to handle the failure.
// It accepts a logger instance for consistent application logging.
func RunMigrations(databaseURL, migrationsPath string, logger *zerolog.Logger) error {
	if databaseURL == "" {
		return errors.New("database URL for migrations cannot be empty")
	}
	if migrationsPath == "" {
		return errors.New("migrations path cannot be empty")
	}

	logger.Info().Str("path", migrationsPath).Msg("Running database migrations...")

	// The migrate library expects the file source path with the 'file://' prefix
	sourceURL := fmt.Sprintf("file://%s", migrationsPath)

	m, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		return fmt.Errorf("could not create new migrate instance: %w", err)
	}

	// The Up() method applies all pending up migrations.
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		// We return the error to be handled by the caller.
		return fmt.Errorf("could not run up migrations: %w", err)
	}

	logger.Info().Msg("Database migrations applied successfully")
	return nil
}

// getMigrationsPath finds the project root by looking for the go.mod file
// and returns the absolute path to the migrations directory.
func GetMigrationsPath() (string, error) {
	// Get the directory of the currently running test file.
	_, b, _, _ := runtime.Caller(0)
	// => /path/to/your/project/internal/repository/account_repository_test.go

	currentDir := filepath.Dir(b)
	// => /path/to/your/project/internal/repository

	// Traverse up the directory tree until we find go.mod
	// We limit to 5 levels up to prevent infinite loops in weird filesystems.
	for i := 0; i < 5; i++ {
		goModPath := filepath.Join(currentDir, "go.mod")
		// os.Stat returns an error if the file doesn't exist.
		if _, err := os.Stat(goModPath); err == nil {
			// go.mod found, so currentDir is the project root.
			// Now we can build the reliable path to the migrations folder.
			return filepath.Join(currentDir, "db/migrations"), nil
		}
		// Go up one level.
		currentDir = filepath.Join(currentDir, "..")
	}

	return "", fmt.Errorf("could not find project root (go.mod file)")
}
