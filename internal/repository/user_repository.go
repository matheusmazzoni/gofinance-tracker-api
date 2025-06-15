package repository

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
)

type UserRepository interface {
	Create(ctx context.Context, user model.User) (int64, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetById(ctx context.Context, id int64) (*model.User, error)
}

type pqUserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) UserRepository {
	return &pqUserRepository{db: db}
}

func (r *pqUserRepository) Create(ctx context.Context, user model.User) (int64, error) {
	query := `INSERT INTO users (name, email, password_hash) VALUES (:name, :email, :password_hash) RETURNING id`
	rows, err := r.db.NamedQueryContext(ctx, query, user)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var id int64
	if rows.Next() {
		rows.Scan(&id)
	}
	return id, nil
}

func (r *pqUserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	query := `SELECT * FROM users WHERE email = $1`
	err := r.db.GetContext(ctx, &user, query, email)
	return &user, err
}

func (r *pqUserRepository) GetById(ctx context.Context, id int64) (*model.User, error) {
	var user model.User
	query := `SELECT id, name, email, created_at, updated_at FROM users WHERE id = $1`

	// Usamos o db.Get do sqlx que é perfeito para buscar um único registro
	err := r.db.GetContext(ctx, &user, query, id)

	// Se o usuário não for encontrado, o sqlx retorna um erro sql.ErrNoRows,
	// que será tratado pela camada de serviço/handler para retornar um 404 Not Found.
	return &user, err
}
