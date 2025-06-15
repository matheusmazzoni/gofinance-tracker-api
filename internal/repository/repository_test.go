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

	// Usa o helper para o setup
	testDB, pgContainer = testhelper.SetupTestDB()

	// Roda todos os testes do pacote
	exitCode := m.Run()

	// Garante que o container será encerrado após todos os testes do pacote rodarem
	if err := pgContainer.Terminate(context.Background()); err != nil {
		log.Fatalf("failed to terminate container: %v", err)
	}

	os.Exit(exitCode)
}
