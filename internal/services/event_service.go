package services

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/mattiusz/based_backend/internal/gen/proto"
	"github.com/mattiusz/based_backend/internal/gen/sqlc"
	"github.com/mattiusz/based_backend/internal/interceptors"
	"github.com/mattiusz/based_backend/internal/repositories"
)

type eventService struct {
	pb.UnimplementedEventServiceServer
	eventRepo repositories.EventRepository
}

func NewEventService(repo repositories.EventRepository) pb.EventServiceServer {
	return &eventService{eventRepo: repo}
}

func (s *eventService) CreateEvent(ctx context.Context, req *pb.CreateEventRequest) (*pb.Event, error) {
	authenticatedUserID, err := interceptors.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := validateCreateEventRequest(req); err != nil {
		return nil, err
	}

	categories := make([]sqlc.EventCategoryType, len(req.Categories))
	for i, category := range req.Categories {
		categories[i] = convertPBCategoryToSQL(category)
	}

	params := &sqlc.CreateEventParams{
		CreatorID:     convertUUID([]byte(authenticatedUserID)),
		Name:          req.Name,
		StMakepoint:   req.Location.Latitude,
		StMakepoint_2: req.Location.Longitude,
		Datetime:      pgtype.Timestamptz{Time: req.Datetime.AsTime(), Valid: true},
		MaxAttendees:  req.MaxAttendees,
		Status:        sqlc.EventStatusTypeUpcoming,
		Thumbnail:     req.Thumbnail,
		Categories:    categories,
		Venue:         pgtype.Text{String: req.Venue, Valid: true},
		Description:   pgtype.Text{String: req.Description, Valid: true},
		AgeRangeMin:   pgtype.Int4{Int32: req.AgeRangeMin, Valid: true},
		AgeRangeMax:   pgtype.Int4{Int32: req.AgeRangeMax, Valid: true},
		AllowFemale:   req.AllowFemale,
		AllowMale:     req.AllowMale,
		AllowDiverse:  req.AllowDiverse,
	}

	createdEvent, err := s.eventRepo.CreateEvent(ctx, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create event: %v", err)
	}

	joinParams := &sqlc.JoinEventParams{
		EventID: createdEvent.EventID,
		UserID:  convertUUID([]byte(authenticatedUserID)),
	}

	if err := s.eventRepo.JoinEvent(ctx, joinParams); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to join event: %v", err)
	}

	return convertCreateEventResponseToEvent(createdEvent), nil
}

