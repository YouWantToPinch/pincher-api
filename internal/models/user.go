package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
}
