package service

import (
	"context"
	"errors"

	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/repository"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

type AccountService struct {
	repo            repository.AccountRepository
	transactionRepo repository.TransactionRepository
}

func NewAccountService(repo repository.AccountRepository, transactionRepo repository.TransactionRepository) *AccountService {
	return &AccountService{
		repo:            repo,
		transactionRepo: transactionRepo,
	}
}

func (s *AccountService) CreateAccount(ctx context.Context, acc model.Account) (int64, error) {
	// Lógica de negócio: Ex: verificar se já existe uma conta com o mesmo nome para este usuário.
	return s.repo.Create(ctx, acc)
}

func (s *AccountService) GetAccountById(ctx context.Context, id, userId int64) (*model.Account, error) {
	account, err := s.repo.GetById(ctx, id, userId)
	if err != nil {
		return nil, err
	}

	balance, err := s.repo.GetCurrentBalance(ctx, id, userId)
	if err != nil {
		// Loga o erro, mas talvez não queira que a requisição inteira falhe por isso
		log.Error().Err(err).Int64("account_id", id).Msg("failed to calculate account balance")
		// Você pode decidir retornar o objeto da conta com saldo zero ou o erro.
		// Vamos retornar o erro para sermos explícitos.
		return nil, err
	}

	// 3. Enriquece o objeto do model com o saldo antes de retorná-lo
	account.Balance = balance

	return account, nil
}

// GetAccountByName retrieves an account by its name for a specific user.
func (s *AccountService) GetAccountByName(ctx context.Context, name string, userId int64) (*model.Account, error) {
	account, err := s.repo.GetByName(ctx, name, userId)
	if err != nil {
		return nil, err
	}

	balance, err := s.repo.GetCurrentBalance(ctx, account.Id, userId)
	if err != nil {
		// Loga o erro, mas talvez não queira que a requisição inteira falhe por isso
		log.Error().Err(err).Int64("account_id", account.Id).Msg("failed to calculate account balance")
		// Você pode decidir retornar o objeto da conta com saldo zero ou o erro.
		// Vamos retornar o erro para sermos explícitos.
		return nil, err
	}

	// 3. Enriquece o objeto do model com o saldo antes de retorná-lo
	account.Balance = balance

	return account, nil
}

// Faça o mesmo para ListAccountsByUserId, iterando sobre a lista de contas
// e buscando o saldo para cada uma delas.
func (s *AccountService) ListAccountsByUserId(ctx context.Context, userId int64) ([]model.Account, error) {
	accounts, err := s.repo.ListByUserId(ctx, userId)
	if err != nil {
		return nil, err
	}

	// Para cada conta, busca e atribui o saldo atual.
	for i := range accounts {
		balance, err := s.repo.GetCurrentBalance(ctx, accounts[i].Id, userId)
		if err != nil {
			// Em uma listagem, talvez seja melhor apenas logar e continuar
			log.Warn().Err(err).Int64("account_id", accounts[i].Id).Msg("failed to get balance for account in list")
			// Deixa o saldo como zero se falhar
			accounts[i].Balance = decimal.Zero
		} else {
			accounts[i].Balance = balance
		}
	}

	return accounts, nil
}

func (s *AccountService) UpdateAccount(ctx context.Context, id, userId int64, acc model.Account) (*model.Account, error) {
	acc.Id = id
	acc.UserId = userId
	err := s.repo.Update(ctx, acc)
	if err != nil {
		return nil, err
	}
	balance, err := s.repo.GetCurrentBalance(ctx, id, userId)
	if err != nil {
		// Loga o erro, mas talvez não queira que a requisição inteira falhe por isso
		log.Error().Err(err).Int64("account_id", id).Msg("failed to calculate account balance")
		// Você pode decidir retornar o objeto da conta com saldo zero ou o erro.
		// Vamos retornar o erro para sermos explícitos.
		return nil, err
	}

	account, err := s.repo.GetById(ctx, id, userId)
	if err != nil {
		return nil, err
	}
	account.Balance = balance

	return account, nil
}

func (s *AccountService) DeleteAccount(ctx context.Context, id, userId int64) error {
	// Primeiro, verificamos se a conta realmente existe e pertence ao usuário.
	// Isso evita iniciar uma transação desnecessariamente.
	_, err := s.repo.GetById(ctx, id, userId)
	if err != nil {
		// Se err for sql.ErrNoRows, a conta não foi encontrada.
		// Retornamos o erro original.
		return err
	}

	// PASSO 1: Deletar todas as transações associadas à conta.
	// O filtro por userId aqui é uma camada extra de segurança.
	err = s.transactionRepo.DeleteByAccountId(ctx, userId, id)
	if err != nil {
		// Se não conseguirmos deletar as transações, abortamos a operação.
		log.Error().Err(err).Msg("failed to delete transactions for account")
		return errors.New("could not delete associated transactions")
	}

	// PASSO 2: Agora que as transações "filhas" foram removidas,
	// podemos deletar a conta "pai" com segurança.
	err = s.repo.Delete(ctx, id, userId)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete account after deleting transactions")
		return errors.New("could not delete account after removing transactions")
	}

	return nil
}
