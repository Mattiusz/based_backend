-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS postgis;

-- Enum types
CREATE TYPE gender_type AS ENUM ('male', 'female', 'diverse');
CREATE TYPE event_status_type AS ENUM ('unspecified', 'upcoming', 'ongoing', 'completed', 'rescheduled', 'cancelled');
CREATE TYPE event_category_type AS ENUM (
    'unspecified', 'sports', 'musicAndMovies', 'art', 'foodAndDrinks', 'partyAndGames',
    'business', 'nature', 'technology', 'travel', 'education', 'charity', 'other'
);

-- Users table
CREATE TABLE users (
    user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    birthday DATE NOT NULL,
    gender gender_type NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Events table
CREATE TABLE events (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_id UUID NOT NULL REFERENCES users(user_id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    name TEXT NOT NULL,
    venue TEXT,
    description TEXT,
    categories event_category_type[] DEFAULT '{}',
    status event_status_type NOT NULL DEFAULT 'upcoming',
    thumbnail BYTEA DEFAULT NULL,
    location geometry(Point, 4326) NOT NULL,
    datetime TIMESTAMP WITH TIME ZONE NOT NULL,
    max_attendees INTEGER NOT NULL CHECK (max_attendees >= 0),
    age_range_min INTEGER CHECK (age_range_min >= 0) DEFAULT 0,
    age_range_max INTEGER CHECK (age_range_max >= age_range_min) DEFAULT 99,
    allow_female BOOLEAN NOT NULL DEFAULT true,
    allow_male BOOLEAN NOT NULL DEFAULT true,
    allow_diverse BOOLEAN NOT NULL DEFAULT true,
    CONSTRAINT valid_age_range CHECK (
        (age_range_min IS NULL AND age_range_max IS NULL) OR
        (age_range_min IS NOT NULL AND age_range_max IS NOT NULL)
    )
);

-- Event attendees with gender tracking
CREATE TABLE event_attendees (
    event_id UUID NOT NULL REFERENCES events(event_id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    gender gender_type NOT NULL,
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (event_id, user_id)
);

-- Chat messages
CREATE TABLE chat_messages (
    message_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(event_id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    comment TEXT NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    message_index INTEGER NOT NULL,
    UNIQUE (event_id, message_index)
);

-- Message likes
CREATE TABLE message_likes (
    message_id UUID NOT NULL REFERENCES chat_messages(message_id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (message_id, user_id)
);

-- Create indexes for better query performance
CREATE INDEX idx_events_creator ON events(creator_id);
CREATE INDEX idx_events_datetime ON events(datetime);
CREATE INDEX idx_events_status ON events(status);
CREATE INDEX idx_chat_messages_event ON chat_messages(event_id);
CREATE INDEX idx_event_attendees_user ON event_attendees(user_id);
CREATE INDEX idx_event_attendees_event ON event_attendees(event_id);
CREATE INDEX idx_message_likes_message ON message_likes(message_id);

-- Create spatial indexes for location queries
CREATE INDEX idx_events_location ON events USING GIST(location);

-- Create statistics views
CREATE MATERIALIZED VIEW event_attendee_statistics AS
SELECT 
    event_id,
    COUNT(CASE WHEN gender = 'female' THEN 1 END) as female_count,
    COUNT(CASE WHEN gender = 'male' THEN 1 END) as male_count,
    COUNT(CASE WHEN gender = 'diverse' THEN 1 END) as diverse_count
FROM event_attendees
GROUP BY event_id;

-- Create index on materialized view for better performance
CREATE UNIQUE INDEX idx_event_attendee_stats ON event_attendee_statistics(event_id);