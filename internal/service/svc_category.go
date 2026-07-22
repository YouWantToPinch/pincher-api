package service

import (
	"context"
	"errors"

	"github.com/YouWantToPinch/pincher-api/internal/database"
	"github.com/YouWantToPinch/pincher-api/internal/models"
	"github.com/YouWantToPinch/pincher-api/internal/repository"
	"github.com/google/uuid"
)

type Category = models.Category

type CategoryService interface {
	GetCategoryByID(ctx context.Context, id uuid.UUID) (Category, error)
	GetCategories(ctx context.Context, groupName string) ([]Category, error)
	CreateCategory(ctx context.Context, name, notes, groupName string) (Category, error)
	UpdateCategory(ctx context.Context, name, notes, groupName string, id uuid.UUID) error
	DeleteCategory(ctx context.Context, id uuid.UUID, newCategoryName string) error
}

var (
	ErrCategoryInvalidInputName = errors.New("category name not provided")
	ErrCategoryInvalidInputID   = errors.New("category ID not provided")
	ErrCategoryForbidden        = errors.New("forbidden")
	ErrCategoryNotFound         = errors.New("could not find category")
	ErrCategoriesNotFound       = errors.New("could not find categories")
	ErrCategoryInternalCreate   = errors.New("could not create new category")
	ErrCategoryInternalUpdate   = errors.New("could not update category")
	ErrCategoryInternalDelete   = errors.New("could not delete category")
)

type CategoryAPIService struct {
	repo repository.CategoryRepository
}

func NewCategoryService(repo repository.CategoryRepository) *CategoryAPIService {
	return &CategoryAPIService{repo: repo}
}

func (s *CategoryAPIService) CreateCategory(ctx context.Context, name, notes, groupName string) (Category, error) {
	if name == "" {
		return Category{}, ErrCategoryInvalidInputName
	}

	budgetID := UUIDFromContext(ctx, "budget_id")

	var assignedGroupID *uuid.UUID
	if groupName != "" {
		if groupID, err := s.repo.GetGroupIDByName(ctx, budgetID, groupName); err != nil {
			return Category{}, ErrGroupNotFound
		} else if groupID != uuid.Nil {
			assignedGroupID = &groupID
		}
	}

	category, err := s.repo.CreateCategory(ctx, database.CreateCategoryParams{
		BudgetID: budgetID,
		Name:     name,
		Notes:    notes,
		GroupID:  assignedGroupID,
	})
	if err != nil {
		return Category{}, ErrCategoryInternalCreate
	}

	return Category{
		CreatedAt: category.CreatedAt,
		UpdatedAt: category.UpdatedAt,
		ID:        category.ID,
		Meta: models.Meta{
			Name:  category.Name,
			Notes: category.Notes,
		},
	}, nil
}

func (s *CategoryAPIService) GetCategoryByID(ctx context.Context, id uuid.UUID) (Category, error) {
	if id == uuid.Nil {
		return Category{}, ErrCategoryInvalidInputID
	}

	category, err := s.repo.GetCategoryByID(ctx, id)
	if err != nil {
		return Category{}, ErrCategoryNotFound
	}

	budgetID := UUIDFromContext(ctx, "budget_id")
	if budgetID != category.ID {
		return Category{}, ErrCategoryForbidden
	}

	return Category{
		CreatedAt: category.CreatedAt,
		UpdatedAt: category.UpdatedAt,
		ID:        category.ID,
		Meta: models.Meta{
			Name:  category.Name,
			Notes: category.Notes,
		},
	}, nil
}

func (s *CategoryAPIService) GetCategories(ctx context.Context, groupName string) ([]Category, error) {
	budgetID := UUIDFromContext(ctx, "budget_id")

	groupID := uuid.Nil
	if groupName != "" {
		var err error
		if groupID, err = s.repo.GetGroupIDByName(ctx, budgetID, groupName); err != nil {
			return nil, ErrGroupNotFound
		}
	}

	dbCategories, err := s.repo.GetCategories(ctx, budgetID, groupID)
	if err != nil {
		return nil, ErrCategoriesNotFound
	}

	var categories []Category
	for _, dbCategory := range dbCategories {
		categories = append(categories, Category{
			ID:        dbCategory.ID,
			CreatedAt: dbCategory.CreatedAt,
			UpdatedAt: dbCategory.UpdatedAt,
			BudgetID:  dbCategory.BudgetID,
			Meta: models.Meta{
				Name:  dbCategory.Name,
				Notes: dbCategory.Notes,
			},
		})
	}

	return categories, nil
}

func (s *CategoryAPIService) UpdateCategory(ctx context.Context, name, notes, newGroupName string, id uuid.UUID) error {
	if name == "" {
		return ErrCategoryInvalidInputName
	}

	budgetID := UUIDFromContext(ctx, "budget_id")

	var assignedGroupID *uuid.UUID
	if newGroupName != "" {
		if groupID, err := s.repo.GetGroupIDByName(ctx, budgetID, newGroupName); err != nil {
			return ErrGroupNotFound
		} else if groupID != uuid.Nil {
			assignedGroupID = &groupID
		}
	}

	_, err := s.repo.UpdateCategory(ctx, database.UpdateCategoryParams{
		ID:      id,
		Name:    name,
		Notes:   notes,
		GroupID: assignedGroupID,
	})
	if err != nil {
		return ErrCategoryInternalUpdate
	}

	return nil
}

func (s *CategoryAPIService) DeleteCategory(ctx context.Context, id uuid.UUID, newCategoryName string) error {
	if id == uuid.Nil {
		return ErrCategoryInvalidInputID
	}

	category, err := s.repo.GetCategoryByID(ctx, id)
	if err != nil {
		return ErrCategoryNotFound
	}

	budgetID := UUIDFromContext(ctx, "budget_id")
	if budgetID != category.BudgetID {
		return ErrCategoryForbidden
	}

	categoryInUse, err := s.repo.IsCategoryInUse(ctx, &id)
	if err != nil {
		return errors.New("could not determine whether category in use")
	}

	if categoryInUse {

		if newCategoryName == "" {
			return ErrCategoryInvalidInputName
		}
		replacementCategory, err := s.repo.GetCategoryIDByName(ctx, budgetID, newCategoryName)
		if err != nil {
			return ErrCategoryNotFound
		}
		if err := s.repo.MergeCategory(ctx, id, replacementCategory); err != nil {
			return ErrCategoryInternalDelete
		}
	} else {
		if err := s.repo.DeleteCategoryByID(ctx, id); err != nil {
			return ErrCategoryInternalDelete
		}
	}

	return nil
}
