package repository

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
)

// TransactionRepository define a interface para o acesso de dados de transações.
// Usar uma interface aqui é uma boa prática para permitir testes e mocks.
type TransactionRepository interface {
	Create(ctx context.Context, tx model.Transaction) (int64, error)
	GetById(ctx context.Context, id, userId int64) (*model.Transaction, error)
	Update(ctx context.Context, tx model.Transaction) error
	Delete(ctx context.Context, id int64, userId int64) error
	ListByUserId(ctx context.Context, userId int64) ([]model.Transaction, error)
	DeleteByAccountId(ctx context.Context, userId, accountId int64) error
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

// ListByUserId busca todas as transações de um usuário específico.
func (r *pqTransactionRepository) ListByUserId(ctx context.Context, userId int64) ([]model.Transaction, error) {
	var transactions []model.Transaction
	query := `
		SELECT 
			t.*, 
			a.name as account_name,
			c.name as category_name
		FROM transactions t
		JOIN accounts a ON t.account_id = a.id
		LEFT JOIN categories c ON t.category_id = c.id
		WHERE t.user_id = $1
		ORDER BY t.date DESC
	`
	err := r.db.SelectContext(ctx, &transactions, query, userId)
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
