-- name: CreateUser :one
INSERT INTO users (name, birthday, gender, created_at, updated_at)
VALUES ($1, $2, $3, NOW(), NOW())
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET name = $2,
    birthday = $3,
    gender = $4,
    updated_at = NOW()
WHERE user_id = $1
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE user_id = $1;

-- name: DeleteUser :exec
-- Note: This will cascade delete all user's events, attendances, messages, and likes
DELETE FROM users
WHERE user_id = $1;