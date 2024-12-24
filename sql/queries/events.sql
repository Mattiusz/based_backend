-- Helper function to extract coordinates
CREATE OR REPLACE FUNCTION get_coordinates(geom geometry)
RETURNS TABLE (latitude float, longitude float) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        ST_Y(geom::geometry) as latitude,
        ST_X(geom::geometry) as longitude;
END;
$$ LANGUAGE plpgsql;

-- name: CreateEvent :one
INSERT INTO events (
    created_at, creator_id, name, location, event_datetime, 
    event_timezone, max_attendees, venue, 
    description, age_range_min, age_range_max,
    allow_female, allow_male, allow_diverse, thumbnail
)
VALUES (
    NOW(), $1, $2, ST_SetSRID(ST_MakePoint($4, $3), 4326), $5, 
    $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
)
RETURNING 
    event_id,
    created_at,
    creator_id,
    name,
    ST_Y(location::geometry) as latitude,
    ST_X(location::geometry) as longitude,
    event_datetime,
    event_timezone,
    max_attendees,
    venue,
    description,
    status,
    age_range_min,
    age_range_max,
    allow_female,
    allow_male,
    allow_diverse,
    thumbnail;

-- name: AddEventCategory :exec
INSERT INTO event_categories (event_id, category)
VALUES ($1, $2);

-- name: GetEventByID :one
SELECT 
    e.*,
    ST_Y(e.location::geometry) as latitude,
    ST_X(e.location::geometry) as longitude,
    json_agg(DISTINCT ec.category) as categories,
    COUNT(DISTINCT cm.message_id) as number_of_comments,
    COUNT(DISTINCT ea.user_id) as number_of_attendees
FROM events e
LEFT JOIN event_categories ec ON e.event_id = ec.event_id
LEFT JOIN chat_messages cm ON e.event_id = cm.event_id
LEFT JOIN event_attendees ea ON e.event_id = ea.event_id
WHERE e.event_id = $1
GROUP BY e.event_id;

-- name: GetNearbyEvents :many
SELECT 
    event_id,
    created_at,
    creator_id,
    name,
    ST_Y(location::geometry) as latitude,
    ST_X(location::geometry) as longitude,
    event_datetime,
    event_timezone,
    max_attendees,
    venue,
    status,
    age_range_min,
    age_range_max,
    allow_female,
    allow_male,
    allow_diverse,
    thumbnail,
    ST_Distance(location, ST_SetSRID(ST_MakePoint($1, $2), 4326)) as distance_meters
FROM events
WHERE 
    status = 'upcoming' AND
    ST_DWithin(
        location,
        ST_SetSRID(ST_MakePoint($1, $2), 4326),
        $3  -- radius in meters
    )
ORDER BY distance_meters
LIMIT $4;

-- name: JoinEvent :exec
INSERT INTO event_attendees (event_id, user_id, gender)
SELECT $1, $2, users.gender
FROM users
WHERE users.user_id = $2
AND NOT EXISTS (
    SELECT 1 FROM event_attendees 
    WHERE event_id = $1 AND user_id = $2
);

-- name: LeaveEvent :exec
DELETE FROM event_attendees
WHERE event_id = $1 AND user_id = $2;

-- name: GetEventAttendeeStats :one
SELECT 
    COUNT(CASE WHEN gender = 'female' THEN 1 END) as female_count,
    COUNT(CASE WHEN gender = 'male' THEN 1 END) as male_count,
    COUNT(CASE WHEN gender = 'diverse' THEN 1 END) as diverse_count
FROM event_attendees
WHERE event_id = $1;

-- name: UpdateEventStatus :one
UPDATE events
SET status = $2
WHERE event_id = $1
RETURNING *;

-- name: SearchEvents :many
SELECT 
    e.*,
    ST_Y(e.location::geometry) as latitude,
    ST_X(e.location::geometry) as longitude,
    json_agg(DISTINCT ec.category) as categories
FROM events e
LEFT JOIN event_categories ec ON e.event_id = ec.event_id
WHERE 
    (LOWER(e.name) LIKE LOWER($1) OR LOWER(e.description) LIKE LOWER($1))
    AND (status = 'upcoming' OR status = 'ongoing')
GROUP BY e.event_id
ORDER BY event_datetime
LIMIT $2 OFFSET $3;

-- name: GetUserEvents :many
SELECT 
    e.*,
    ST_Y(e.location::geometry) as latitude,
    ST_X(e.location::geometry) as longitude
FROM events e
JOIN event_attendees ea ON e.event_id = ea.event_id
WHERE ea.user_id = $1
ORDER BY e.event_datetime DESC;