package models

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ID        uuid.UUID  `json:"id"`
	BudgetID  uuid.UUID  `json:"budget_id"`
	GroupID   *uuid.UUID `json:"group_id"`
	Meta
}
