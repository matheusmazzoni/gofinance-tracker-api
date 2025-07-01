package repository

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
)

type AccountRepository interface {
	Create(ctx context.Context, acc model.Account) (int64, error)
	GetById(ctx context.Context, id, userId int64) (*model.Account, error)
	GetByName(ctx context.Context, name string, userId int64) (*model.Account, error)
	ListByUserId(ctx context.Context, userId int64) ([]model.Account, error)
	Update(ctx context.Context, acc model.Account) error
	Delete(ctx context.Context, id, userId int64) error
	GetCurrentBalance(ctx context.Context, accountID int64, userId int64) (decimal.Decimal, error)
}

type pqAccountRepository struct {
	db *sqlx.DB
}

func NewAccountRepository(db *sqlx.DB) AccountRepository {
	return &pqAccountRepository{db: db}
}

func (r *pqAccountRepository) Create(ctx context.Context, acc model.Account) (int64, error) {
	query := `
		INSERT INTO accounts (user_id, name, type, initial_balance, statement_closing_day, payment_due_day) 
		VALUES (:user_id, :name, :type, :initial_balance, :statement_closing_day, :payment_due_day) 
		RETURNING id
	`

	rows, err := r.db.NamedQueryContext(ctx, query, acc)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("Error closing rows")
		}
	}()

	var id int64
	if rows.Next() {
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
	}
	return id, nil
}

func (r *pqAccountRepository) GetById(ctx context.Context, id, userId int64) (*model.Account, error) {
	var acc model.Account
	query := `SELECT * FROM accounts WHERE id = $1 AND user_id = $2`
	err := r.db.GetContext(ctx, &acc, query, id, userId)
	return &acc, err
}

func (r *pqAccountRepository) GetByName(ctx context.Context, name string, userId int64) (*model.Account, error) {
	var acc model.Account
	// The query is simple and secure, filtering by both name and user ID.
	// The UNIQUE constraint on (user_id, name) guarantees we get at most one row.
	query := `SELECT * FROM accounts WHERE name = $1 AND user_id = $2`

	// GetContext is perfect here as we expect exactly one result.
	// It will correctly return sql.ErrNoRows if the account is not found.
	err := r.db.GetContext(ctx, &acc, query, name, userId)
	return &acc, err
}

func (r *pqAccountRepository) ListByUserId(ctx context.Context, userId int64) ([]model.Account, error) {
	var accounts []model.Account
	query := `SELECT * FROM accounts WHERE user_id = $1 ORDER BY name`
	err := r.db.SelectContext(ctx, &accounts, query, userId)
	return accounts, err
}

func (r *pqAccountRepository) Update(ctx context.Context, acc model.Account) error {
	query := `
		UPDATE accounts 
		SET 
			name = :name, 
			type = :type, 
			initial_balance = :initial_balance,
			statement_closing_day = :statement_closing_day,
			payment_due_day = :payment_due_day,
			updated_at = NOW() 
		WHERE 
			id = :id AND user_id = :user_id
	`
	result, err := r.db.NamedExecContext(ctx, query, acc)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *pqAccountRepository) Delete(ctx context.Context, id, userId int64) error {
	// ATENÇÃO: A constraint 'ON DELETE RESTRICT' na tabela 'transactions'
	// impedirá a exclusão de uma conta que tenha transações.
	query := `DELETE FROM accounts WHERE id = $1 AND user_id = $2`
	result, err := r.db.ExecContext(ctx, query, id, userId)
	if err != nil {
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *pqAccountRepository) GetCurrentBalance(ctx context.Context, accountID int64, userId int64) (decimal.Decimal, error) {
	var balance decimal.Decimal

	query := `
		WITH movements AS (
			-- Une todas as movimentações de CRÉDITO para esta conta
			SELECT amount FROM transactions WHERE destination_account_id = $1 AND type = 'transfer' AND user_id = $2
			UNION ALL
			SELECT amount FROM transactions WHERE account_id = $1 AND type = 'income' AND user_id = $2
			UNION ALL
			-- Une todas as movimentações de DÉBITO para esta conta (com valor negativo)
			SELECT -amount FROM transactions WHERE account_id = $1 AND type IN ('expense', 'transfer') AND user_id = $2
		)
		-- A query final seleciona o saldo inicial e soma com o total de todas as movimentações.
		SELECT
			a.initial_balance + (SELECT COALESCE(SUM(amount), 0) FROM movements)
		FROM
			accounts a
		WHERE
			a.id = $1 AND a.user_id = $2;
	`
	// O GetContext executará a query e colocará o resultado único na variável 'balance'.
	err := r.db.GetContext(ctx, &balance, query, accountID, userId)
	return balance, err
}
