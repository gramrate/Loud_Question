package repository

import (
	"LoudQuestionBot/internal/domain/schema"
	"context"
)

type FormStateRepository interface {
	Get(ctx context.Context, userID int64) (schema.FormState, bool, error)
	Set(ctx context.Context, userID int64, state schema.FormState) error
	Delete(ctx context.Context, userID int64) error
}
