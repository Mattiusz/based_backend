-- name: CreateEvent :one
INSERT INTO events (
    creator_id, name, location, datetime, 
    max_attendees, venue, description, age_range_min, age_range_max,
    allow_female, allow_male, allow_diverse, thumbnail, status, categories
)
VALUES (
    $1, $2, ST_SetSRID(ST_MakePoint($4, $3), 4326)::geometry, $5, 
    $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
)
RETURNING 
    event_id,
    created_at,
    updated_at,
    creator_id,
    name,
    ST_Y(location::geometry) as latitude,
    ST_X(location::geometry) as longitude,
    datetime,
    max_attendees,
    venue,
    description,
    status,
    age_range_min,
    age_range_max,
    allow_female,
    allow_male,
    allow_diverse,
    thumbnail,
    status,
    categories;

-- name: GetEventByID :one
SELECT 
    e.event_id,
    e.created_at,
    e.updated_at,
    e.creator_id,
    e.name,
    ST_Y(e.location::geometry) as latitude,
    ST_X(e.location::geometry) as longitude,
    e.datetime,
    e.max_attendees,
    e.venue,
    e.description,
    e.status,
    e.age_range_min,
    e.age_range_max,
    e.allow_female,
    e.allow_male,
    e.allow_diverse,
    e.thumbnail,
    e.categories,
    COUNT(DISTINCT cm.message_id) as number_of_comments,
    COUNT(DISTINCT ea.user_id) as number_of_attendees
FROM events e
LEFT JOIN chat_messages cm ON e.event_id = cm.event_id
LEFT JOIN event_attendees ea ON e.event_id = ea.event_id
WHERE e.event_id = $1
GROUP BY 
    e.event_id,
    e.created_at,
    e.updated_at,
    e.creator_id,
    e.name,
    e.location,
    e.datetime,
    e.max_attendees,
    e.venue,
    e.description,
    e.status,
    e.age_range_min,
    e.age_range_max,
    e.allow_female,
    e.allow_male,
    e.allow_diverse,
    e.thumbnail,
    e.categories;

-- name: GetNearbyEventsByStatus :many
SELECT 
    e.event_id,
    e.created_at,
    e.creator_id,
    e.name,
    ST_Y(e.location::geometry) as latitude,
    ST_X(e.location::geometry) as longitude,
    e.datetime,
    e.max_attendees,
    e.venue,
    e.status,
    e.age_range_min,
    e.age_range_max,
    e.allow_female,
    e.allow_male,
    e.allow_diverse,
    e.categories,
    ST_Distance(
        ST_Transform(e.location, 3857),
        ST_Transform(ST_SetSRID(ST_MakePoint($3, $2), 4326), 3857)
    ) as distance_meters,
    COUNT(DISTINCT ea.user_id) as number_of_attendees
FROM events e
LEFT JOIN event_attendees ea ON e.event_id = ea.event_id
JOIN users u ON u.user_id = $5  -- Join users table once at the top level
WHERE 
    e.status = $1 AND
    ST_DWithin(
        ST_Transform(e.location, 3857),
        ST_Transform(ST_SetSRID(ST_MakePoint($3, $2), 4326), 3857),
        $4  -- radius in meters
    ) AND
    CASE 
        WHEN u.gender = 'female' THEN e.allow_female = true
        WHEN u.gender = 'male' THEN e.allow_male = true
        WHEN u.gender = 'diverse' THEN e.allow_diverse = true
    END
GROUP BY 
    e.event_id,
    e.created_at,
    e.creator_id,
    e.name,
    ST_Y(e.location::geometry),
    ST_X(e.location::geometry),
    e.datetime,
    e.max_attendees,
    e.venue,
    e.status,
    e.age_range_min,
    e.age_range_max,
    e.allow_female,
    e.allow_male,
    e.allow_diverse,
    e.categories
ORDER BY distance_meters
LIMIT $6;

-- name: GetEventAttendeeStats :one
SELECT 
    COUNT(CASE WHEN gender = 'female' THEN 1 END) as female_count,
    COUNT(CASE WHEN gender = 'male' THEN 1 END) as male_count,
    COUNT(CASE WHEN gender = 'diverse' THEN 1 END) as diverse_count
FROM event_attendees
WHERE event_id = $1;

-- name: UpdateEvent :one
UPDATE events
SET updated_at = NOW(), status = $2, name = $3, location = ST_SetSRID(ST_MakePoint($5, $4), 4326)::geometry,
    datetime = $6, max_attendees = $7, venue = $8, description = $9, 
    age_range_min = $10, age_range_max = $11, allow_male = $12, allow_female = $13, allow_diverse = $14,
    categories = $15
WHERE event_id = $1
RETURNING 
    event_id,
    created_at,
    updated_at,
    creator_id,
    name,
    ST_Y(location::geometry) as latitude,
    ST_X(location::geometry) as longitude,
    datetime,
    max_attendees,
    venue,
    description,
    status,
    age_range_min,
    age_range_max,
    allow_female,
    allow_male,
    allow_diverse,
    categories;

-- name: GetUserEvents :many
SELECT 
    e.event_id,
    e.created_at,
    e.updated_at,
    e.creator_id,
    e.name,
    ST_Y(e.location::geometry) as latitude,
    ST_X(e.location::geometry) as longitude,
    e.datetime,
    e.max_attendees,
    e.venue,
    e.description,
    e.status,
    e.age_range_min,
    e.age_range_max,
    e.allow_female,
    e.allow_male,
    e.allow_diverse,
    e.categories,
    COALESCE(cm.comment_count, 0) as number_of_comments,
    COALESCE(ea.attendee_count, 0) as number_of_attendees
FROM events e
INNER JOIN event_attendees my_attendance ON e.event_id = my_attendance.event_id 
    AND my_attendance.user_id = $1
LEFT JOIN (
    SELECT event_id, COUNT(*) as comment_count
    FROM chat_messages
    GROUP BY event_id
) cm ON e.event_id = cm.event_id
LEFT JOIN (
    SELECT event_id, COUNT(*) as attendee_count
    FROM event_attendees
    GROUP BY event_id
) ea ON e.event_id = ea.event_id
ORDER BY e.datetime DESC
LIMIT $2;

-- name: JoinEvent :exec
INSERT INTO event_attendees (event_id, user_id, gender)
SELECT $1, $2, u.gender
FROM users u
JOIN events e ON e.event_id = $1
WHERE u.user_id = $2
AND NOT EXISTS (
    SELECT 1 FROM event_attendees 
    WHERE event_id = $1 AND user_id = $2
);

-- name: LeaveEvent :exec
DELETE FROM event_attendees
WHERE event_id = $1 AND user_id = $2;

-- name: DeleteEvent :exec
-- Note: This will cascade delete all event attendances, messages, and likes
DELETE FROM events
WHERE event_id = $1
AND creator_id = $2; -- Optional creator check for security