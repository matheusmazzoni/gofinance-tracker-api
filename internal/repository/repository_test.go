package repository

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/testhelper"
	"github.com/testcontainers/testcontainers-go"
)

var testDB *sqlx.DB

func TestMain(m *testing.M) {
	var pgContainer testcontainers.Container
	testDB, pgContainer = testhelper.SetupTestDB()
	exitCode := m.Run()

	if err := pgContainer.Terminate(context.Background()); err != nil {
		log.Fatalf("failed to terminate container: %v", err)
	}

	os.Exit(exitCode)
}
