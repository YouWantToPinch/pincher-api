package repository

import (
	"context"

	"github.com/YouWantToPinch/pincher-api/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GroupRepository interface {
	GetGroupByID(ctx context.Context, id uuid.UUID) (database.Group, error)
	GetGroups(ctx context.Context, budgetID uuid.UUID) ([]database.Group, error)
	CreateGroup(ctx context.Context, arg database.CreateGroupParams) (database.Group, error)
	UpdateGroup(ctx context.Context, arg database.UpdateGroupParams) (database.Group, error)
	DeleteGroupByID(ctx context.Context, uuid uuid.UUID) error
}

type PGGroupRepository struct {
	queries *database.Queries
	Pool    *pgxpool.Pool // for running transactions
}

func NewPGGroupRepository(db *pgxpool.Pool) *PGGroupRepository {
	return &PGGroupRepository{
		queries: database.New(db),
		Pool:    db,
	}
}

func (r *PGGroupRepository) GetGroupByID(ctx context.Context, id uuid.UUID) (database.Group, error) {
	return r.queries.GetGroupByID(ctx, id)
}

func (r *PGGroupRepository) GetGroups(ctx context.Context, budgetID uuid.UUID) ([]database.Group, error) {
	return r.queries.GetGroupsByBudgetID(ctx, budgetID)
}

func (r *PGGroupRepository) CreateGroup(ctx context.Context, arg database.CreateGroupParams) (database.Group, error) {
	return r.queries.CreateGroup(ctx, arg)
}

func (r *PGGroupRepository) UpdateGroup(ctx context.Context, arg database.UpdateGroupParams) (database.Group, error) {
	return r.queries.UpdateGroup(ctx, arg)
}

func (r *PGGroupRepository) DeleteGroupByID(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteGroupByID(ctx, id)
}
