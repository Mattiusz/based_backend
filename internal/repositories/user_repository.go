package repositories

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mattiusz/based_backend/internal/gen/sqlc"
)

type UserRepository interface {
	CreateUser(ctx context.Context, req *sqlc.CreateUserParams) (*sqlc.User, error)
	GetUserByID(ctx context.Context, userID pgtype.UUID) (*sqlc.User, error)
}

type repository struct {
	queries *sqlc.Queries
}

func NewUserRepository(q *sqlc.Queries) UserRepository {
	return &repository{queries: q}
}

func (r *repository) CreateUser(ctx context.Context, req *sqlc.CreateUserParams) (*sqlc.User, error) {
	user, err := r.queries.CreateUser(ctx, *req)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *repository) GetUserByID(ctx context.Context, userID pgtype.UUID) (*sqlc.User, error) {
	user, err := r.queries.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &user, nil
}
