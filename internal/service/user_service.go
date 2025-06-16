package service

import (
	"context"

	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/repository"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

// UserService encapsulates the business logic for user-related operations.
// It orchestrates calls to the repository and handles tasks like password hashing.
type UserService struct {
	repo         repository.UserRepository
	categoryRepo repository.CategoryRepository
}

// NewUserService creates a new instance of UserService.
func NewUserService(repo repository.UserRepository, categoryRepo repository.CategoryRepository) *UserService {
	return &UserService{
		repo:         repo,
		categoryRepo: categoryRepo,
	}
}

// CreateUser handles the business logic for creating a new user.
// It hashes the user's password, creates the user record via the repository,
// and then launches a background task to seed default categories for the new user.
func (s *UserService) CreateUser(ctx context.Context, user model.User) (int64, error) {
	// Hash the password for secure storage.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	user.Password = "" // Clear the plaintext password.
	user.PasswordHash = string(hashedPassword)

	// Create the user in the database.
	createdUserId, err := s.repo.Create(ctx, user)
	if err != nil {
		return 0, err
	}

	// Start a background task to create default categories for the new user.
	// We pass the original context to propagate tracing and cancellation.
	go s.seedDefaultCategories(ctx, createdUserId)

	return createdUserId, nil
}

// GetUserById retrieves a user by their unique ID.
func (s *UserService) GetUserById(ctx context.Context, id int64) (*model.User, error) {
	return s.repo.GetById(ctx, id)
}

// seedDefaultCategories creates the initial set of categories for a new user.
// This function is designed to be run in a goroutine as a non-critical background task.
// If a category fails to be created, an error is logged, but the process continues.
func (s *UserService) seedDefaultCategories(ctx context.Context, userId int64) {
	logger := zerolog.Ctx(ctx)

	logger.Info().Int64("userId", userId).Msg("Starting to seed default categories for new user")

	for _, defaultCat := range DefaultCategories {
		catToCreate := defaultCat
		catToCreate.UserId = userId

		// Create each category using the repository.
		_, err := s.categoryRepo.Create(ctx, catToCreate)
		if err != nil {
			logger.Error().
				Err(err).
				Int64("userId", userId).
				Str("categoryName", catToCreate.Name).
				Msg("Failed to seed a default category")
		}
	}
	logger.Info().Int64("userId", userId).Msg("Finished seeding default categories")
}
