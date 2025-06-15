package model

import "time"

// User representa um usuário do sistema.
// A struct foi desenhada para lidar com a senha de forma segura, separando
// a senha em texto plano (usada apenas para o cadastro) do hash (armazenado no banco).
type User struct {
	Id    int64  `json:"id" db:"id"`
	Name  string `json:"name" db:"name"`
	Email string `json:"email" db:"email"`

	// Password é um campo VIRTUAL, usado APENAS para receber a senha
	// do usuário no momento do cadastro via JSON.
	// A tag `omitempty` garante que ele não será enviado em respostas se estiver vazio.
	// A tag `db:"-"` é CRUCIAL e diz ao sqlx para IGNORAR este campo ao salvar no banco.
	Password string `json:"password,omitempty" db:"-"`

	// PasswordHash é o campo que REALMENTE é salvo no banco de dados.
	// Ele contém o hash bcrypt da senha.
	// A tag `json:"-"` é CRUCIAL e impede que o hash seja exposto
	// em qualquer resposta da API, por segurança.
	PasswordHash string `json:"-" db:"password_hash"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
