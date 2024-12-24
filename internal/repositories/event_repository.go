package repositories

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mattiusz/based_backend/internal/gen/sqlc"
)

type EventRepository interface {
	CreateEvent(ctx context.Context, req *sqlc.CreateEventParams) (*sqlc.CreateEventRow, error)
	GetEventByID(ctx context.Context, eventID pgtype.UUID) (*sqlc.GetEventByIDRow, error)
	GetNearbyEvents(ctx context.Context, params *sqlc.GetNearbyEventsParams) ([]sqlc.GetNearbyEventsRow, error)
	GetUserEvents(ctx context.Context, userID pgtype.UUID) ([]sqlc.GetUserEventsRow, error)
	SearchEvents(ctx context.Context, params *sqlc.SearchEventsParams) ([]sqlc.SearchEventsRow, error)
	GetEventAttendeeStats(ctx context.Context, eventID pgtype.UUID) (*sqlc.GetEventAttendeeStatsRow, error)
	JoinEvent(ctx context.Context, params *sqlc.JoinEventParams) error
	LeaveEvent(ctx context.Context, params *sqlc.LeaveEventParams) error
	UpdateEventStatus(ctx context.Context, params *sqlc.UpdateEventStatusParams) (*sqlc.Event, error)
	AddEventCategory(ctx context.Context, params *sqlc.AddEventCategoryParams) error
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

func (r *eventRepository) GetNearbyEvents(ctx context.Context, params *sqlc.GetNearbyEventsParams) ([]sqlc.GetNearbyEventsRow, error) {
	events, err := r.queries.GetNearbyEvents(ctx, *params)
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (r *eventRepository) GetUserEvents(ctx context.Context, userID pgtype.UUID) ([]sqlc.GetUserEventsRow, error) {
	events, err := r.queries.GetUserEvents(ctx, userID)
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (r *eventRepository) SearchEvents(ctx context.Context, params *sqlc.SearchEventsParams) ([]sqlc.SearchEventsRow, error) {
	events, err := r.queries.SearchEvents(ctx, *params)
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

func (r *eventRepository) JoinEvent(ctx context.Context, params *sqlc.JoinEventParams) error {
	return r.queries.JoinEvent(ctx, *params)
}

func (r *eventRepository) LeaveEvent(ctx context.Context, params *sqlc.LeaveEventParams) error {
	return r.queries.LeaveEvent(ctx, *params)
}

func (r *eventRepository) UpdateEventStatus(ctx context.Context, params *sqlc.UpdateEventStatusParams) (*sqlc.Event, error) {
	event, err := r.queries.UpdateEventStatus(ctx, *params)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *eventRepository) AddEventCategory(ctx context.Context, params *sqlc.AddEventCategoryParams) error {
	return r.queries.AddEventCategory(ctx, *params)
}
