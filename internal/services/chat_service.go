package services

import (
	"context"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/mattiusz/based_backend/internal/gen/proto"
	"github.com/mattiusz/based_backend/internal/gen/sqlc"
	"github.com/mattiusz/based_backend/internal/interceptors"
	"github.com/mattiusz/based_backend/internal/repositories"
)

type chatService struct {
	pb.UnimplementedChatServiceServer
	chatRepo    repositories.ChatRepository
	subscribers sync.Map
	mu          sync.RWMutex
}

type subscriber struct {
	eventID   []byte
	userID    []byte
	messages  chan *pb.Message
	done      chan struct{}
	createdAt time.Time
}

func NewChatService(repo repositories.ChatRepository) pb.ChatServiceServer {
	service := &chatService{
		chatRepo: repo,
	}
	go service.cleanupInactiveSubscribers()
	return service
}

func (s *chatService) CreateMessage(ctx context.Context, req *pb.CreateMessageRequest) (*pb.Message, error) {
	authenticatedUserID, err := interceptors.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if len(req.EventId) == 0 || len(authenticatedUserID) == 0 || req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "event_id, user_id, and content are required")
	}

	params := &sqlc.CreateChatMessageParams{
		EventID: convertUUID(req.EventId),
		UserID:  convertUUID([]byte(authenticatedUserID)),
		Content: req.Content,
		Type:    convertPbMessageTypeToSQL(req.Type),
	}

	msg, err := s.chatRepo.CreateMessage(ctx, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create message: %v", err)
	}

	pbMsg := &pb.Message{
		MessageId: msg.MessageID.Bytes[:],
		EventId:   msg.EventID.Bytes[:],
		SenderId:  msg.UserID.Bytes[:],
		CreatedAt: timestamppb.New(msg.CreatedAt.Time),
		Content:   msg.Content,
		Status:    convertSQLMessageStatusToPB(msg.Status),
		Type:      convertSQLMessageTypeToPB(msg.Type),
	}

	s.broadcastMessage(pbMsg)
	return pbMsg, nil
}

func (s *chatService) GetMessages(ctx context.Context, req *pb.GetMessagesRequest) (*pb.GetMessagesResponse, error) {
	authenticatedUserID, err := interceptors.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if len(req.EventId) == 0 || len(authenticatedUserID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "event_id and user_id are required")
	}

	params := &sqlc.GetEventMessagesParams{
		EventID: convertUUID(req.EventId),
		UserID:  convertUUID([]byte(authenticatedUserID)),
	}

	messages, err := s.chatRepo.GetEventMessages(ctx, params)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "failed to get messages: %v", err)
	}

	response := &pb.GetMessagesResponse{
		Messages: make([]*pb.Message, len(messages)),
	}

	for i, msg := range messages {
		response.Messages[i] = &pb.Message{
			MessageId:     msg.MessageID.Bytes[:],
			EventId:       msg.EventID.Bytes[:],
			SenderId:      msg.UserID.Bytes[:],
			Content:       msg.Content,
			CreatedAt:     timestamppb.New(msg.CreatedAt.Time),
			Status:        convertSQLMessageStatusToPB(msg.Status),
			Type:          convertSQLMessageTypeToPB(msg.Type),
			NumberOfLikes: int32(msg.NumberOfLikes),
			IsLikedByUser: msg.IsLikedByUser,
		}
	}

	return response, nil
}

func (s *chatService) LikeMessage(ctx context.Context, req *pb.LikeMessageRequest) (*emptypb.Empty, error) {
	authenticatedUserID, err := interceptors.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if len(req.MessageId) == 0 || len(authenticatedUserID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "message_id and user_id are required")
	}

	params := &sqlc.LikeMessageParams{
		MessageID: convertUUID(req.MessageId),
		UserID:    convertUUID([]byte(authenticatedUserID)),
	}

	if err := s.chatRepo.LikeMessage(ctx, params); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to like message: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *chatService) UnlikeMessage(ctx context.Context, req *pb.UnlikeMessageRequest) (*emptypb.Empty, error) {
	authenticatedUserID, err := interceptors.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if len(req.MessageId) == 0 || len(authenticatedUserID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "message_id and user_id are required")
	}

	params := &sqlc.UnlikeMessageParams{
		MessageID: convertUUID(req.MessageId),
		UserID:    convertUUID([]byte(authenticatedUserID)),
	}

	if err := s.chatRepo.UnlikeMessage(ctx, params); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unlike message: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *chatService) DeleteMesssage(ctx context.Context, req *pb.DeleteMessageRequest) (*emptypb.Empty, error) {
	authenticatedUserID, err := interceptors.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if len(req.MessageId) == 0 || len(authenticatedUserID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "message_id and user_id are required")
	}

	params := &sqlc.DeleteChatMessageParams{
		MessageID: convertUUID(req.MessageId),
		UserID:    convertUUID([]byte(authenticatedUserID)),
	}

	if err := s.chatRepo.DeleteMesssage(ctx, params); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete message: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *chatService) StreamMessages(req *pb.StreamMessagesRequest, stream pb.ChatService_StreamMessagesServer) error {
	authenticatedUserID, err := interceptors.GetUserIDFromContext(stream.Context())
	if err != nil {
		return err
	}

	if len(req.EventId) == 0 || len(authenticatedUserID) == 0 {
		return status.Error(codes.InvalidArgument, "event_id and user_id are required")
	}

	sub := &subscriber{
		eventID:   req.EventId,
		userID:    []byte(authenticatedUserID),
		messages:  make(chan *pb.Message, 100),
		done:      make(chan struct{}),
		createdAt: time.Now(),
	}

	s.addSubscriber(req.EventId, sub)
	defer s.removeSubscriber(req.EventId, sub)

	params := &sqlc.GetEventMessagesParams{
		EventID: convertUUID(req.EventId),
		UserID:  convertUUID([]byte(authenticatedUserID)),
	}

	messages, err := s.chatRepo.GetEventMessages(stream.Context(), params)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get messages: %v", err)
	}

	for _, msg := range messages {
		pbMsg := &pb.Message{
			MessageId:     msg.MessageID.Bytes[:],
			EventId:       msg.EventID.Bytes[:],
			SenderId:      msg.UserID.Bytes[:],
			Content:       msg.Content,
			CreatedAt:     timestamppb.New(msg.CreatedAt.Time),
			Status:        convertSQLMessageStatusToPB(msg.Status),
			Type:          convertSQLMessageTypeToPB(msg.Type),
			NumberOfLikes: int32(msg.NumberOfLikes),
			IsLikedByUser: msg.IsLikedByUser,
		}
		if err := stream.Send(pbMsg); err != nil {
			return status.Errorf(codes.Internal, "failed to send message: %v", err)
		}
	}

	for {
		select {
		case <-stream.Context().Done():
			return status.Error(codes.Canceled, "client cancelled the stream")
		case <-sub.done:
			return status.Error(codes.Canceled, "server cancelled the stream")
		case msg := <-sub.messages:
			if err := stream.Send(msg); err != nil {
				return status.Errorf(codes.Internal, "failed to send message: %v", err)
			}
		}
	}
}

