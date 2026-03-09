package repository

import (
	"LoudQuestionBot/internal/domain/schema"
	"context"
)

type ListQuestionsResult struct {
	Items []schema.Question
	Total int
}

type QuestionRepository interface {
	Create(ctx context.Context, q schema.Question) (schema.Question, error)
	GetByID(ctx context.Context, id string) (schema.Question, error)
	GetActiveUnseenByUser(ctx context.Context, userID int64) (schema.Question, error)
	GetActiveUnseenByTeam(ctx context.Context, teamID string, userID int64) (schema.Question, error)
	MarkSeenByUser(ctx context.Context, userID int64, questionID string) error
	MarkSeenByTeam(ctx context.Context, teamID string, questionID string) error
	CountSeenByTeam(ctx context.Context, teamID string) (int, error)
	ListByAuthor(ctx context.Context, authorID int64, page, pageSize int) (ListQuestionsResult, error)
	UpdateByAuthor(ctx context.Context, authorID int64, questionID string, draft schema.QuestionDraft) (schema.Question, error)
	SoftDeleteByAuthor(ctx context.Context, authorID int64, questionID string) error
}
