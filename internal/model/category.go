package model

import "time"

// Category representa uma categoria para classificar transações.
type Category struct {
	Id     int64           `json:"id" db:"id"`
	UserId int64           `json:"user_id" db:"user_id"`
	Name   string          `json:"name" db:"name"`
	Type   TransactionType `json:"type" db:"type"`
	// Opcional: Adicionar um 'type' ('income' ou 'expense') se quiser forçar
	// que uma categoria seja apenas para receitas ou apenas para despesas.
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