// Helper functions for managing subscribers

func (s *chatService) addSubscriber(eventID []byte, sub *subscriber) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var subs []*subscriber
	if val, ok := s.subscribers.Load(string(eventID)); ok {
		subs = val.([]*subscriber)
	}
	subs = append(subs, sub)
	s.subscribers.Store(string(eventID), subs)
}

func (s *chatService) removeSubscriber(eventID []byte, sub *subscriber) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if val, ok := s.subscribers.Load(string(eventID)); ok {
		subs := val.([]*subscriber)
		for i, existing := range subs {
			if existing == sub {
				close(sub.done)
				close(sub.messages)
				subs = append(subs[:i], subs[i+1:]...)
				break
			}
		}
		if len(subs) == 0 {
			s.subscribers.Delete(string(eventID))
		} else {
			s.subscribers.Store(string(eventID), subs)
		}
	}
}

func (s *chatService) broadcastMessage(msg *pb.Message) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var subs []*subscriber
	if val, ok := s.subscribers.Load(string(msg.EventId)); ok {
		subs = val.([]*subscriber)
	}

	for _, sub := range subs {
		select {
		case sub.messages <- msg:
			// Message sent successfully
		default:
			// Channel is full, skip this subscriber
			go s.removeSubscriber(msg.EventId, sub)
		}
	}
}

func (s *chatService) cleanupInactiveSubscribers() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		s.subscribers.Range(func(eventID, value interface{}) bool {
			subs := value.([]*subscriber)

			var activeSubscribers []*subscriber
			for _, sub := range subs {
				// Remove subscribers older than 10 minutes
				if now.Sub(sub.createdAt) < 10*time.Minute {
					activeSubscribers = append(activeSubscribers, sub)
				} else {
					close(sub.done)
					close(sub.messages)
				}
			}

			if len(activeSubscribers) == 0 {
				s.subscribers.Delete(eventID)
			} else {
				s.subscribers.Store(eventID, activeSubscribers)
			}
			return true
		})
	}
}

func convertPbMessageTypeToSQL(status pb.MessageType) sqlc.MessageCategoryType {
	switch status {
	case pb.MessageType_MESSAGE_TYPE_TEXT:
		return sqlc.MessageCategoryTypeText
	case pb.MessageType_MESSAGE_TYPE_LOCATION:
		return sqlc.MessageCategoryTypeLocation
	default:
		return sqlc.MessageCategoryTypeUnspecified
	}
}

func convertSQLMessageTypeToPB(status sqlc.MessageCategoryType) pb.MessageType {
	switch status {
	case sqlc.MessageCategoryTypeText:
		return pb.MessageType_MESSAGE_TYPE_TEXT
	case sqlc.MessageCategoryTypeLocation:
		return pb.MessageType_MESSAGE_TYPE_LOCATION
	default:
		return pb.MessageType_MESSAGE_TYPE_UNSPECIFIED
	}
}

func convertPbMessageStatusToSQL(status pb.MessageStatus) sqlc.MessageStatusType {
	switch status {
	case pb.MessageStatus_MESSAGE_STATUS_SENT:
		return sqlc.MessageStatusTypeSent
	case pb.MessageStatus_MESSAGE_STATUS_DELETED:
		return sqlc.MessageStatusTypeDeleted
	case pb.MessageStatus_MESSAGE_STATUS_EDITED:
		return sqlc.MessageStatusTypeEdited
	default:
		return sqlc.MessageStatusTypeUnspecified
	}
}

func convertSQLMessageStatusToPB(status sqlc.MessageStatusType) pb.MessageStatus {
	switch status {
	case sqlc.MessageStatusTypeSent:
		return pb.MessageStatus_MESSAGE_STATUS_SENT
	case sqlc.MessageStatusTypeDeleted:
		return pb.MessageStatus_MESSAGE_STATUS_DELETED
	case sqlc.MessageStatusTypeEdited:
		return pb.MessageStatus_MESSAGE_STATUS_EDITED
	default:
		return pb.MessageStatus_MESSAGE_STATUS_UNSPECIFIED
	}
}
