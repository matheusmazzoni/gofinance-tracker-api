package service

import (
	"context"

	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) CreateUser(ctx context.Context, user model.User) (int64, error) {
	// Gere o hash da senha
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	user.Password = "" // Limpa a senha em texto plano
	user.PasswordHash = string(hashedPassword)

	return s.repo.Create(ctx, user)
}

func (s *UserService) GetUserById(ctx context.Context, id int64) (*model.User, error) {
	return s.repo.GetById(ctx, id)
}
