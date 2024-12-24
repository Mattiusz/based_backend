package services

import (
	"context"
	"fmt"

	"github.com/cridenour/go-postgis"
	"github.com/jackc/pgx/v5/pgtype"
	pb "github.com/mattiusz/based_backend/internal/gen/proto"
	"github.com/mattiusz/based_backend/internal/gen/sqlc"
	"github.com/mattiusz/based_backend/internal/repositories"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type eventService struct {
	pb.UnimplementedEventServiceServer
	eventRepo repositories.EventRepository
}

func NewEventService(repo repositories.EventRepository) pb.EventServiceServer {
	return &eventService{
		eventRepo: repo,
	}
}

func (s *eventService) CreateEvent(ctx context.Context, req *pb.CreateEventRequest) (*pb.Event, error) {
	if err := validateCreateEventRequest(req); err != nil {
		return nil, err
	}

	params := &sqlc.CreateEventParams{
		CreatorID:             convertUUID(req.CreatorId),
		Name:                  req.Name,
		StMakepoint:           req.Location.Longitude,
		StMakepoint_2:         req.Location.Latitude,
		EventDatetime:         pgtype.Timestamptz{Time: req.EventDatetime.AsTime(), Valid: true},
		TimezoneOffsetMinutes: req.TimezoneOffsetMinutes,
		MaxAttendees:          req.MaxAttendees,
		Venue:                 pgtype.Text{String: req.Venue, Valid: true},
		Description:           pgtype.Text{String: req.Description, Valid: true},
		AgeRangeMin:           pgtype.Int4{Int32: req.AgeRangeMin, Valid: true},
		AgeRangeMax:           pgtype.Int4{Int32: req.AgeRangeMax, Valid: true},
		AllowFemale:           req.AllowFemale,
		AllowMale:             req.AllowMale,
		AllowDiverse:          req.AllowDiverse,
	}

	event, err := s.eventRepo.CreateEvent(ctx, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create event: %v", err)
	}

	// Add categories if provided
	for _, category := range req.Categories {
		categoryParams := &sqlc.AddEventCategoryParams{
			EventID:  event.EventID,
			Category: sqlc.EventCategoryType(category.String()),
		}
		if err := s.eventRepo.AddEventCategory(ctx, categoryParams); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to add category: %v", err)
		}
	}

	return convertEventToProto(event), nil
}

func (s *eventService) GetEventByID(ctx context.Context, req *pb.GetEventByIDRequest) (*pb.EventDetails, error) {
	if len(req.EventId) == 0 {
		return nil, status.Error(codes.InvalidArgument, "event_id is required")
	}

	eventID := convertUUID(req.EventId)
	event, err := s.eventRepo.GetEventByID(ctx, eventID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get event: %v", err)
	}

	return convertEventDetailsToProto(*event), nil
}

func (s *eventService) GetNearbyEvents(ctx context.Context, req *pb.GetNearbyEventsRequest) (*pb.GetNearbyEventsResponse, error) {
	if req.Location == nil {
		return nil, status.Error(codes.InvalidArgument, "location is required")
	}

	params := &sqlc.GetNearbyEventsParams{
		StMakepoint:   req.Location.Longitude,
		StMakepoint_2: req.Location.Latitude,
		StDwithin:     req.RadiusMeters,
		Limit:         req.Limit,
	}

	events, err := s.eventRepo.GetNearbyEvents(ctx, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get nearby events: %v", err)
	}

	return convertNearbyEventsToProto(events), nil
}

func (s *eventService) GetUserEvents(ctx context.Context, req *pb.GetUserEventsRequest) (*pb.GetUserEventsResponse, error) {
	if len(req.UserId) == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	userID := convertUUID(req.UserId)
	events, err := s.eventRepo.GetUserEvents(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user events: %v", err)
	}

	response := &pb.GetUserEventsResponse{
		Events: make([]*pb.Event, len(events)),
	}
	for i, event := range events {
		response.Events[i] = convertEventToProto(&event)
	}

	return response, nil
}

func (s *eventService) SearchEvents(ctx context.Context, req *pb.SearchEventsRequest) (*pb.SearchEventsResponse, error) {
	if req.Query == "" {
		return nil, status.Error(codes.InvalidArgument, "query is required")
	}

	params := &sqlc.SearchEventsParams{
		Lower:  fmt.Sprintf("%%%s%%", req.Query),
		Limit:  req.Limit,
		Offset: req.Offset,
	}

	events, err := s.eventRepo.SearchEvents(ctx, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to search events: %v", err)
	}

	response := &pb.SearchEventsResponse{
		Events: make([]*pb.EventDetails, len(events)),
	}
	for i, event := range events {
		response.Events[i] = convertSearchEventToProto(event)
	}

	return response, nil
}

