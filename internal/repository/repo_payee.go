package repository

import (
	"context"

	"github.com/YouWantToPinch/pincher-api/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PayeeRepository interface {
	GetPayeeByID(ctx context.Context, id uuid.UUID) (database.Payee, error)
	GetPayees(ctx context.Context, budgetID uuid.UUID) ([]database.Payee, error)
	CreatePayee(ctx context.Context, arg database.CreatePayeeParams) (database.Payee, error)
	UpdatePayee(ctx context.Context, arg database.UpdatePayeeParams) (database.Payee, error)
	IsPayeeInUse(ctx context.Context, id uuid.UUID) (bool, error)
	DeletePayeeByID(ctx context.Context, id, budgetID uuid.UUID, newPayeeName string) error
}

type PGPayeeRepository struct {
	queries *database.Queries
	Pool    *pgxpool.Pool // for running transactions
}

func NewPGPayeeRepository(db *pgxpool.Pool) *PGPayeeRepository {
	return &PGPayeeRepository{
		queries: database.New(db),
		Pool:    db,
	}
}

func (r *PGPayeeRepository) GetPayeeByID(ctx context.Context, id uuid.UUID) (database.Payee, error) {
	return r.queries.GetPayeeByID(ctx, id)
}

func (r *PGPayeeRepository) GetPayees(ctx context.Context, budgetID uuid.UUID) ([]database.Payee, error) {
	return r.queries.GetBudgetPayees(ctx, budgetID)
}

func (r *PGPayeeRepository) CreatePayee(ctx context.Context, arg database.CreatePayeeParams) (database.Payee, error) {
	return r.queries.CreatePayee(ctx, arg)
}

func (r *PGPayeeRepository) UpdatePayee(ctx context.Context, arg database.UpdatePayeeParams) (database.Payee, error) {
	return r.queries.UpdatePayee(ctx, arg)
}

func (r *PGPayeeRepository) IsPayeeInUse(ctx context.Context, id uuid.UUID) (bool, error) {
	payeeInUse, err := r.queries.IsPayeeInUse(ctx, id)
	if err != nil {
		return false, err
	}
	return payeeInUse, nil
}

func (r *PGPayeeRepository) DeletePayeeByID(ctx context.Context, id, budgetID uuid.UUID, newPayeeName string) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // nolint

	q := r.queries.WithTx(tx)

	payeeInUse, err := q.IsPayeeInUse(ctx, id)
	if err != nil {
		return err
	}

	if payeeInUse {

		PayeeID, err := r.queries.GetBudgetPayeeIDByName(ctx,
			database.GetBudgetPayeeIDByNameParams{
				PayeeName: newPayeeName,
				BudgetID:  budgetID,
			})
		if err != nil {
			return err
		}

		err = q.ReassignTransactionPayees(ctx, database.ReassignTransactionPayeesParams{
			OldPayeeID: id,
			NewPayeeID: PayeeID,
		})
		if err != nil {
			return err
		}
	}

	if err := q.DeletePayee(ctx, id); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
