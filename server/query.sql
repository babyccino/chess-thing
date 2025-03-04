-- name: GetUserById :one
SELECT
  *
FROM
  users
WHERE
  id = ?
LIMIT
  1;

-- name: GetUserByEmail :one
SELECT
  *
FROM
  users
WHERE
  email = ?
LIMIT
  1;

-- name: ListUsers :many
SELECT
  *
FROM
  users;

-- name: GetSessionById :one
SELECT
  *
FROM
  sessions
WHERE
  id = ?
LIMIT
  1;

-- name: GetSessionExists :one
SELECT
  EXISTS (
    SELECT
      1
    FROM
      sessions
    WHERE
      id = ?
    LIMIT
      1
  );

-- name: GetSessionByIdAndUser :one
SELECT
  u.id as user_id,
  u.username as user_username,
  u.email as user_email,
  u.created_at as user_created_at,
  u.updated_at as user_updated_at,
  s.id as session_id,
  s.user_id as session_user_id,
  s.access_token as session_access_token,
  s.refresh_token as session_refresh_token,
  s.expires_at as session_expires_at,
  s.created_at as session_created_at,
  s.last_accessed_at as session_last_accessed_at
FROM
  sessions as s
  INNER JOIN users as u ON s.user_id = u.id
WHERE
  s.id = ?
LIMIT
  1;

-- name: CreateUser :one
INSERT INTO
  users (username, email)
VALUES
  (?, ?) RETURNING *;

-- name: CreateSession :one
INSERT INTO
  sessions (
    id,
    user_id,
    access_token,
    refresh_token,
    expires_at
  )
VALUES
  (?, ?, ?, ?, ?) RETURNING id;

-- name: DeleteSessionsByUserId :exec
DELETE FROM sessions
WHERE
  user_id = ?;

-- name: DeleteSessionsById :exec
DELETE FROM sessions
WHERE
  id = ?;
