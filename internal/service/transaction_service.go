package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/matheusmazzoni/gofinance-tracker-api/internal/api/dto"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/repository"
)

// TransactionService encapsula a lógica de negócio para transações.
type TransactionService struct {
	repo        repository.TransactionRepository
	accountRepo repository.AccountRepository
}

// NewTransactionService cria uma nova instância do serviço.
func NewTransactionService(repo repository.TransactionRepository, accountRepo repository.AccountRepository) *TransactionService {
	return &TransactionService{
		repo:        repo,
		accountRepo: accountRepo,
	}
}

// CreateTransaction lida com a criação de uma transação.
func (s *TransactionService) CreateTransaction(ctx context.Context, tx model.Transaction) (int64, error) {
	// AQUI ENTRA A LÓGICA DE NEGÓCIO!
	if tx.Amount.IsNegative() || tx.Amount.IsZero() {
		return 0, errors.New("transaction amount must be positive")
	}

	// Verificar se a conta de origem (account_id) existe e pertence ao usuário.
	// O método GetById do accountRepo já faz essa dupla verificação (Id da conta e Id do usuário).
	_, err := s.accountRepo.GetById(ctx, tx.AccountId, tx.UserId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Retorna um erro claro se a conta não for encontrada para aquele usuário.
			return 0, errors.New("source account not found or does not belong to the user")
		}
		// Retorna outros erros inesperados do banco de dados.
		return 0, err
	}

	// Se for uma transferência, validar a conta de destino.
	if tx.Type == model.Transfer {
		// Garante que o Id da conta de destino foi fornecido.
		if tx.DestinationAccountId == nil {
			return 0, errors.New("destination account is required for a transfer")
		}

		// Garante que a conta de origem e destino não são a mesma.
		if tx.AccountId == *tx.DestinationAccountId {
			return 0, errors.New("source and destination accounts cannot be the same")
		}

		// Verifica se a conta de destino existe e também pertence ao usuário.
		_, err := s.accountRepo.GetById(ctx, *tx.DestinationAccountId, tx.UserId)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, errors.New("destination account not found or does not belong to the user")
			}
			return 0, err
		}
	}

	// Se todas as validações passaram, podemos criar a transação com segurança.
	return s.repo.Create(ctx, tx)
}

// GetTransactionById busca uma transação, garantindo que ela pertença ao usuário.
func (s *TransactionService) GetTransactionById(ctx context.Context, id, userId int64) (*model.Transaction, error) {
	return s.repo.GetById(ctx, id, userId)
}

// ListTransactionsByUserId lista todas as transações de um usuário.
func (s *TransactionService) ListTransactionsByUserId(ctx context.Context, userId int64) ([]model.Transaction, error) {
	return s.repo.ListByUserId(ctx, userId)
}

// UpdateTransaction lida com a atualização de uma transação.
// Recebe o userId para garantir a autorização.
func (s *TransactionService) UpdateTransaction(ctx context.Context, id, userId int64, tx model.Transaction) (*model.Transaction, error) {
	// Garante que o Id e o UserId estejam corretos no objeto da transação.
	tx.Id = id
	tx.UserId = userId

	// Validações de negócio
	if tx.Amount.IsNegative() || tx.Amount.IsZero() {
		return nil, errors.New("transaction amount must be positive")
	}
	if tx.DestinationAccountId == nil && tx.Type == model.Transfer {
		return nil, errors.New("transaction type transfer must have destinationAccountId")
	}

	err := s.repo.Update(ctx, tx)
	if err != nil {
		return nil, err
	}

	// Retorna a transação atualizada para o handler poder mostrá-la ao cliente.
	return s.repo.GetById(ctx, id, userId) // <-- MUDANÇA: Passa o userId para garantir a busca segura.
}

// PatchTransaction aplica uma atualização parcial a uma transação.
func (s *TransactionService) PatchTransaction(ctx context.Context, id, userId int64, req dto.PatchTransactionRequest) (*model.Transaction, error) {
	// 1. BUSCAR o estado original da transação.
	txToUpdate, err := s.repo.GetById(ctx, id, userId)
	if err != nil {
		return nil, err // Retorna erro se não encontrar (ex: sql.ErrNoRows)
	}

	if req.Description != nil {
		txToUpdate.Description = *req.Description
	}
	if req.Amount != nil {
		// Validação de negócio para o novo valor
		if req.Amount.IsNegative() || req.Amount.IsZero() {
			return nil, errors.New("transaction amount must be positive")
		}
		txToUpdate.Amount = *req.Amount
	}
	if req.Date != nil {
		txToUpdate.Date = *req.Date
	}
	if req.AccountId != nil {
		// Validação extra: verificar se a nova conta pertence ao usuário
		_, err := s.accountRepo.GetById(ctx, *req.AccountId, userId)
		if err != nil {
			return nil, errors.New("new account not found or does not belong to the user")
		}
		txToUpdate.AccountId = *req.AccountId
	}
	if req.CategoryId != nil {
		txToUpdate.CategoryId = req.CategoryId // CategoryId já é um ponteiro no model
	}

	// 3. SALVAR o objeto completo e mesclado.
	if err := s.repo.Update(ctx, *txToUpdate); err != nil {
		return nil, err
	}

	// Retorna a transação com os dados atualizados para o handler.
	return s.repo.GetById(ctx, id, userId)
}

// DeleteTransaction lida com a exclusão de uma transação.
// Recebe o userId para garantir que o usuário só pode deletar suas próprias transações.
func (s *TransactionService) DeleteTransaction(ctx context.Context, id, userId int64) error { // <-- MUDANÇA: Recebe userId como argumento.
	// Futuramente, você pode adicionar lógicas aqui, como criar um log de auditoria
	// antes de deletar a transação.
	return s.repo.Delete(ctx, id, userId) // <-- MUDANÇA: Passa o userId para o repositório.
}
