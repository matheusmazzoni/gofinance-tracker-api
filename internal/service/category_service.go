package service

import (
	"context"

	"github.com/matheusmazzoni/gofinance-tracker-api/internal/model"
	"github.com/matheusmazzoni/gofinance-tracker-api/internal/repository"
)

type CategoryService struct {
	repo repository.CategoryRepository
}

func NewCategoryService(repo repository.CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

func (s *CategoryService) CreateCategory(ctx context.Context, acc model.Category) (int64, error) {
	// Lógica de negócio: Ex: verificar se já existe uma conta com o mesmo nome para este usuário.
	return s.repo.Create(ctx, acc)
}

func (s *CategoryService) GetCategoryById(ctx context.Context, id, userId int64) (*model.Category, error) {
	return s.repo.GetById(ctx, id, userId)
}

func (s *CategoryService) ListCategoriesByUserId(ctx context.Context, userId int64) ([]model.Category, error) {
	return s.repo.ListByUserId(ctx, userId)
}

func (s *CategoryService) UpdateCategory(ctx context.Context, id, userId int64, acc model.Category) (*model.Category, error) {
	acc.Id = id
	acc.UserId = userId
	err := s.repo.Update(ctx, acc)
	if err != nil {
		return nil, err
	}
	return s.repo.GetById(ctx, id, userId)
}

func (s *CategoryService) DeleteCategory(ctx context.Context, id, userId int64) error {
	// Verifica se nao tem nenhuma transacao nessa categoria
	return s.repo.Delete(ctx, id, userId)
}