func (s *eventService) JoinEvent(ctx context.Context, req *pb.JoinEventRequest) (*emptypb.Empty, error) {
	authenticatedUserID, err := interceptors.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	params := &sqlc.JoinEventParams{
		EventID: convertUUID(req.EventId),
		UserID:  convertUUID([]byte(authenticatedUserID)),
	}

	if err := s.eventRepo.JoinEvent(ctx, params); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to join event: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *eventService) LeaveEvent(ctx context.Context, req *pb.LeaveEventRequest) (*emptypb.Empty, error) {
	authenticatedUserID, err := interceptors.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	params := &sqlc.LeaveEventParams{
		EventID: convertUUID(req.EventId),
		UserID:  convertUUID([]byte(authenticatedUserID)),
	}

	if err := s.eventRepo.LeaveEvent(ctx, params); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to leave event: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *eventService) DeleteEvent(ctx context.Context, req *pb.DeleteEventRequest) (*emptypb.Empty, error) {
	authenticatedUserID, err := interceptors.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	params := &sqlc.DeleteEventParams{
		EventID:   convertUUID(req.EventId),
		CreatorID: convertUUID([]byte(authenticatedUserID)),
	}

	if err := s.eventRepo.DeleteEvent(ctx, params); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete event: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func validateCreateEventRequest(req *pb.CreateEventRequest) error {
	if req.Name == "" {
		return status.Error(codes.InvalidArgument, "name is required")
	}
	if req.Location == nil {
		return status.Error(codes.InvalidArgument, "location is required")
	}
	if req.Datetime == nil {
		return status.Error(codes.InvalidArgument, "event_datetime is required")
	}
	if req.MaxAttendees <= 0 {
		return status.Error(codes.InvalidArgument, "max_attendees must be positive")
	}
	return nil
}

func convertCreateEventResponseToEvent(event *sqlc.CreateEventRow) *pb.Event {
	categories := make([]pb.EventCategory, len(event.Categories))
	for i, category := range event.Categories {
		categories[i] = convertSQLCategoryToPB(category)
	}

	return &pb.Event{
		EventId:   event.EventID.Bytes[:],
		CreatorId: event.CreatorID.Bytes[:],
		Name:      event.Name,
		Location: &pb.Location{
			Latitude:  event.Latitude.(float64),
			Longitude: event.Longitude.(float64),
		},
		Datetime:          timestamppb.New(event.Datetime.Time),
		MaxAttendees:      event.MaxAttendees,
		Venue:             &event.Venue.String,
		Description:       &event.Description.String,
		Thumbnail:         event.Thumbnail,
		Status:            convertSQLStatusToPB(event.Status),
		AgeRangeMin:       event.AgeRangeMin.Int32,
		AgeRangeMax:       event.AgeRangeMax.Int32,
		AllowFemale:       event.AllowFemale,
		AllowMale:         event.AllowMale,
		AllowDiverse:      event.AllowDiverse,
		Categories:        categories,
		NumberOfComments:  0, // New event has no comments
		NumberOfAttendees: 1, // Creator is the first attendee
	}
}

func convertGetUserEventsResponseToEvent(events []sqlc.GetUserEventsRow) *pb.GetUserEventsResponse {
	response := &pb.GetUserEventsResponse{
		Events: make([]*pb.Event, len(events)),
	}

	for i, event := range events {
		categories := make([]pb.EventCategory, len(event.Categories))
		for i, category := range event.Categories {
			categories[i] = convertSQLCategoryToPB(category)
		}

		response.Events[i] = &pb.Event{
			EventId:   event.EventID.Bytes[:],
			CreatorId: event.CreatorID.Bytes[:],
			Name:      event.Name,
			Location: &pb.Location{
				Latitude:  event.Latitude.(float64),
				Longitude: event.Longitude.(float64),
			},
			Datetime:          timestamppb.New(event.Datetime.Time),
			MaxAttendees:      event.MaxAttendees,
			Venue:             &event.Venue.String,
			Status:            convertSQLStatusToPB(event.Status),
			AgeRangeMin:       event.AgeRangeMin.Int32,
			AgeRangeMax:       event.AgeRangeMax.Int32,
			AllowFemale:       event.AllowFemale,
			AllowMale:         event.AllowMale,
			AllowDiverse:      event.AllowDiverse,
			NumberOfAttendees: event.NumberOfAttendees,
			Categories:        categories,
		}
	}

	return response
}

func convertGetEventByIdResponseToEvent(event sqlc.GetEventByIDRow) *pb.Event {
	categories := make([]pb.EventCategory, len(event.Categories))
	for i, category := range event.Categories {
		categories[i] = convertSQLCategoryToPB(category)
	}

	return &pb.Event{
		EventId:   event.EventID.Bytes[:],
		CreatorId: event.CreatorID.Bytes[:],
		Name:      event.Name,
		Location: &pb.Location{
			Latitude:  event.Latitude.(float64),
			Longitude: event.Longitude.(float64),
		},
		Datetime:          timestamppb.New(event.Datetime.Time),
		MaxAttendees:      event.MaxAttendees,
		Venue:             &event.Venue.String,
		Description:       &event.Description.String,
		Thumbnail:         event.Thumbnail,
		Status:            convertSQLStatusToPB(event.Status),
		AgeRangeMin:       event.AgeRangeMin.Int32,
		AgeRangeMax:       event.AgeRangeMax.Int32,
		AllowFemale:       event.AllowFemale,
		AllowMale:         event.AllowMale,
		AllowDiverse:      event.AllowDiverse,
		Categories:        categories,
		NumberOfComments:  event.NumberOfComments,
		NumberOfAttendees: event.NumberOfAttendees,
	}
}

func convertGetNearbyEventsResponseToProto(events []sqlc.GetNearbyEventsByStatusAndGenderRow) *pb.GetNearbyEventsResponse {
	response := &pb.GetNearbyEventsResponse{
		Events: make([]*pb.EventWithDistance, len(events)),
	}

	for i, event := range events {
		categories := make([]pb.EventCategory, len(event.Categories))
		for i, category := range event.Categories {
			categories[i] = convertSQLCategoryToPB(category)
		}

		response.Events[i] = &pb.EventWithDistance{
			Event: &pb.Event{
				EventId:   event.EventID.Bytes[:],
				CreatorId: event.CreatorID.Bytes[:],
				Name:      event.Name,
				Location: &pb.Location{
					Latitude:  event.Latitude.(float64),
					Longitude: event.Longitude.(float64),
				},
				Datetime:          timestamppb.New(event.Datetime.Time),
				MaxAttendees:      event.MaxAttendees,
				Venue:             &event.Venue.String,
				Status:            convertSQLStatusToPB(event.Status),
				AgeRangeMin:       event.AgeRangeMin.Int32,
				AgeRangeMax:       event.AgeRangeMax.Int32,
				AllowFemale:       event.AllowFemale,
				AllowMale:         event.AllowMale,
				AllowDiverse:      event.AllowDiverse,
				NumberOfAttendees: event.NumberOfAttendees,
				Categories:        categories,
			},
			DistanceMeters: event.DistanceMeters.(float64),
		}
	}

	return response
}

func convertUUID(uuid []byte) pgtype.UUID {
	var bytes [16]byte
	copy(bytes[:], uuid)
	return pgtype.UUID{Bytes: bytes, Valid: true}
}

func convertPBCategoryToSQL(category pb.EventCategory) sqlc.EventCategoryType {
	switch category {
	case pb.EventCategory_EVENT_CATEGORY_ART:
		return sqlc.EventCategoryTypeArt
	case pb.EventCategory_EVENT_CATEGORY_SPORTS:
		return sqlc.EventCategoryTypeSports
	case pb.EventCategory_EVENT_CATEGORY_MUSIC_AND_MOVIES:
		return sqlc.EventCategoryTypeMusicAndMovies
	case pb.EventCategory_EVENT_CATEGORY_FOOD_AND_DRINKS:
		return sqlc.EventCategoryTypeFoodAndDrinks
	case pb.EventCategory_EVENT_CATEGORY_PARTY_AND_GAMES:
		return sqlc.EventCategoryTypePartyAndGames
	case pb.EventCategory_EVENT_CATEGORY_BUSINESS:
		return sqlc.EventCategoryTypeBusiness
	case pb.EventCategory_EVENT_CATEGORY_NATURE:
		return sqlc.EventCategoryTypeNature
	case pb.EventCategory_EVENT_CATEGORY_TECHNOLOGY:
		return sqlc.EventCategoryTypeTechnology
	case pb.EventCategory_EVENT_CATEGORY_TRAVEL:
		return sqlc.EventCategoryTypeTravel
	case pb.EventCategory_EVENT_CATEGORY_EDUCATION:
		return sqlc.EventCategoryTypeEducation
	case pb.EventCategory_EVENT_CATEGORY_CHARITY:
		return sqlc.EventCategoryTypeCharity
	case pb.EventCategory_EVENT_CATEGORY_OTHER:
		return sqlc.EventCategoryTypeOther
	default:
		return sqlc.EventCategoryTypeUnspecified
	}
}

func convertSQLCategoryToPB(category sqlc.EventCategoryType) pb.EventCategory {
	switch category {
	case sqlc.EventCategoryTypeArt:
		return pb.EventCategory_EVENT_CATEGORY_ART
	case sqlc.EventCategoryTypeSports:
		return pb.EventCategory_EVENT_CATEGORY_SPORTS
	case sqlc.EventCategoryTypeMusicAndMovies:
		return pb.EventCategory_EVENT_CATEGORY_MUSIC_AND_MOVIES
	case sqlc.EventCategoryTypeFoodAndDrinks:
		return pb.EventCategory_EVENT_CATEGORY_FOOD_AND_DRINKS
	case sqlc.EventCategoryTypePartyAndGames:
		return pb.EventCategory_EVENT_CATEGORY_PARTY_AND_GAMES
	case sqlc.EventCategoryTypeBusiness:
		return pb.EventCategory_EVENT_CATEGORY_BUSINESS
	case sqlc.EventCategoryTypeNature:
		return pb.EventCategory_EVENT_CATEGORY_NATURE
	case sqlc.EventCategoryTypeTechnology:
		return pb.EventCategory_EVENT_CATEGORY_TECHNOLOGY
	case sqlc.EventCategoryTypeTravel:
		return pb.EventCategory_EVENT_CATEGORY_TRAVEL
	case sqlc.EventCategoryTypeEducation:
		return pb.EventCategory_EVENT_CATEGORY_EDUCATION
	case sqlc.EventCategoryTypeCharity:
		return pb.EventCategory_EVENT_CATEGORY_CHARITY
	case sqlc.EventCategoryTypeOther:
		return pb.EventCategory_EVENT_CATEGORY_OTHER
	default:
		return pb.EventCategory_EVENT_CATEGORY_UNSPECIFIED
	}
}

func convertPbStatusToSQL(status pb.EventStatus) sqlc.EventStatusType {
	switch status {
	case pb.EventStatus_EVENT_STATUS_UPCOMING:
		return sqlc.EventStatusTypeUpcoming
	case pb.EventStatus_EVENT_STATUS_ONGOING:
		return sqlc.EventStatusTypeOngoing
	case pb.EventStatus_EVENT_STATUS_CANCELLED:
		return sqlc.EventStatusTypeCancelled
	case pb.EventStatus_EVENT_STATUS_COMPLETED:
		return sqlc.EventStatusTypeCompleted
	case pb.EventStatus_EVENT_STATUS_RESCHEDULED:
		return sqlc.EventStatusTypeRescheduled
	default:
		return sqlc.EventStatusTypeUnspecified
	}
}

func convertSQLStatusToPB(status sqlc.EventStatusType) pb.EventStatus {
	switch status {
	case sqlc.EventStatusTypeUpcoming:
		return pb.EventStatus_EVENT_STATUS_UPCOMING
	case sqlc.EventStatusTypeOngoing:
		return pb.EventStatus_EVENT_STATUS_ONGOING
	case sqlc.EventStatusTypeCancelled:
		return pb.EventStatus_EVENT_STATUS_CANCELLED
	case sqlc.EventStatusTypeCompleted:
		return pb.EventStatus_EVENT_STATUS_COMPLETED
	case sqlc.EventStatusTypeRescheduled:
		return pb.EventStatus_EVENT_STATUS_RESCHEDULED
	default:
		return pb.EventStatus_EVENT_STATUS_UNSPECIFIED
	}
}
