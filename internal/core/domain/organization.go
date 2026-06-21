package domain

import (
	"time"

	"github.com/google/uuid"
)

type Organization struct {
	ID         uuid.UUID `json:"id" db:"id"`
	Name       string    `json:"name" db:"name"`
	OwnerEmail string    `json:"owner_email" db:"owner_email"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}