func (s *eventService) JoinEvent(ctx context.Context, req *pb.JoinEventRequest) (*emptypb.Empty, error) {
	if len(req.EventId) == 0 || len(req.UserId) == 0 {
		return nil, status.Error(codes.InvalidArgument, "event_id and user_id are required")
	}

	params := &sqlc.JoinEventParams{
		EventID: convertUUID(req.EventId),
		UserID:  convertUUID(req.UserId),
	}

	if err := s.eventRepo.JoinEvent(ctx, params); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to join event: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *eventService) LeaveEvent(ctx context.Context, req *pb.LeaveEventRequest) (*emptypb.Empty, error) {
	if len(req.EventId) == 0 || len(req.UserId) == 0 {
		return nil, status.Error(codes.InvalidArgument, "event_id and user_id are required")
	}

	params := &sqlc.LeaveEventParams{
		EventID: convertUUID(req.EventId),
		UserID:  convertUUID(req.UserId),
	}

	if err := s.eventRepo.LeaveEvent(ctx, params); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to leave event: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *eventService) UpdateEventStatus(ctx context.Context, req *pb.UpdateEventStatusRequest) (*pb.Event, error) {
	if len(req.EventId) == 0 {
		return nil, status.Error(codes.InvalidArgument, "event_id is required")
	}

	params := &sqlc.UpdateEventStatusParams{
		EventID: convertUUID(req.EventId),
		Status:  sqlc.EventStatusType(req.Status),
	}

	event, err := s.eventRepo.UpdateEventStatus(ctx, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update event status: %v", err)
	}

	return convertEventToProto(event), nil
}

func (s *eventService) GetEventAttendeeStats(ctx context.Context, req *pb.GetEventAttendeeStatsRequest) (*pb.EventAttendeeStats, error) {
	if len(req.EventId) == 0 {
		return nil, status.Error(codes.InvalidArgument, "event_id is required")
	}

	eventId := convertUUID(req.EventId)
	stats, err := s.eventRepo.GetEventAttendeeStats(ctx, eventId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get attendee stats: %v", err)
	}

	return &pb.EventAttendeeStats{
		FemaleCount:  stats.FemaleCount,
		MaleCount:    stats.MaleCount,
		DiverseCount: stats.DiverseCount,
	}, nil
}

