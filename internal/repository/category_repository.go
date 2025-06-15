package repository

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
)

type CategoryRepository interface {
	Create(ctx context.Context, ct model.Category) (int64, error)
	GetById(ctx context.Context, id, userId int64) (*model.Category, error)
	ListByUserId(ctx context.Context, userId int64) ([]model.Category, error)
	Update(ctx context.Context, ct model.Category) error
	Delete(ctx context.Context, id, userId int64) error
}

type pqCategoryRepository struct {
	db *sqlx.DB
}

func NewCategoryRepository(db *sqlx.DB) CategoryRepository {
	return &pqCategoryRepository{db: db}
}

func (r *pqCategoryRepository) Create(ctx context.Context, ct model.Category) (int64, error) {
	query := `INSERT INTO categories (user_id, name, type) VALUES (:user_id, :name, :type) RETURNING id`
	rows, err := r.db.NamedQueryContext(ctx, query, ct)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var id int64
	if rows.Next() {
		err = rows.Scan(&id)
	}
	return id, err
}

func (r *pqCategoryRepository) GetById(ctx context.Context, id, userId int64) (*model.Category, error) {
	var ct model.Category
	query := `SELECT * FROM categories WHERE id = $1 AND user_id = $2`
	err := r.db.GetContext(ctx, &ct, query, id, userId)
	return &ct, err
}

func (r *pqCategoryRepository) ListByUserId(ctx context.Context, userId int64) ([]model.Category, error) {
	var categories []model.Category
	query := `SELECT * FROM categories WHERE user_id = $1 ORDER BY name`
	err := r.db.SelectContext(ctx, &categories, query, userId)
	return categories, err
}

func (r *pqCategoryRepository) Update(ctx context.Context, ct model.Category) error {
	query := `UPDATE categories SET name = :name, type = :type, updated_at = NOW() WHERE id = :id AND user_id = :user_id`
	result, err := r.db.NamedExecContext(ctx, query, ct)
	if err != nil {
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *pqCategoryRepository) Delete(ctx context.Context, id, userId int64) error {
	// ATENÇÃO: A constraint 'ON DELETE RESTRICT' na tabela 'transactions'
	// impedirá a exclusão de uma conta que tenha transações.
	query := `DELETE FROM categories WHERE id = $1 AND user_id = $2`
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
