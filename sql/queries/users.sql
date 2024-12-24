-- name: CreateUser :one
INSERT INTO users (name, birthday, gender, created_at)
VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET name = $2,
    birthday = $3,
    gender = $4
WHERE user_id = $1
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE user_id = $1;