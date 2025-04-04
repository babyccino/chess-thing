// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0

package model

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID             uuid.UUID
	UserID         string
	AccessToken    string
	RefreshToken   sql.NullString
	ExpiresAt      time.Time
	CreatedAt      time.Time
	LastAccessedAt time.Time
}

type User struct {
	ID        uuid.UUID
	Username  sql.NullString
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}
