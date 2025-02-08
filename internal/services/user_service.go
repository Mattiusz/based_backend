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

type userService struct {
	pb.UnimplementedUserServiceServer
	userRepo repositories.UserRepository
}

func NewUserService(repo repositories.UserRepository) pb.UserServiceServer {
	return &userService{userRepo: repo}
}

func (s *userService) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.User, error) {
	authenticatedUserID, err := interceptors.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	userID := convertUUID([]byte(authenticatedUserID))

	params := &sqlc.CreateUserParams{
		UserID:   userID,
		Name:     req.DisplayName,
		Birthday: pgtype.Date{Time: req.Birthday.AsTime(), Valid: req.Birthday != nil},
		Gender:   convertPBGenderToSQL(req.Gender),
	}

	user, err := s.userRepo.CreateUser(ctx, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	return &pb.User{
		DisplayName: user.Name,
		Birthday:    timestamppb.New(user.Birthday.Time),
		Gender:      convertSQLGenderToPB(user.Gender),
		UserId:      []byte(authenticatedUserID),
	}, nil
}

func (s *userService) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.User, error) {
	authenticatedUserID, err := interceptors.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	userID := convertUUID([]byte(authenticatedUserID))

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

	updatedUser, err := s.userRepo.UpdateUser(ctx, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update user: %v", err)
	}

	return &pb.User{
		DisplayName: updatedUser.Name,
		Birthday:    timestamppb.New(updatedUser.Birthday.Time),
		Gender:      convertSQLGenderToPB(updatedUser.Gender),
		UserId:      []byte(authenticatedUserID),
	}, nil
}

func (s *userService) GetUser(ctx context.Context, _ *emptypb.Empty) (*pb.User, error) {
	authenticatedUserID, err := interceptors.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	userID := convertUUID([]byte(authenticatedUserID))

	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}

	return &pb.User{
		DisplayName: user.Name,
		Birthday:    timestamppb.New(user.Birthday.Time),
		Gender:      convertSQLGenderToPB(user.Gender),
		UserId:      []byte(authenticatedUserID),
	}, nil
}

func (s *userService) DeleteUser(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	authenticatedUserID, err := interceptors.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	userID := convertUUID([]byte(authenticatedUserID))

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
