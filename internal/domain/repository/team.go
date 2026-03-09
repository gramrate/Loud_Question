package repository

import (
	"LoudQuestionBot/internal/domain/schema"
	"context"
)

type TeamRepository interface {
	Create(ctx context.Context, ownerID int64, profile schema.UserProfile) (schema.Team, error)
	GetByID(ctx context.Context, teamID string) (schema.Team, error)
	GetByUserID(ctx context.Context, userID int64) (schema.Team, bool, error)
	ListMembers(ctx context.Context, teamID string) ([]schema.TeamMember, error)
	Join(ctx context.Context, teamID string, userID int64, profile schema.UserProfile) error
	Leave(ctx context.Context, teamID string, userID int64) error
	Kick(ctx context.Context, teamID string, userID int64) error
	TransferOwnership(ctx context.Context, teamID string, newOwnerID int64) error
}