func (s *eventService) AddEventCategory(ctx context.Context, req *pb.AddEventCategoryRequest) (*emptypb.Empty, error) {
	if len(req.EventId) == 0 {
		return nil, status.Error(codes.InvalidArgument, "event_id is required")
	}

	params := &sqlc.AddEventCategoryParams{
		EventID:  convertUUID(req.EventId),
		Category: sqlc.EventCategoryType(req.Category.String()),
	}

	if err := s.eventRepo.AddEventCategory(ctx, params); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add category: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// Helper functions for request validation and data conversion

func validateCreateEventRequest(req *pb.CreateEventRequest) error {
	if len(req.CreatorId) == 0 {
		return status.Error(codes.InvalidArgument, "creator_id is required")
	}
	if req.Name == "" {
		return status.Error(codes.InvalidArgument, "name is required")
	}
	if req.Location == nil {
		return status.Error(codes.InvalidArgument, "location is required")
	}
	if req.EventDatetime == nil {
		return status.Error(codes.InvalidArgument, "event_datetime is required")
	}
	if req.MaxAttendees <= 0 {
		return status.Error(codes.InvalidArgument, "max_attendees must be positive")
	}
	return nil
}

func convertEventToProto(event *sqlc.Event) *pb.Event {
	return &pb.Event{
		EventId:   event.EventID.Bytes[:],
		CreatorId: event.CreatorID.Bytes[:],
		Name:      event.Name,
		Location: &pb.Location{
			Latitude:  event.Location.Y,
			Longitude: event.Location.X,
		},
		EventDatetime:         timestamppb.New(event.EventDatetime.Time),
		TimezoneOffsetMinutes: event.TimezoneOffsetMinutes,
		MaxAttendees:          event.MaxAttendees,
		Venue:                 event.Venue.String,
		Description:           &event.Description.String,
		Thumbnail:             event.Thumbnail,
		Status:                pb.EventStatus(pb.EventStatus_value[string(event.Status)]),
		AgeRangeMin:           event.AgeRangeMin.Int32,
		AgeRangeMax:           event.AgeRangeMax.Int32,
		AllowFemale:           event.AllowFemale,
		AllowMale:             event.AllowMale,
		AllowDiverse:          event.AllowDiverse,
		CreatedAt:             timestamppb.New(event.CreatedAt.Time),
	}
}

func convertEventDetailsToProto(event sqlc.GetEventByIDRow) *pb.EventDetails {
	return &pb.EventDetails{
		Event: &pb.Event{
			EventId:   event.EventID.Bytes[:],
			CreatorId: event.CreatorID.Bytes[:],
			Name:      event.Name,
			Location: &pb.Location{
				Latitude:  event.Location.Y,
				Longitude: event.Location.X,
			},
			EventDatetime:         timestamppb.New(event.EventDatetime.Time),
			TimezoneOffsetMinutes: event.TimezoneOffsetMinutes,
			MaxAttendees:          event.MaxAttendees,
			Venue:                 event.Venue.String,
			Description:           &event.Description.String,
			Thumbnail:             event.Thumbnail,
			Status:                pb.EventStatus(pb.EventStatus_value[string(event.Status)]),
			AgeRangeMin:           event.AgeRangeMin.Int32,
			AgeRangeMax:           event.AgeRangeMax.Int32,
			AllowFemale:           event.AllowFemale,
			AllowMale:             event.AllowMale,
			AllowDiverse:          event.AllowDiverse,
			CreatedAt:             timestamppb.New(event.CreatedAt.Time),
		},
		NumberOfComments:  event.NumberOfComments,
		NumberOfAttendees: event.NumberOfAttendees,
	}
}

func convertNearbyEventsToProto(events []sqlc.GetNearbyEventsRow) *pb.GetNearbyEventsResponse {
	response := &pb.GetNearbyEventsResponse{
		Events: make([]*pb.EventWithDistance, len(events)),
	}

	for i, event := range events {
		response.Events[i] = &pb.EventWithDistance{
			Event: &pb.Event{
				EventId:   event.EventID.Bytes[:],
				CreatorId: event.CreatorID.Bytes[:],
				Name:      event.Name,
				Location: &pb.Location{
					Latitude:  event.Location.Y,
					Longitude: event.Location.X,
				},
				EventDatetime:         timestamppb.New(event.EventDatetime.Time),
				TimezoneOffsetMinutes: event.TimezoneOffsetMinutes,
				MaxAttendees:          event.MaxAttendees,
				Venue:                 event.Venue.String,
				Status:                pb.EventStatus(pb.EventStatus_value[string(event.Status)]),
				AgeRangeMin:           event.AgeRangeMin.Int32,
				AgeRangeMax:           event.AgeRangeMax.Int32,
				AllowFemale:           event.AllowFemale,
				AllowMale:             event.AllowMale,
				AllowDiverse:          event.AllowDiverse,
				CreatedAt:             timestamppb.New(event.CreatedAt.Time),
			},
			DistanceMeters: event.DistanceMeters.(float64),
		}
	}

	return response
}

func convertSearchEventToProto(event sqlc.SearchEventsRow) *pb.EventDetails {
	var categories []string
	// Assuming categories is a JSON array stored as bytes
	if event.Categories != nil {
		// In a real implementation, you would properly parse the JSON array
		// This is a placeholder for the actual JSON parsing logic
		// categories = parseJSONCategories(event.Categories)
	}

	return &pb.EventDetails{
		Event: &pb.Event{
			EventId:   event.EventID.Bytes[:],
			CreatorId: event.CreatorID.Bytes[:],
			Name:      event.Name,
			Location: &pb.Location{
				Latitude:  event.Location.Y,
				Longitude: event.Location.X,
			},
			EventDatetime:         timestamppb.New(event.EventDatetime.Time),
			TimezoneOffsetMinutes: event.TimezoneOffsetMinutes,
			MaxAttendees:          event.MaxAttendees,
			Venue:                 event.Venue.String,
			Description:           &event.Description.String,
			Thumbnail:             event.Thumbnail,
			Status:                pb.EventStatus(pb.EventStatus_value[string(event.Status)]),
			AgeRangeMin:           event.AgeRangeMin.Int32,
			AgeRangeMax:           event.AgeRangeMax.Int32,
			AllowFemale:           event.AllowFemale,
			AllowMale:             event.AllowMale,
			AllowDiverse:          event.AllowDiverse,
			CreatedAt:             timestamppb.New(event.CreatedAt.Time),
		},
		Categories: categories,
	}
}

func convertLocationToPoint(location *pb.Location) postgis.Point {
	return postgis.Point{
		X: location.Longitude,
		Y: location.Latitude,
	}
}

func convertUUID(uuid []byte) pgtype.UUID {
	var bytes [16]byte
	copy(bytes[:], uuid)
	return pgtype.UUID{Bytes: bytes, Valid: true}
}
