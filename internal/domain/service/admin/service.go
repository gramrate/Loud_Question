package admin

import (
	"LoudQuestionBot/internal/domain/repository"
	"LoudQuestionBot/internal/domain/schema"
	"context"
)

type Service struct {
	questions repository.QuestionRepository
}

func New(questions repository.QuestionRepository) *Service {
	return &Service{questions: questions}
}

func (s *Service) CreateQuestion(ctx context.Context, authorID int64, draft schema.QuestionDraft) (schema.Question, error) {
	created, err := s.questions.Create(ctx, schema.Question{
		QuestionText: draft.QuestionText,
		AnswerText:   draft.AnswerText,
		AuthorID:     authorID,
		Status:       schema.QuestionStatusActive,
	})
	if err != nil {
		return schema.Question{}, err
	}
	if err := s.questions.MarkSeen(ctx, authorID, created.ID); err != nil {
		return schema.Question{}, err
	}
	return created, nil
}

func (s *Service) MyQuestions(ctx context.Context, authorID int64, page, pageSize int) (repository.ListQuestionsResult, error) {
	return s.questions.ListByAuthor(ctx, authorID, page, pageSize)
}

func (s *Service) GetQuestion(ctx context.Context, questionID int64) (schema.Question, error) {
	return s.questions.GetByID(ctx, questionID)
}

func (s *Service) UpdateQuestion(ctx context.Context, authorID, questionID int64, draft schema.QuestionDraft) (schema.Question, error) {
	return s.questions.UpdateByAuthor(ctx, authorID, questionID, draft)
}

func (s *Service) DeleteQuestion(ctx context.Context, authorID, questionID int64) error {
	return s.questions.SoftDeleteByAuthor(ctx, authorID, questionID)
}
