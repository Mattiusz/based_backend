package repositories

import (
	"context"

	"github.com/mattiusz/based_backend/internal/gen/sqlc"
)

// ChatRepository interface defines all chat-related database operations
type ChatRepository interface {
	CreateMessage(ctx context.Context, params *sqlc.CreateChatMessageParams) (*sqlc.ChatMessage, error)
	GetEventMessages(ctx context.Context, params *sqlc.GetEventMessagesParams) ([]sqlc.GetEventMessagesRow, error)
	LikeMessage(ctx context.Context, params *sqlc.LikeMessageParams) error
	UnlikeMessage(ctx context.Context, params *sqlc.UnlikeMessageParams) error
	DeleteMesssage(ctx context.Context, params *sqlc.DeleteChatMessageParams) error
}

type chatRepository struct {
	queries *sqlc.Queries
}

func NewChatRepository(q *sqlc.Queries) ChatRepository {
	return &chatRepository{queries: q}
}

func (r *chatRepository) CreateMessage(ctx context.Context, params *sqlc.CreateChatMessageParams) (*sqlc.ChatMessage, error) {
	message, err := r.queries.CreateChatMessage(ctx, *params)
	if err != nil {
		return nil, err
	}
	return &message, nil
}

func (r *chatRepository) GetEventMessages(ctx context.Context, params *sqlc.GetEventMessagesParams) ([]sqlc.GetEventMessagesRow, error) {
	messages, err := r.queries.GetEventMessages(ctx, *params)
	if err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *chatRepository) LikeMessage(ctx context.Context, params *sqlc.LikeMessageParams) error {
	return r.queries.LikeMessage(ctx, *params)
}

func (r *chatRepository) UnlikeMessage(ctx context.Context, params *sqlc.UnlikeMessageParams) error {
	return r.queries.UnlikeMessage(ctx, *params)
}

func (r *chatRepository) DeleteMesssage(ctx context.Context, params *sqlc.DeleteChatMessageParams) error {
	return r.queries.DeleteChatMessage(ctx, *params)
}
