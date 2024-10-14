package repository

import (
	"context"

	"github.com/cridenour/go-postgis"
	service "github.com/mattiusz/based_backend/internal/gen/proto/v1"
	"github.com/mattiusz/based_backend/internal/gen/sqlc"
)

type UserRepository interface {
	CreateUser(ctx context.Context, req *service.CreateUserRequest) (*service.UserResponse, error)
	GetUser(ctx context.Context, id int64) (*service.UserResponse, error)
}

type userRepository struct {
	queries *sqlc.Queries
}

func NewUserRepository(q *sqlc.Queries) UserRepository {
	return &userRepository{
		queries: q,
	}
}

func (r *userRepository) CreateUser(ctx context.Context, req *service.CreateUserRequest) (*service.UserResponse, error) {
	user, err := r.queries.CreateUser(ctx, sqlc.CreateUserParams{
		Name:     req.Name,
		Email:    req.Email,
		Location: postgis.Point{X: req.Longitude, Y: req.Latitude},
	})
	if err != nil {
		return nil, err
	}

	return &service.UserResponse{
		Id:        user.id,
		Name:      user.Name,
		Email:     user.Email,
		Longitude: user.Longitude,
		Latitude:  user.Latitude,
	}, nil
}

func (r *userRepository) GetUser(ctx context.Context, id int64) (*service.UserResponse, error) {
	user, err := r.queries.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}

	return &service.UserResponse{
		Id:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Longitude: user.Longitude,
		Latitude:  user.Latitude,
	}, nil
}
