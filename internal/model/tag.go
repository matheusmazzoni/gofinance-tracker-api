package model

import "time"

// Tag permite uma classificação flexível e múltipla para transações.
type Tag struct {
	Id        int64     `json:"id" db:"id"`
	UserId    int64     `json:"user_id" db:"user_id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// TransactionTag é a tabela de junção para a relação M-M.
type TransactionTag struct {
	TransactionId int64 `db:"transaction_id"`
	TagId         int64 `db:"tag_id"`
}
