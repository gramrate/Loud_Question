package user

import (
	"LoudQuestionBot/internal/domain/repository"
	"LoudQuestionBot/internal/domain/schema"
	"context"
)

type Service struct {
	repo repository.UserRepository
}

func New(repo repository.UserRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) RegisterStart(ctx context.Context, user schema.BotUser) (schema.BotUser, bool, error) {
	return s.repo.RegisterStart(ctx, user)
}

func (s *Service) GetByID(ctx context.Context, userID int64) (schema.BotUser, bool, error) {
	return s.repo.GetByID(ctx, userID)
}

func (s *Service) TouchInteraction(ctx context.Context, userID int64) error {
	return s.repo.TouchInteraction(ctx, userID)
}
