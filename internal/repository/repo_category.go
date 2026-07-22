package repository

import (
	"context"
	"fmt"

	"github.com/YouWantToPinch/pincher-api/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CategoryRepository interface {
	GetCategoryByID(ctx context.Context, id uuid.UUID) (database.Category, error)
	GetCategoryIDByName(ctx context.Context, budgetID uuid.UUID, name string) (uuid.UUID, error)
	GetGroupIDByName(ctx context.Context, budgetID uuid.UUID, name string) (uuid.UUID, error)
	GetCategories(ctx context.Context, budgetID, groupID uuid.UUID) ([]database.Category, error)
	CreateCategory(ctx context.Context, arg database.CreateCategoryParams) (database.Category, error)
	UpdateCategory(ctx context.Context, arg database.UpdateCategoryParams) (database.Category, error)
	MergeCategory(ctx context.Context, from, into uuid.UUID) error
	DeleteCategoryByID(ctx context.Context, uuid uuid.UUID) error
	IsCategoryInUse(ctx context.Context, id *uuid.UUID) (bool, error)
}

type PGCategoryRepository struct {
	queries *database.Queries
	Pool    *pgxpool.Pool // for running transactions
}

func NewPGCategoryRepository(db *pgxpool.Pool) *PGCategoryRepository {
	return &PGCategoryRepository{
		queries: database.New(db),
		Pool:    db,
	}
}

func (r *PGCategoryRepository) GetCategoryByID(ctx context.Context, id uuid.UUID) (database.Category, error) {
	return r.queries.GetCategoryByID(ctx, id)
}

func (r *PGCategoryRepository) IsCategoryInUse(ctx context.Context, id *uuid.UUID) (bool, error) {
	return r.queries.IsCategoryInUse(ctx, id)
}

func (r *PGCategoryRepository) GetCategoryIDByName(ctx context.Context, budgetID uuid.UUID, name string) (uuid.UUID, error) {
	return r.queries.GetBudgetCategoryIDByName(ctx, database.GetBudgetCategoryIDByNameParams{
		BudgetID:     budgetID,
		CategoryName: name,
	})
}

func (r *PGCategoryRepository) GetGroupIDByName(ctx context.Context, budgetID uuid.UUID, name string) (uuid.UUID, error) {
	return r.queries.GetBudgetGroupIDByName(ctx, database.GetBudgetGroupIDByNameParams{
		BudgetID:  budgetID,
		GroupName: name,
	})
}

func (r *PGCategoryRepository) GetCategories(ctx context.Context, budgetID, groupID uuid.UUID) ([]database.Category, error) {
	return r.queries.GetCategories(ctx, database.GetCategoriesParams{
		BudgetID: budgetID,
		GroupID:  groupID,
	})
}

func (r *PGCategoryRepository) CreateCategory(ctx context.Context, arg database.CreateCategoryParams) (database.Category, error) {
	return r.queries.CreateCategory(ctx, arg)
}

func (r *PGCategoryRepository) UpdateCategory(ctx context.Context, arg database.UpdateCategoryParams) (database.Category, error) {
	return r.queries.UpdateCategory(ctx, arg)
}

func (r *PGCategoryRepository) MergeCategory(ctx context.Context, from, into uuid.UUID) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // nolint

	q := r.queries.WithTx(tx)

	if exists, err := q.DoesCategoryExist(ctx, into); !exists || err != nil {
		return err
	}

	if err := q.ReassignTransactionCategories(ctx, database.ReassignTransactionCategoriesParams{
		OldCategoryID: from,
		NewCategoryID: into,
	}); err != nil {
		return fmt.Errorf("could not reassign transactions to new category: %w", err)
	}

	if err := q.ReassignAssignmentCategories(ctx, database.ReassignAssignmentCategoriesParams{
		OldCategoryID: from,
		NewCategoryID: into,
	}); err != nil {
		return fmt.Errorf("could not reassign assignments to new category: %w", err)
	}

	if err := q.DeleteCategory(ctx, from); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *PGCategoryRepository) DeleteCategoryByID(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteCategory(ctx, id)
}
