-- name: CreateUser :one
INSERT INTO users (name, email, location)
VALUES ($1, $2, $3)
RETURNING id, name, email, location, created_at;