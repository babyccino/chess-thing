// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0

package model

import (
	"database/sql"
	"time"
)

type Session struct {
	ID             string
	UserID         string
	AccessToken    string
	RefreshToken   sql.NullString
	ExpiresAt      time.Time
	CreatedAt      time.Time
	LastAccessedAt time.Time
}

type User struct {
	ID        string
	Username  sql.NullString
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}
