package game

import (
	"LoudQuestionBot/internal/domain/errorz"
	"LoudQuestionBot/internal/domain/repository"
	"LoudQuestionBot/internal/domain/schema"
	"context"
	"errors"
)

var ErrNoNewQuestions = errors.New("no new questions")

type Service struct {
	questions repository.QuestionRepository
}

func New(questions repository.QuestionRepository) *Service {
	return &Service{questions: questions}
}

func (s *Service) NextQuestion(ctx context.Context, userID int64) (schema.Question, error) {
	q, err := s.questions.GetActiveUnseenByUser(ctx, userID)
	if err != nil {
		if errors.Is(err, errorz.ErrNotFound) {
			return schema.Question{}, ErrNoNewQuestions
		}
		return schema.Question{}, err
	}
	if err := s.questions.MarkSeen(ctx, userID, q.ID); err != nil {
		return schema.Question{}, err
	}
	return q, nil
}

func (s *Service) AnswerByQuestionID(ctx context.Context, questionID int64) (string, error) {
	q, err := s.questions.GetByID(ctx, questionID)
	if err != nil {
		return "", err
	}
	if q.Status != schema.QuestionStatusActive {
		return "", errorz.ErrNotFound
	}
	return q.AnswerText, nil
}
