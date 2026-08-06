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

type Payee = models.Payee

type PayeeService interface {
	GetPayeeByID(ctx context.Context, id uuid.UUID) (Payee, error)
	GetPayees(ctx context.Context) ([]Payee, error)
	CreatePayee(ctx context.Context, name, notes string) (Payee, error)
	UpdatePayee(ctx context.Context, name, notes string, id uuid.UUID) error
	DeletePayeeByID(ctx context.Context, id uuid.UUID, newPayeeName string) error
}

var (
	ErrPayeeInvalidInputName       = errors.New("payee name not provided")
	ErrPayeeInvalidInputID         = errors.New("payee ID not provided")
	ErrPayeeForbidden              = errors.New("forbidden")
	ErrPayeeNotFound               = errors.New("could not find payee")
	ErrPayeesNotFound              = errors.New("could not find payees")
	ErrPayeeInternalCreate         = errors.New("could not create new payee")
	ErrPayeeInternalUpdate         = errors.New("could not update payee")
	ErrPayeeInternalDelete         = errors.New("could not delete payee")
	ErrPayeeInvalidReplacementName = errors.New("replacement payee name not provided")
)

type PayeeAPIService struct {
	repo repository.PayeeRepository
}

func NewPayeeService(repo repository.PayeeRepository) *PayeeAPIService {
	return &PayeeAPIService{repo: repo}
}

func (s *PayeeAPIService) CreatePayee(ctx context.Context, name, notes string) (Payee, error) {
	slog.Error(name)
	if name == "" {
		return Payee{}, ErrPayeeInvalidInputName
	}

	budgetID := UUIDFromContext(ctx, "budget_id")
	payee, err := s.repo.CreatePayee(ctx, database.CreatePayeeParams{
		BudgetID: budgetID,
		Name:     name,
		Notes:    notes,
	})
	if err != nil {
		return Payee{}, ErrPayeeInternalCreate
	}

	return Payee{
		CreatedAt: payee.CreatedAt,
		UpdatedAt: payee.UpdatedAt,
		ID:        payee.ID,
		Meta: models.Meta{
			Name:  payee.Name,
			Notes: payee.Notes,
		},
	}, nil
}

func (s *PayeeAPIService) GetPayeeByID(ctx context.Context, id uuid.UUID) (Payee, error) {
	if id == uuid.Nil {
		return Payee{}, ErrPayeeInvalidInputID
	}

	payee, err := s.repo.GetPayeeByID(ctx, id)
	if err != nil {
		return Payee{}, ErrPayeeNotFound
	}

	budgetID := UUIDFromContext(ctx, "budget_id")
	if budgetID != payee.ID {
		return Payee{}, ErrPayeeForbidden
	}

	return Payee{
		CreatedAt: payee.CreatedAt,
		UpdatedAt: payee.UpdatedAt,
		ID:        payee.ID,
		Meta: models.Meta{
			Name:  payee.Name,
			Notes: payee.Notes,
		},
	}, nil
}

func (s *PayeeAPIService) GetPayees(ctx context.Context) ([]Payee, error) {
	budgetID := UUIDFromContext(ctx, "budget_id")

	dbPayees, err := s.repo.GetPayees(ctx, budgetID)
	if err != nil {
		return nil, ErrPayeesNotFound
	}

	var payees []Payee
	for _, dbPayee := range dbPayees {
		payees = append(payees, Payee{
			ID:        dbPayee.ID,
			CreatedAt: dbPayee.CreatedAt,
			UpdatedAt: dbPayee.UpdatedAt,
			BudgetID:  dbPayee.BudgetID,
			Meta: models.Meta{
				Name:  dbPayee.Name,
				Notes: dbPayee.Notes,
			},
		})
	}

	return payees, nil
}

func (s *PayeeAPIService) UpdatePayee(ctx context.Context, name, notes string, id uuid.UUID) error {
	if name == "" {
		return ErrPayeeInvalidInputName
	}

	_, err := s.repo.UpdatePayee(ctx, database.UpdatePayeeParams{
		ID:    id,
		Name:  name,
		Notes: notes,
	})
	if err != nil {
		return ErrPayeeInternalUpdate
	}

	return nil
}

func (s *PayeeAPIService) DeletePayeeByID(ctx context.Context, id uuid.UUID, newPayeeName string) error {
	if id == uuid.Nil {
		return ErrPayeeInvalidInputID
	}

	payee, err := s.repo.GetPayeeByID(ctx, id)
	if err != nil {
		return ErrPayeeNotFound
	}

	budgetID := UUIDFromContext(ctx, "budget_id")
	if budgetID != payee.BudgetID {
		return ErrPayeeForbidden
	}

	if payeeInUse, err := s.repo.IsPayeeInUse(ctx, id); err != nil {
		return ErrPayeeInternalDelete
	} else if payeeInUse && newPayeeName == "" {
		return ErrPayeeInvalidReplacementName
	}

	err = s.repo.DeletePayeeByID(ctx, id, budgetID, newPayeeName)
	if err != nil {
		return ErrPayeeInternalDelete
	}

	return nil
}
