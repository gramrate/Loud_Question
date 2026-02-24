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
	GetByID(ctx context.Context, id int64) (schema.Question, error)
	GetActiveUnseenByUser(ctx context.Context, userID int64) (schema.Question, error)
	MarkSeen(ctx context.Context, userID, questionID int64) error
	ListByAuthor(ctx context.Context, authorID int64, page, pageSize int) (ListQuestionsResult, error)
	UpdateByAuthor(ctx context.Context, authorID, questionID int64, draft schema.QuestionDraft) (schema.Question, error)
	SoftDeleteByAuthor(ctx context.Context, authorID, questionID int64) error
}
