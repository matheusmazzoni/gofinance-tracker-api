package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/repository"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

// StatementPeriod represents the start and end dates of a statement period
type StatementPeriod struct {
	Start time.Time
	End   time.Time
}

// StatementDetails is an internal struct to hold all calculated information for a statement.
type StatementDetails struct {
	AccountName     string
	StatementTotal  decimal.Decimal
	PaymentDueDate  time.Time
	StatementPeriod StatementPeriod
	Transactions    []model.Transaction
}

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

// Do the same for ListAccountsByUserId, iterating over a list of accounts
// and fetching the balance for each one.
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

func (s *AccountService) UpdateAccount(ctx context.Context, acc model.Account) (*model.Account, error) {
	logger := zerolog.Ctx(ctx)

	err := s.repo.Update(ctx, acc)
	if err != nil {
		return nil, err
	}

	balance, err := s.repo.GetCurrentBalance(ctx, acc.Id, acc.UserId)
	if err != nil {
		logger.Error().Err(err).Int64("account_id", acc.Id).Msg("failed to calculate account balance")
		return nil, err
	}

	account, err := s.repo.GetById(ctx, acc.Id, acc.UserId)
	if err != nil {
		return nil, err
	}
	account.Balance = balance

	return account, nil
}

func (s *AccountService) DeleteAccount(ctx context.Context, id, userId int64) error {
	logger := zerolog.Ctx(ctx)

	_, err := s.repo.GetById(ctx, id, userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("account not found to be deleted")
		}
		return err
	}

	err = s.transactionRepo.DeleteByAccountId(ctx, userId, id)
	if err != nil {
		logger.Error().Err(err).Msg("failed to delete transactions for account")
		return errors.New("could not delete associated transactions")
	}

	err = s.repo.Delete(ctx, id, userId)
	if err != nil {
		logger.Error().Err(err).Msg("failed to delete account after deleting transactions")
		return errors.New("could not delete account after removing transactions")
	}

	return nil
}

// GetStatementDetails calculates the statement period and fetches related transactions.
func (s *AccountService) GetStatementDetails(ctx context.Context, userId, accountId int64, targetYear, targetMonth int) (*StatementDetails, error) {
	logger := zerolog.Ctx(ctx)

	// Fetch the account to get its billing cycle details
	account, err := s.repo.GetById(ctx, accountId, userId)
	if err != nil {
		return nil, err
	}

	// Validate account type and billing cycle data
	if account.Type != model.CreditCard {
		return nil, errors.New("operation only valid for credit card accounts")
	}
	if account.StatementClosingDay == nil || account.PaymentDueDay == nil {
		return nil, errors.New("credit card account must have billing cycle data")
	}

	// Calculate statement period dates
	statementPeriod := s.calculateStatementPeriod(targetYear, targetMonth, *account.StatementClosingDay)
	paymentDueDate := s.calculatePaymentDueDate(targetYear, targetMonth, *account.PaymentDueDay)

	logger.Debug().
		Str("start_period", statementPeriod.Start.Format("2006-01-02")).
		Str("end_period", statementPeriod.End.Format("2006-01-02")).
		Msg("Calculated statement period")

	// Fetch all transactions within the calculated period
	transactions, err := s.transactionRepo.ListByAccountAndDateRange(
		ctx, userId, accountId, statementPeriod.Start, statementPeriod.End,
	)
	if err != nil {
		return nil, err
	}

	// Calculate the statement total
	statementTotal := s.calculateStatementTotal(transactions)

	// Build the enriched response object
	statementDetails := &StatementDetails{
		AccountName:     account.Name,
		StatementTotal:  statementTotal,
		PaymentDueDate:  paymentDueDate,
		StatementPeriod: statementPeriod,
		Transactions:    transactions,
	}

	return statementDetails, nil
}

// calculateStatementPeriod calculates the start and end dates for a statement period
func (s *AccountService) calculateStatementPeriod(targetYear, targetMonth, closingDay int) StatementPeriod {
	// Calculate statement period bounds
	currentMonth := time.Month(targetMonth)
	previousMonth, previousYear := s.getPreviousMonth(targetYear, targetMonth)

	// Calculate end of statement (current month)
	endDate := s.calculateStatementDate(targetYear, currentMonth, closingDay)

	// Calculate start of statement (previous month)
	startDate := s.calculateStatementDate(previousYear, previousMonth, closingDay)

	return StatementPeriod{
		Start: startDate,
		End:   endDate,
	}
}

// getPreviousMonth returns the previous month and year, handling year rollover
func (s *AccountService) getPreviousMonth(targetYear, targetMonth int) (time.Month, int) {
	previousMonth := targetMonth - 1
	previousYear := targetYear

	if previousMonth <= 0 {
		previousMonth = 12
		previousYear = targetYear - 1
	}

	return time.Month(previousMonth), previousYear
}

// calculateStatementDate calculates a statement date, adjusting for months with fewer days
func (s *AccountService) calculateStatementDate(year int, month time.Month, closingDay int) time.Time {
	lastDayOfMonth := s.getLastDayOfMonth(year, month)
	adjustedClosingDay := s.adjustClosingDayForMonth(closingDay, lastDayOfMonth)

	return time.Date(year, month, adjustedClosingDay, 0, 0, 0, 0, time.UTC)
}

// getLastDayOfMonth returns the last day of the specified month
func (s *AccountService) getLastDayOfMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// adjustClosingDayForMonth adjusts the closing day if it exceeds the days in the month
func (s *AccountService) adjustClosingDayForMonth(closingDay, lastDayOfMonth int) int {
	if closingDay > lastDayOfMonth {
		return lastDayOfMonth
	}
	return closingDay
}

// calculatePaymentDueDate calculates the payment due date for the statement
func (s *AccountService) calculatePaymentDueDate(targetYear, targetMonth, paymentDueDay int) time.Time {
	return time.Date(targetYear, time.Month(targetMonth), paymentDueDay, 0, 0, 0, 0, time.UTC)
}

// calculateStatementTotal calculates the total amount for all expenses in the statement
func (s *AccountService) calculateStatementTotal(transactions []model.Transaction) decimal.Decimal {
	var total decimal.Decimal

	for _, tx := range transactions {
		if tx.Type == model.Expense {
			total = total.Add(tx.Amount)
		}
		// Note: Could also handle refunds (income on a credit card) here
	}

	return total
}
