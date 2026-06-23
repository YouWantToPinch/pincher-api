package repository

import (
	"context"

	"github.com/YouWantToPinch/pincher-api/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	GetUserByUsername(ctx context.Context, arg string) (database.User, error)
	CreateUser(ctx context.Context, arg database.CreateUserParams) (database.User, error)
	UpdateUserCredentials(ctx context.Context, arg database.UpdateUserCredentialsParams) (database.User, error)
	DeleteUserByID(ctx context.Context, uuid uuid.UUID) error
}

type PGUserRepository struct {
	queries *database.Queries
	Pool    *pgxpool.Pool // for running transactions
}

func NewPGUserRepository(db *pgxpool.Pool) *PGUserRepository {
	return &PGUserRepository{
		queries: database.New(db),
		Pool:    db,
	}
}

func (r *PGUserRepository) GetUserByUsername(ctx context.Context, arg string) (database.User, error) {
	return r.queries.GetUserByUsername(ctx, arg)
}

func (r *PGUserRepository) CreateUser(ctx context.Context, arg database.CreateUserParams) (database.User, error) {
	return r.queries.CreateUser(ctx, arg)
}

func (r *PGUserRepository) UpdateUserCredentials(ctx context.Context, arg database.UpdateUserCredentialsParams) (database.User, error) {
	return r.queries.UpdateUserCredentials(ctx, arg)
}

func (r *PGUserRepository) DeleteUserByID(ctx context.Context, uuid uuid.UUID) error {
	return r.queries.DeleteUserByID(ctx, uuid)
}
