package dto

import "time"

// CreateUserRequest é o DTO para a requisição de criação de usuário.
// Contém apenas os campos que o cliente deve enviar, com as devidas validações.
type CreateUserRequest struct {
	Name     string `json:"name" binding:"required,min=2"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// UserResponse é o DTO para a resposta de um usuário.
// Contém apenas os campos públicos e seguros que a API deve retornar.
type UserResponse struct {
	Id        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}
