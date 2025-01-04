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
	if req.DisplayName == "" || req.Birthday == nil {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	gender := convertPBGenderToSQL(req.Gender)

	params := &sqlc.CreateUserParams{
		Name:     req.DisplayName,
		Birthday: pgtype.Date{Time: req.Birthday.AsTime(), Valid: req.Birthday != nil},
		Gender:   gender,
	}

	user, err := s.userRepo.CreateUser(ctx, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	return &pb.User{
		UserId:      user.UserID.Bytes[:],
		DisplayName: user.Name,
		Birthday:    timestamppb.New(user.Birthday.Time),
		Gender:      convertSQLGenderToPB(user.Gender),
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

	return &pb.User{
		UserId:      updated_user.UserID.Bytes[:],
		DisplayName: updated_user.Name,
		Birthday:    timestamppb.New(updated_user.Birthday.Time),
		Gender:      convertSQLGenderToPB(updated_user.Gender),
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

	return &pb.User{
		UserId:      user.UserID.Bytes[:],
		DisplayName: user.Name,
		Birthday:    timestamppb.New(user.Birthday.Time),
		Gender:      convertSQLGenderToPB(user.Gender),
	}, nil
}

func (s *userService) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*emptypb.Empty, error) {
	if len(req.UserId) == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	userID := pgtype.UUID{
		Bytes: [16]byte(req.UserId),
		Valid: true,
	}

	if err := s.userRepo.DeleteUser(ctx, userID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete user: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func convertPBGenderToSQL(gender pb.UserGender) sqlc.GenderType {
	switch gender {
	case pb.UserGender_USER_GENDER_MALE:
		return sqlc.GenderTypeMale
	case pb.UserGender_USER_GENDER_FEMALE:
		return sqlc.GenderTypeFemale
	case pb.UserGender_USER_GENDER_DIVERSE:
		return sqlc.GenderTypeDiverse
	default:
		return sqlc.GenderTypeUnspecified
	}
}

func convertSQLGenderToPB(gender sqlc.GenderType) pb.UserGender {
	switch gender {
	case sqlc.GenderTypeMale:
		return pb.UserGender_USER_GENDER_MALE
	case sqlc.GenderTypeFemale:
		return pb.UserGender_USER_GENDER_FEMALE
	case sqlc.GenderTypeDiverse:
		return pb.UserGender_USER_GENDER_DIVERSE
	default:
		return pb.UserGender_USER_GENDER_UNSPECIFIED
	}
}
