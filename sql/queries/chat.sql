
-- name: CreateChatMessage :one
INSERT INTO chat_messages (event_id, user_id, content, edited_at, type, status)
VALUES ($1, $2, $3, NOW(), $4, 'sent')
RETURNING *;

-- name: GetEventMessages :many
SELECT 
    cm.*,
    COUNT(ml.user_id) as number_of_likes,
    EXISTS(
        SELECT 1 FROM message_likes 
        WHERE message_id = cm.message_id AND message_likes.user_id = $2
    ) as is_liked_by_user
FROM chat_messages cm
LEFT JOIN message_likes ml ON cm.message_id = ml.message_id
WHERE cm.event_id = $1
GROUP BY cm.message_id
ORDER BY cm.created_at;

-- name: LikeMessage :exec
INSERT INTO message_likes (message_id, user_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: UnlikeMessage :exec
DELETE FROM message_likes
WHERE message_id = $1 AND user_id = $2;

-- name: DeleteChatMessage :exec
-- Note: This will cascade delete all message likes
DELETE FROM chat_messages
WHERE message_id = $1
AND user_id = $2; -- Optional author check for security