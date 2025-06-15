package dto

// ErrorResponse é o DTO padrão para respostas de erro da API.
type ErrorResponse struct {
	// A mensagem de erro descritiva.
	Error string `json:"error" example:"a descrição do erro aparece aqui"`
}
