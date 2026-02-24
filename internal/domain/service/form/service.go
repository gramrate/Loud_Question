package form

import (
	"LoudQuestionBot/internal/domain/repository"
	"LoudQuestionBot/internal/domain/schema"
	"context"
)

type Service struct {
	repo repository.FormStateRepository
}

func New(repo repository.FormStateRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) StartCreate(ctx context.Context, userID int64) error {
	return s.repo.Set(ctx, userID, schema.FormState{Mode: schema.FormModeCreate, Step: schema.FormStepQuestion})
}

func (s *Service) StartEdit(ctx context.Context, userID, questionID int64, page int, draft schema.QuestionDraft) error {
	return s.repo.Set(ctx, userID, schema.FormState{
		Mode:       schema.FormModeEdit,
		Step:       schema.FormStepChooseField,
		QuestionID: questionID,
		Page:       page,
		Draft:      draft,
	})
}

func (s *Service) Get(ctx context.Context, userID int64) (schema.FormState, bool, error) {
	return s.repo.Get(ctx, userID)
}

func (s *Service) Save(ctx context.Context, userID int64, state schema.FormState) error {
	return s.repo.Set(ctx, userID, state)
}

func (s *Service) Cancel(ctx context.Context, userID int64) error {
	return s.repo.Delete(ctx, userID)
}
