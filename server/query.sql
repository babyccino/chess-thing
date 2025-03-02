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

-- name: CreateUser :one
INSERT INTO
  users (username, email)
VALUES
  (?, ?) RETURNING *;

-- name: CreateSession :one
INSERT INTO
  sessions (user_id, access_token, refresh_token, expires_at)
VALUES
  (?, ?, ?, ?) RETURNING id;

-- name: DeleteSessionsByUserId :exec
DELETE FROM sessions
WHERE
  user_id = ?;

-- name: DeleteSessionsById :exec
DELETE FROM sessions
WHERE
  id = ?;
