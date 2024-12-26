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

func convertPBGenderToSQL(gender pb.GENDER) (sqlc.GenderType, error) {
	switch gender {
	case pb.GENDER_MALE:
		return sqlc.GenderTypeMale, nil
	case pb.GENDER_FEMALE:
		return sqlc.GenderTypeFemale, nil
	case pb.GENDER_DIVERSE:
		return sqlc.GenderTypeDiverse, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "invalid gender value")
	}
}

func convertSQLGenderToPB(gender sqlc.GenderType) (pb.GENDER, error) {
	switch gender {
	case sqlc.GenderTypeMale:
		return pb.GENDER_MALE, nil
	case sqlc.GenderTypeFemale:
		return pb.GENDER_FEMALE, nil
	case sqlc.GenderTypeDiverse:
		return pb.GENDER_DIVERSE, nil
	default:
		return pb.GENDER_DIVERSE, status.Errorf(codes.InvalidArgument, "invalid gender value")
	}
}

func (s *userService) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.User, error) {
	if req.DisplayName == "" || req.Birthday == nil {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	gender, err := convertPBGenderToSQL(req.Gender)
	if err != nil {
		return nil, err
	}

	params := &sqlc.CreateUserParams{
		Name:     req.DisplayName,
		Birthday: pgtype.Date{Time: req.Birthday.AsTime(), Valid: req.Birthday != nil},
		Gender:   gender,
	}

	user, err := s.userRepo.CreateUser(ctx, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	gender_pb, err := convertSQLGenderToPB(user.Gender)
	if err != nil {
		return nil, err
	}

	return &pb.User{
		UserId:      user.UserID.Bytes[:],
		DisplayName: user.Name,
		Birthday:    timestamppb.New(user.Birthday.Time),
		Gender:      gender_pb,
	}, nil
}

func (s *userService) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.User, error) {
	if len(req.UserId) == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	userID := pgtype.UUID{
		Bytes: [16]byte(req.UserId),
		Valid: true,
	}

	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}

	params := &sqlc.UpdateUserParams{
		UserID:   userID,
		Name:     req.DisplayName,
		Birthday: pgtype.Date{Time: req.Birthday.AsTime(), Valid: req.Birthday != nil},
		Gender:   user.Gender,
	}

	updated_user, err := s.userRepo.UpdateUser(ctx, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update user: %v", err)
	}

	gender_pb, err := convertSQLGenderToPB(updated_user.Gender)
	if err != nil {
		return nil, err
	}
	return &pb.User{
		UserId:      updated_user.UserID.Bytes[:],
		DisplayName: updated_user.Name,
		Birthday:    timestamppb.New(updated_user.Birthday.Time),
		Gender:      gender_pb,
	}, nil
}

func (s *userService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
	if len(req.UserId) == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	userID := pgtype.UUID{
		Bytes: [16]byte(req.UserId),
		Valid: true,
	}

	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}

	gender, err := convertSQLGenderToPB(user.Gender)
	if err != nil {
		return nil, err
	}

	return &pb.User{
		UserId:      user.UserID.Bytes[:],
		DisplayName: user.Name,
		Birthday:    timestamppb.New(user.Birthday.Time),
		Gender:      gender,
	}, nil
}
