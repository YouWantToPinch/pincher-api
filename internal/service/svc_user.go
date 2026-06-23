package service

import (
	"context"
	"errors"

	"github.com/YouWantToPinch/pincher-api/internal/auth"
	"github.com/YouWantToPinch/pincher-api/internal/database"
	"github.com/YouWantToPinch/pincher-api/internal/models"
	"github.com/YouWantToPinch/pincher-api/internal/repository"
)

type User = models.User

type UserService interface {
	RegisterUser(ctx context.Context, username, password string) (User, error)
	UpdateUserCredentials(ctx context.Context, username, password string) error
	DeleteUser(ctx context.Context, username, password string) error
}

var (
	ErrUserEmptyCredentials = errors.New("missing username or password")
	ErrUserUnauthorized     = errors.New("incorrect username or password")
	ErrUserForbidden        = errors.New("forbidden")
	ErrUserNotFound         = errors.New("user not found")
	ErrUserInternalCreate   = errors.New("could not create new user")
	ErrUserInternalUpdate   = errors.New("could not update user credentials")
	ErrUserInternalDelete   = errors.New("could not delete user")
)

type UserAPIService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserAPIService {
	return &UserAPIService{repo: repo}
}

func (s *UserAPIService) RegisterUser(ctx context.Context, username, password string) (User, error) {
	if username == "" || password == "" {
		return User{}, ErrUserEmptyCredentials
	}

	hashedPass, err := auth.HashPassword(password)
	if err != nil {
		return User{}, ErrUserInternalCreate
	}

	user, err := s.repo.CreateUser(ctx, database.CreateUserParams{
		Username:       username,
		HashedPassword: hashedPass,
	})
	if err != nil {
		return User{}, ErrUserInternalCreate
	}

	return User{
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		ID:        user.ID,
		Username:  user.Username,
	}, nil
}

func (s *UserAPIService) UpdateUserCredentials(ctx context.Context, username, password string) error {
	if username == "" || password == "" {
		return ErrUserEmptyCredentials
	}

	hashedPass, err := auth.HashPassword(password)
	if err != nil {
		return ErrUserInternalUpdate
	}

	userID := UUIDFromContext(ctx, "user_id")
	_, err = s.repo.UpdateUserCredentials(ctx, database.UpdateUserCredentialsParams{
		ID:             userID,
		Username:       username,
		HashedPassword: hashedPass,
	})
	if err != nil {
		return ErrUserInternalUpdate
	}

	return nil
}

func (s *UserAPIService) DeleteUser(ctx context.Context, username, password string) error {
	if username == "" || password == "" {
		return ErrUserEmptyCredentials
	}

	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		// mask database error as unauthorized
		return ErrUserUnauthorized
	}

	userID := UUIDFromContext(ctx, "user_id")
	if userID != user.ID {
		return ErrUserForbidden
	}

	match, err := auth.CheckPasswordHash(password, user.HashedPassword)
	if err != nil || !match {
		return ErrUserUnauthorized
	}

	err = s.repo.DeleteUserByID(ctx, userID)
	if err != nil {
		return ErrUserInternalDelete
	}

	return nil
}
