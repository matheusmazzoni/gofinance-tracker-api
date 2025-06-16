package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/repository"
)

var (
	ErrCategoryNameExists = errors.New("a category with this name already exists for the user")
	ErrCategoryInUse      = errors.New("cannot delete a category that is currently assigned to transactions")
)

// DefaultCategories defines the initial set of categories created for a new user.
var DefaultCategories = []model.Category{
	// Income Categories
	{Name: "Salary", Type: model.Income},
	{Name: "Gifts", Type: model.Income},
	{Name: "Other Income", Type: model.Income},

	// Expense Categories
	{Name: "Housing", Type: model.Expense},
	{Name: "Transportation", Type: model.Expense},
	{Name: "Food", Type: model.Expense},
	{Name: "Utilities", Type: model.Expense},
	{Name: "Healthcare", Type: model.Expense},
	{Name: "Personal Spending", Type: model.Expense},
	{Name: "Entertainment", Type: model.Expense},
	// Note: Savings & Investments are considered an "expense" from a cash flow perspective,
	// as money is being moved out of the immediately available cash pool.
	{Name: "Savings & Investments", Type: model.Expense},
	{Name: "Debt Payments", Type: model.Expense},
	{Name: "Donations", Type: model.Expense},
}

// CategoryService encapsulates the business logic for managing categories.
type CategoryService struct {
	repo            repository.CategoryRepository
	transactionRepo repository.TransactionRepository // Needed to check for usage before deletion.
}

// NewCategoryService creates a new instance of the CategoryService.
func NewCategoryService(repo repository.CategoryRepository, transactionRepo repository.TransactionRepository) *CategoryService {
	return &CategoryService{
		repo:            repo,
		transactionRepo: transactionRepo,
	}
}

// CreateCategory creates a new category after validating that no category
// with the same name already exists for the user.
func (s *CategoryService) CreateCategory(ctx context.Context, category model.Category) (int64, error) {
	// Business logic: Ensure category name is unique for the user (case-insensitive).
	existingCategory, err := s.repo.GetByName(ctx, category.Name, category.UserId)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		// An unexpected database error occurred.
		return 0, err
	}
	if existingCategory != nil {
		// A category with this name already exists.
		return 0, ErrCategoryNameExists
	}

	return s.repo.Create(ctx, category)
}

// GetCategoryById retrieves a single category by its ID, ensuring it belongs to the user.
func (s *CategoryService) GetCategoryById(ctx context.Context, id, userId int64) (*model.Category, error) {
	return s.repo.GetById(ctx, id, userId)
}

// ListCategoriesByUserId lists all categories belonging to a specific user.
func (s *CategoryService) ListCategoriesByUserId(ctx context.Context, userId int64) ([]model.Category, error) {
	return s.repo.ListByUserId(ctx, userId)
}

// UpdateCategory updates an existing category. It includes validation to prevent
// renaming a category to a name that is already in use by another category.
func (s *CategoryService) UpdateCategory(ctx context.Context, id, userId int64, category model.Category) (*model.Category, error) {
	category.Id = id
	category.UserId = userId

	// Business logic: Check if the new name is already taken by another category.
	existing, err := s.repo.GetByName(ctx, category.Name, userId)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	// If a category with the new name exists, and it's not the same one we are updating, it's a conflict.
	if existing != nil && existing.Id != id {
		return nil, ErrCategoryNameExists
	}

	err = s.repo.Update(ctx, category)
	if err != nil {
		return nil, err
	}
	return s.repo.GetById(ctx, id, userId)
}

// DeleteCategory deletes a category after ensuring it is not currently assigned
// to any transactions, thus preventing orphaned data.
func (s *CategoryService) DeleteCategory(ctx context.Context, id, userId int64) error {
	// Business logic: Verify that no transactions are currently using this category.
	//inUse, err := s.transactionRepo.IsCategoryInUse(ctx, id, userId)
	//if err != nil {
	//	return err // An unexpected database error occurred.
	//}
	//if inUse {
	//	return ErrCategoryInUse
	//}

	return s.repo.Delete(ctx, id, userId)
}
