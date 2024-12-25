package repositories

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mattiusz/based_backend/internal/gen/sqlc"
)

type EventRepository interface {
	CreateEvent(ctx context.Context, req *sqlc.CreateEventParams) (*sqlc.CreateEventRow, error)
	GetEventByID(ctx context.Context, eventID pgtype.UUID) (*sqlc.GetEventByIDRow, error)
	GetNearbyEvents(ctx context.Context, params *sqlc.GetNearbyEventsByStatusAndGenderParams) ([]sqlc.GetNearbyEventsByStatusAndGenderRow, error)
	GetUserEvents(ctx context.Context, params *sqlc.GetUserEventsParams) ([]sqlc.GetUserEventsRow, error)
	GetEventAttendeeStats(ctx context.Context, eventID pgtype.UUID) (*sqlc.GetEventAttendeeStatsRow, error)
	UpdateEvent(ctx context.Context, params *sqlc.UpdateEventParams) (*sqlc.UpdateEventRow, error)
	JoinEvent(ctx context.Context, params *sqlc.JoinEventParams) error
	LeaveEvent(ctx context.Context, params *sqlc.LeaveEventParams) error
}

type eventRepository struct {
	queries *sqlc.Queries
}

func NewEventRepository(q *sqlc.Queries) EventRepository {
	return &eventRepository{queries: q}
}

func (r *eventRepository) CreateEvent(ctx context.Context, req *sqlc.CreateEventParams) (*sqlc.CreateEventRow, error) {
	event, err := r.queries.CreateEvent(ctx, *req)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *eventRepository) GetEventByID(ctx context.Context, eventID pgtype.UUID) (*sqlc.GetEventByIDRow, error) {
	event, err := r.queries.GetEventByID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *eventRepository) GetNearbyEvents(ctx context.Context, params *sqlc.GetNearbyEventsByStatusAndGenderParams) ([]sqlc.GetNearbyEventsByStatusAndGenderRow, error) {
	events, err := r.queries.GetNearbyEventsByStatusAndGender(ctx, *params)
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (r *eventRepository) GetUserEvents(ctx context.Context, params *sqlc.GetUserEventsParams) ([]sqlc.GetUserEventsRow, error) {
	events, err := r.queries.GetUserEvents(ctx, *params)
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (r *eventRepository) GetEventAttendeeStats(ctx context.Context, eventID pgtype.UUID) (*sqlc.GetEventAttendeeStatsRow, error) {
	stats, err := r.queries.GetEventAttendeeStats(ctx, eventID)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (r *eventRepository) UpdateEvent(ctx context.Context, params *sqlc.UpdateEventParams) (*sqlc.UpdateEventRow, error) {
	event, err := r.queries.UpdateEvent(ctx, *params)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *eventRepository) JoinEvent(ctx context.Context, params *sqlc.JoinEventParams) error {
	return r.queries.JoinEvent(ctx, *params)
}

func (r *eventRepository) LeaveEvent(ctx context.Context, params *sqlc.LeaveEventParams) error {
	return r.queries.LeaveEvent(ctx, *params)
}
