package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
)

type BudgetRepository interface {
	Create(ctx context.Context, budget model.Budget) (int64, error)
	GetById(ctx context.Context, id, userId int64) (*model.Budget, error)
	ListByUserAndPeriod(ctx context.Context, userId int64, month, year int) ([]model.Budget, error)
	Update(ctx context.Context, budget model.Budget) error
	Delete(ctx context.Context, id, userId int64) error
}

type pqBudgetRepository struct {
	db *sqlx.DB
}

func NewBudgetRepository(db *sqlx.DB) BudgetRepository {
	return &pqBudgetRepository{db: db}
}

func (r *pqBudgetRepository) Create(ctx context.Context, budget model.Budget) (int64, error) {
	query := `
        INSERT INTO budgets (user_id, category_id, amount, month, year)
        VALUES (:user_id, :category_id, :amount, :month, :year)
        RETURNING id
    `
	rows, err := r.db.NamedQueryContext(ctx, query, budget)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	if rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return 0, fmt.Errorf("failed to scan returned id from budget creation: %w", err)
		}
		return id, nil
	}

	return 0, errors.New("budget creation failed, no id was returned")
}

func (r *pqBudgetRepository) GetById(ctx context.Context, id, userId int64) (*model.Budget, error) {
	var budget model.Budget
	query := `
        SELECT b.*, c.name as category_name
        FROM budgets b
        JOIN categories c ON b.category_id = c.id
        WHERE b.id = $1 AND b.user_id = $2
    `
	err := r.db.GetContext(ctx, &budget, query, id, userId)
	return &budget, err
}

func (r *pqBudgetRepository) ListByUserAndPeriod(ctx context.Context, userId int64, month, year int) ([]model.Budget, error) {
	var budgets []model.Budget
	query := `
        SELECT b.*, c.name as category_name
        FROM budgets b
        JOIN categories c ON b.category_id = c.id
        WHERE b.user_id = $1 AND b.month = $2 AND b.year = $3
        ORDER BY c.name
    `
	err := r.db.SelectContext(ctx, &budgets, query, userId, month, year)
	return budgets, err
}

func (r *pqBudgetRepository) Update(ctx context.Context, budget model.Budget) error {
	query := `
        UPDATE budgets SET amount = :amount, updated_at = NOW()
        WHERE id = :id AND user_id = :user_id
    `
	result, err := r.db.NamedExecContext(ctx, query, budget)
	if err != nil {
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *pqBudgetRepository) Delete(ctx context.Context, id, userId int64) error {
	query := `DELETE FROM budgets WHERE id = $1 AND user_id = $2`
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
