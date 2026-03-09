package repository

import (
	"LoudQuestionBot/internal/domain/schema"
	"context"
)

type UserRepository interface {
	RegisterStart(ctx context.Context, user schema.BotUser) (schema.BotUser, bool, error)
	GetByID(ctx context.Context, userID int64) (schema.BotUser, bool, error)
	TouchInteraction(ctx context.Context, userID int64) error
}
