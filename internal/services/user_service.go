package services

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/mattiusz/based_backend/internal/gen/proto"
	"github.com/mattiusz/based_backend/internal/gen/sqlc"
	"github.com/mattiusz/based_backend/internal/repositories"
)

type userService struct {
	pb.UnimplementedUserServiceServer
	userRepo repositories.UserRepository
}

func NewUserService(repo repositories.UserRepository) pb.UserServiceServer {
	return &userService{userRepo: repo}
}

func (s *userService) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.User, error) {
	if req.Email == "" || req.DisplayName == "" {
		return nil, status.Error(codes.InvalidArgument, "email and display_name are required")
	}

	params := &sqlc.CreateUserParams{
		Name: req.DisplayName,
		// Add other required fields based on your schema
	}

	user, err := s.userRepo.CreateUser(ctx, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	return &pb.User{
		UserId:      user.UserID.Bytes[:],
		Email:       req.Email,
		DisplayName: user.Name,
		CreatedAt:   timestamppb.New(user.CreatedAt.Time),
	}, nil
}

func (s *userService) GetUserByID(ctx context.Context, req *pb.GetUserByIDRequest) (*pb.User, error) {
	if len(req.UserId) == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	var userID pgtype.UUID
	if err := userID.Scan(req.UserId); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user ID format: %v", err)
	}

	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}

	return &pb.User{
		UserId:      user.UserID.Bytes[:],
		DisplayName: user.Name,
		CreatedAt:   timestamppb.New(user.CreatedAt.Time),
	}, nil
}
