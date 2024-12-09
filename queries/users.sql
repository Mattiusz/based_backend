-- name: CreateUser :one
INSERT INTO users (name, birthday, gender)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE user_id = $1;