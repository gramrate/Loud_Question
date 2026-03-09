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

func (s *Service) NextQuestion(ctx context.Context, userID int64, teamID string) (schema.Question, error) {
	var (
		q   schema.Question
		err error
	)
	if teamID == "" {
		q, err = s.questions.GetActiveUnseenByUser(ctx, userID)
	} else {
		q, err = s.questions.GetActiveUnseenByTeam(ctx, teamID, userID)
	}
	if err != nil {
		if errors.Is(err, errorz.ErrNotFound) {
			return schema.Question{}, ErrNoNewQuestions
		}
		return schema.Question{}, err
	}
	if teamID == "" {
		if err := s.questions.MarkSeenByUser(ctx, userID, q.ID); err != nil {
			return schema.Question{}, err
		}
	} else {
		if err := s.questions.MarkSeenByTeam(ctx, teamID, q.ID); err != nil {
			return schema.Question{}, err
		}
	}
	return q, nil
}

func (s *Service) TeamAnsweredCount(ctx context.Context, teamID string) (int, error) {
	if teamID == "" {
		return 0, nil
	}
	cnt, err := s.questions.CountSeenByTeam(ctx, teamID)
	if err != nil {
		return 0, err
	}
	return cnt, nil
}

func (s *Service) AnswerByQuestionID(ctx context.Context, questionID string) (string, error) {
	q, err := s.questions.GetByID(ctx, questionID)
	if err != nil {
		return "", err
	}
	if q.Status != schema.QuestionStatusActive {
		return "", errorz.ErrNotFound
	}
	return q.AnswerText, nil
}

func (s *Service) MarkAnsweredByUser(ctx context.Context, userID int64, questionID string) error {
	return s.questions.MarkAnsweredByUser(ctx, userID, questionID)
}

func (s *Service) AnsweredByUserCount(ctx context.Context, userID int64) (int, error) {
	return s.questions.CountAnsweredByUser(ctx, userID)
}
