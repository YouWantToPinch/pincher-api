package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/YouWantToPinch/pincher-api/internal/database"
	"github.com/YouWantToPinch/pincher-api/internal/models"
	"github.com/YouWantToPinch/pincher-api/internal/repository"
	"github.com/google/uuid"
)

type Group = models.Group

type GroupService interface {
	GetGroupByID(ctx context.Context, id uuid.UUID) (Group, error)
	GetGroups(ctx context.Context) ([]Group, error)
	CreateGroup(ctx context.Context, name, notes string) (Group, error)
	UpdateGroup(ctx context.Context, name, notes string, id uuid.UUID) error
	DeleteGroupByID(ctx context.Context, id uuid.UUID) error
}

var (
	ErrGroupInvalidInputName = errors.New("group name not provided")
	ErrGroupInvalidInputID   = errors.New("group ID not provided")
	ErrGroupForbidden        = errors.New("forbidden")
	ErrGroupNotFound         = errors.New("could not find group")
	ErrGroupsNotFound        = errors.New("could not find groups")
	ErrGroupInternalCreate   = errors.New("could not create new group")
	ErrGroupInternalUpdate   = errors.New("could not update group")
	ErrGroupInternalDelete   = errors.New("could not delete group")
)

type GroupAPIService struct {
	repo repository.GroupRepository
}

func NewGroupService(repo repository.GroupRepository) *GroupAPIService {
	return &GroupAPIService{repo: repo}
}

func (s *GroupAPIService) CreateGroup(ctx context.Context, name, notes string) (Group, error) {
	slog.Error(name)
	if name == "" {
		return Group{}, ErrGroupInvalidInputName
	}

	budgetID := UUIDFromContext(ctx, "budget_id")
	group, err := s.repo.CreateGroup(ctx, database.CreateGroupParams{
		BudgetID: budgetID,
		Name:     name,
		Notes:    notes,
	})
	if err != nil {
		slog.Error(err.Error())
		return Group{}, ErrGroupInternalCreate
	}

	return Group{
		CreatedAt: group.CreatedAt,
		UpdatedAt: group.UpdatedAt,
		ID:        group.ID,
		Meta: models.Meta{
			Name:  group.Name,
			Notes: group.Notes,
		},
	}, nil
}

func (s *GroupAPIService) GetGroupByID(ctx context.Context, id uuid.UUID) (Group, error) {
	if id == uuid.Nil {
		return Group{}, ErrGroupInvalidInputID
	}

	group, err := s.repo.GetGroupByID(ctx, id)
	if err != nil {
		return Group{}, ErrGroupNotFound
	}

	budgetID := UUIDFromContext(ctx, "budget_id")
	if budgetID != group.ID {
		return Group{}, ErrGroupForbidden
	}

	return Group{
		CreatedAt: group.CreatedAt,
		UpdatedAt: group.UpdatedAt,
		ID:        group.ID,
		Meta: models.Meta{
			Name:  group.Name,
			Notes: group.Notes,
		},
	}, nil
}

func (s *GroupAPIService) GetGroups(ctx context.Context) ([]Group, error) {
	budgetID := UUIDFromContext(ctx, "budget_id")

	dbGroups, err := s.repo.GetGroups(ctx, budgetID)
	if err != nil {
		return nil, ErrGroupsNotFound
	}

	var groups []Group
	for _, dbGroup := range dbGroups {
		groups = append(groups, Group{
			ID:        dbGroup.ID,
			CreatedAt: dbGroup.CreatedAt,
			UpdatedAt: dbGroup.UpdatedAt,
			BudgetID:  dbGroup.BudgetID,
			Meta: models.Meta{
				Name:  dbGroup.Name,
				Notes: dbGroup.Notes,
			},
		})
	}

	return groups, nil
}

func (s *GroupAPIService) UpdateGroup(ctx context.Context, name, notes string, id uuid.UUID) error {
	if name == "" {
		return ErrGroupInvalidInputName
	}

	_, err := s.repo.UpdateGroup(ctx, database.UpdateGroupParams{
		ID:    id,
		Name:  name,
		Notes: notes,
	})
	if err != nil {
		return ErrGroupInternalUpdate
	}

	return nil
}

func (s *GroupAPIService) DeleteGroupByID(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrGroupInvalidInputID
	}

	group, err := s.repo.GetGroupByID(ctx, id)
	if err != nil {
		return ErrGroupNotFound
	}

	budgetID := UUIDFromContext(ctx, "budget_id")
	if budgetID != group.BudgetID {
		return ErrGroupForbidden
	}

	err = s.repo.DeleteGroupByID(ctx, id)
	if err != nil {
		return ErrGroupInternalDelete
	}

	return nil
}
