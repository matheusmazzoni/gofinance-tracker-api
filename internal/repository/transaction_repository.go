package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/shopspring/decimal"
)

// TransactionRepository define a interface para o acesso de dados de transações.
// Usar uma interface aqui é uma boa prática para permitir testes e mocks.
type TransactionRepository interface {
	Create(ctx context.Context, tx model.Transaction) (int64, error)
	GetById(ctx context.Context, id, userId int64) (*model.Transaction, error)
	Update(ctx context.Context, tx model.Transaction) error
	Delete(ctx context.Context, id int64, userId int64) error
	List(ctx context.Context, userId int64, filters ListTransactionFilters) ([]model.Transaction, error)
	DeleteByAccountId(ctx context.Context, userId, accountId int64) error
	SumExpensesByCategoryAndPeriod(ctx context.Context, userID, categoryID int64, startDate, endDate time.Time) (decimal.Decimal, error)
}

// ListTransactionFilters holds all possible optional filters for listing transactions.
// We use pointers so we can differentiate between a zero value and a filter that wasn't provided.
type ListTransactionFilters struct {
	Description *string
	StartDate   *time.Time
	EndDate     *time.Time
	Type        *model.TransactionType
	AccountId   *int64
	CategoryIds []int64 // A slice to allow filtering by multiple categories
}

// pqTransactionRepository é a implementação em PostgreSQL do TransactionRepository.
type pqTransactionRepository struct {
	db *sqlx.DB
}

// NewTransactionRepository cria uma nova instância do repositório.
func NewTransactionRepository(db *sqlx.DB) TransactionRepository {
	return &pqTransactionRepository{db: db}
}

// Create insere uma nova transação no banco de dados.
func (r *pqTransactionRepository) Create(ctx context.Context, tx model.Transaction) (int64, error) {
	query := `
		INSERT INTO transactions (user_id, description, amount, date, type, account_id, destination_account_id, category_id)
		VALUES (:user_id, :description, :amount, :date, :type, :account_id, :destination_account_id, :category_id)
		RETURNING id
	`
	// Usamos NamedExec para que o sqlx mapeie os campos da struct para os parâmetros da query.
	rows, err := r.db.NamedQueryContext(ctx, query, tx)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var id int64
	if rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			return 0, err
		}
	}
	return id, nil
}

// GetById busca uma transação por seu Id.
func (r *pqTransactionRepository) GetById(ctx context.Context, id, userId int64) (*model.Transaction, error) {
	var tx model.Transaction
	query := `
		SELECT 
			t.*, 
			a.name as account_name,
			c.name as category_name
		FROM transactions t
		JOIN accounts a ON t.account_id = a.id
		LEFT JOIN categories c ON t.category_id = c.id
		WHERE t.id = $1 AND t.user_id = $2
	`
	err := r.db.GetContext(ctx, &tx, query, id, userId)
	return &tx, err
}

// Update atualiza uma transação existente no banco de dados.
func (r *pqTransactionRepository) Update(ctx context.Context, tx model.Transaction) error {
	query := `
		UPDATE transactions
		SET
			description = :description,
			amount = :amount,
			date = :date,
			type = :type,
			account_id = :account_id,
			category_id = :category_id,
			destination_account_id = :destination_account_id,
			updated_at = NOW()
		WHERE id = :id AND user_id = :user_id
	`
	result, err := r.db.NamedExecContext(ctx, query, tx)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows // Usa um erro padrão para "não encontrado"
	}

	return nil
}

// Delete remove uma transação do banco de dados pelo seu Id.
func (r *pqTransactionRepository) Delete(ctx context.Context, id int64, userId int64) error {
	query := `DELETE FROM transactions WHERE id = $1 AND user_id = $2`
	result, err := r.db.ExecContext(ctx, query, id, userId)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// List busca todas as transações de um usuário específico.
func (r *pqTransactionRepository) List(ctx context.Context, userId int64, filters ListTransactionFilters) ([]model.Transaction, error) {
	// Use squirrel's builder for PostgreSQL ($1, $2 placeholders)
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	// Start building the query
	queryBuilder := psql.Select(
		"t.*",
		"a.name as account_name",
		"c.name as category_name",
	).
		From("transactions t").
		Join("accounts a ON t.account_id = a.id").
		LeftJoin("categories c ON t.category_id = c.id").
		Where(squirrel.Eq{"t.user_id": userId}). // Mandatory filter for security
		OrderBy("t.date DESC, t.created_at DESC")

	// Apply optional filters dynamically
	if filters.Description != nil && *filters.Description != "" {
		// Using ILIKE for case-insensitive search in PostgreSQL
		queryBuilder = queryBuilder.Where(squirrel.ILike{"t.description": "%" + *filters.Description + "%"})
	}
	if filters.Type != nil {
		queryBuilder = queryBuilder.Where(squirrel.Eq{"t.type": *filters.Type})
	}
	if filters.AccountId != nil {
		queryBuilder = queryBuilder.Where(squirrel.Eq{"t.account_id": *filters.AccountId})
	}
	if filters.StartDate != nil {
		queryBuilder = queryBuilder.Where(squirrel.GtOrEq{"t.date": *filters.StartDate}) // GtOrEq means >=
	}
	if filters.EndDate != nil {
		queryBuilder = queryBuilder.Where(squirrel.LtOrEq{"t.date": *filters.EndDate}) // LtOrEq means <=
	}
	if len(filters.CategoryIds) > 0 {
		queryBuilder = queryBuilder.Where(squirrel.Eq{"t.category_id": filters.CategoryIds}) // Handles IN (...) clause
	}

	// Generate the final SQL query and arguments
	sql, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build list transaction query: %w", err)
	}

	var transactions []model.Transaction
	err = r.db.SelectContext(ctx, &transactions, sql, args...)
	return transactions, err
}

// DeleteByAccountId removes all transactions associated with a specific account and user.
// This is used when deleting an account.
func (r *pqTransactionRepository) DeleteByAccountId(ctx context.Context, userId, accountId int64) error {
	// The key change is the OR clause to check both source and destination Ids.
	query := `
		DELETE FROM transactions
		WHERE user_id = $1 AND (account_id = $2 OR destination_account_id = $2)
	`
	_, err := r.db.ExecContext(ctx, query, userId, accountId)
	return err
}

// SumExpensesByCategoryAndPeriod calculates the total amount of expenses for a given
// category within a specific date range for a user.
func (r *pqTransactionRepository) SumExpensesByCategoryAndPeriod(ctx context.Context, userID, categoryID int64, startDate, endDate time.Time) (decimal.Decimal, error) {
	var totalExpenses decimal.Decimal
	query := `
        SELECT COALESCE(SUM(amount), 0) 
        FROM transactions
        WHERE user_id = $1
          AND category_id = $2
          AND type = 'expense'
          AND date >= $3 AND date < $4
    `
	// We use GetContext because we expect a single row (the sum) in return.
	err := r.db.GetContext(ctx, &totalExpenses, query, userID, categoryID, startDate, endDate)
	// sql.ErrNoRows is not a problem here; it just means the sum is zero, which COALESCE handles.
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return decimal.Zero, err
	}
	return totalExpenses, nil
}
