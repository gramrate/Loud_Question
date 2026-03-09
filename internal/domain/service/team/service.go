package team

import (
	"LoudQuestionBot/internal/domain/errorz"
	"LoudQuestionBot/internal/domain/repository"
	"LoudQuestionBot/internal/domain/schema"
	"context"
)

type Service struct {
	teams repository.TeamRepository
}

const MaxMembers = 10

func New(teams repository.TeamRepository) *Service {
	return &Service{teams: teams}
}

func (s *Service) Create(ctx context.Context, ownerID int64, profile schema.UserProfile) (schema.Team, error) {
	_, ok, err := s.GetByUserID(ctx, ownerID)
	if err != nil {
		return schema.Team{}, err
	}
	if ok {
		return schema.Team{}, errorz.ErrConflict
	}
	return s.teams.Create(ctx, ownerID, profile)
}

func (s *Service) GetByUserID(ctx context.Context, userID int64) (schema.Team, bool, error) {
	return s.teams.GetByUserID(ctx, userID)
}

func (s *Service) GetByID(ctx context.Context, teamID string) (schema.Team, error) {
	return s.teams.GetByID(ctx, teamID)
}

func (s *Service) Members(ctx context.Context, teamID string) ([]schema.TeamMember, error) {
	return s.teams.ListMembers(ctx, teamID)
}

func (s *Service) Join(ctx context.Context, teamID string, userID int64, profile schema.UserProfile) error {
	team, err := s.teams.GetByID(ctx, teamID)
	if err != nil {
		return err
	}
	current, inTeam, err := s.teams.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if inTeam {
		if current.ID == teamID {
			return errorz.ErrAlreadyExists
		}
		return errorz.ErrConflict
	}
	return s.teams.Join(ctx, team.ID, userID, profile)
}

func (s *Service) Leave(ctx context.Context, userID int64) error {
	team, ok, err := s.teams.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if !ok {
		return errorz.ErrNotFound
	}
	return s.teams.Leave(ctx, team.ID, userID)
}

func (s *Service) Kick(ctx context.Context, ownerID, memberID int64) error {
	team, ok, err := s.teams.GetByUserID(ctx, ownerID)
	if err != nil {
		return err
	}
	if !ok {
		return errorz.ErrNotFound
	}
	if team.OwnerID != ownerID {
		return errorz.ErrForbidden
	}
	if ownerID == memberID {
		return errorz.ErrForbidden
	}
	return s.teams.Kick(ctx, team.ID, memberID)
}

func (s *Service) TransferOwnership(ctx context.Context, ownerID, newOwnerID int64) error {
	team, ok, err := s.teams.GetByUserID(ctx, ownerID)
	if err != nil {
		return err
	}
	if !ok {
		return errorz.ErrNotFound
	}
	if team.OwnerID != ownerID {
		return errorz.ErrForbidden
	}
	if ownerID == newOwnerID {
		return nil
	}
	members, err := s.teams.ListMembers(ctx, team.ID)
	if err != nil {
		return err
	}
	found := false
	for _, m := range members {
		if m.UserID == newOwnerID {
			found = true
			break
		}
	}
	if !found {
		return errorz.ErrNotFound
	}
	return s.teams.TransferOwnership(ctx, team.ID, newOwnerID)
}
