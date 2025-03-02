// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0
// source: query.sql

package model

import (
	"context"
	"database/sql"
	"time"
)

const createSession = `-- name: CreateSession :one
INSERT INTO
  sessions (user_id, access_token, refresh_token, expires_at)
VALUES
  (?, ?, ?, ?) RETURNING id
`

type CreateSessionParams struct {
	UserID       int64
	AccessToken  string
	RefreshToken sql.NullString
	ExpiresAt    time.Time
}

func (q *Queries) CreateSession(ctx context.Context, arg CreateSessionParams) (string, error) {
	row := q.db.QueryRowContext(ctx, createSession,
		arg.UserID,
		arg.AccessToken,
		arg.RefreshToken,
		arg.ExpiresAt,
	)
	var id string
	err := row.Scan(&id)
	return id, err
}

const createUser = `-- name: CreateUser :one
INSERT INTO
  users (username, email)
VALUES
  (?, ?) RETURNING id, username, email, created_at, updated_at
`

type CreateUserParams struct {
	Username sql.NullString
	Email    string
}

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (User, error) {
	row := q.db.QueryRowContext(ctx, createUser, arg.Username, arg.Email)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.Email,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const deleteSessionsById = `-- name: DeleteSessionsById :exec
DELETE FROM sessions
WHERE
  id = ?
`

func (q *Queries) DeleteSessionsById(ctx context.Context, id string) error {
	_, err := q.db.ExecContext(ctx, deleteSessionsById, id)
	return err
}

const deleteSessionsByUserId = `-- name: DeleteSessionsByUserId :exec
DELETE FROM sessions
WHERE
  user_id = ?
`

func (q *Queries) DeleteSessionsByUserId(ctx context.Context, userID int64) error {
	_, err := q.db.ExecContext(ctx, deleteSessionsByUserId, userID)
	return err
}

const getSessionById = `-- name: GetSessionById :one
SELECT
  id, user_id, access_token, refresh_token, expires_at, created_at, last_accessed_at
FROM
  sessions
WHERE
  id = ?
LIMIT
  1
`

func (q *Queries) GetSessionById(ctx context.Context, id string) (Session, error) {
	row := q.db.QueryRowContext(ctx, getSessionById, id)
	var i Session
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.AccessToken,
		&i.RefreshToken,
		&i.ExpiresAt,
		&i.CreatedAt,
		&i.LastAccessedAt,
	)
	return i, err
}

const getUserByEmail = `-- name: GetUserByEmail :one
SELECT
  id, username, email, created_at, updated_at
FROM
  users
WHERE
  email = ?
LIMIT
  1
`

func (q *Queries) GetUserByEmail(ctx context.Context, email string) (User, error) {
	row := q.db.QueryRowContext(ctx, getUserByEmail, email)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.Email,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getUserById = `-- name: GetUserById :one
SELECT
  id, username, email, created_at, updated_at
FROM
  users
WHERE
  id = ?
LIMIT
  1
`

func (q *Queries) GetUserById(ctx context.Context, id int64) (User, error) {
	row := q.db.QueryRowContext(ctx, getUserById, id)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.Email,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const listUsers = `-- name: ListUsers :many
SELECT
  id, username, email, created_at, updated_at
FROM
  users
`

func (q *Queries) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := q.db.QueryContext(ctx, listUsers)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []User
	for rows.Next() {
		var i User
		if err := rows.Scan(
			&i.ID,
			&i.Username,
			&i.Email,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
