package postgres

import (
	"LoudQuestionBot/internal/domain/errorz"
	"LoudQuestionBot/internal/domain/schema"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TeamRepo struct {
	pool *pgxpool.Pool
}

const maxTeamMembers = 10

func NewTeamRepo(pool *pgxpool.Pool) *TeamRepo {
	return &TeamRepo{pool: pool}
}

func (r *TeamRepo) Migrate(ctx context.Context) error {
	queries := []string{
		`CREATE EXTENSION IF NOT EXISTS pgcrypto;`,
		`CREATE TABLE IF NOT EXISTS teams (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			owner_id BIGINT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS team_members (
			team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
			user_id BIGINT NOT NULL,
			first_name TEXT NOT NULL DEFAULT '',
			last_name TEXT NOT NULL DEFAULT '',
			username TEXT NOT NULL DEFAULT '',
			joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY(team_id, user_id),
			UNIQUE(user_id)
		);`,
		`ALTER TABLE team_members ADD COLUMN IF NOT EXISTS first_name TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE team_members ADD COLUMN IF NOT EXISTS last_name TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE team_members ADD COLUMN IF NOT EXISTS username TEXT NOT NULL DEFAULT '';`,
		`CREATE INDEX IF NOT EXISTS idx_team_members_team_id ON team_members(team_id);`,
	}

	for _, q := range queries {
		if _, err := r.pool.Exec(ctx, q); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	return nil
}

func (r *TeamRepo) Create(ctx context.Context, ownerID int64, profile schema.UserProfile) (schema.Team, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return schema.Team{}, err
	}
	defer tx.Rollback(ctx)

	var out schema.Team
	if err := tx.QueryRow(ctx, `INSERT INTO teams(owner_id) VALUES($1) RETURNING id::text, owner_id, created_at;`, ownerID).Scan(
		&out.ID,
		&out.OwnerID,
		&out.CreatedAt,
	); err != nil {
		return schema.Team{}, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO team_members(team_id, user_id, first_name, last_name, username)
		VALUES($1, $2, $3, $4, $5);
	`, out.ID, ownerID, profile.FirstName, profile.LastName, profile.Username); err != nil {
		return schema.Team{}, mapPgErr(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return schema.Team{}, err
	}
	return out, nil
}

func (r *TeamRepo) GetByID(ctx context.Context, teamID string) (schema.Team, error) {
	var out schema.Team
	if err := r.pool.QueryRow(ctx, `SELECT id::text, owner_id, created_at FROM teams WHERE id = $1;`, teamID).Scan(
		&out.ID,
		&out.OwnerID,
		&out.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return schema.Team{}, errorz.ErrNotFound
		}
		return schema.Team{}, err
	}
	return out, nil
}

func (r *TeamRepo) GetByUserID(ctx context.Context, userID int64) (schema.Team, bool, error) {
	const query = `
	SELECT t.id::text, t.owner_id, t.created_at
	FROM teams t
	INNER JOIN team_members tm ON tm.team_id = t.id
	WHERE tm.user_id = $1
	LIMIT 1;
	`
	var out schema.Team
	if err := r.pool.QueryRow(ctx, query, userID).Scan(&out.ID, &out.OwnerID, &out.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return schema.Team{}, false, nil
		}
		return schema.Team{}, false, err
	}
	return out, true, nil
}

func (r *TeamRepo) ListMembers(ctx context.Context, teamID string) ([]schema.TeamMember, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT team_id::text, user_id, first_name, last_name, username, joined_at
		FROM team_members
		WHERE team_id = $1
		ORDER BY joined_at ASC;
	`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]schema.TeamMember, 0, 8)
	for rows.Next() {
		var m schema.TeamMember
		if err := rows.Scan(&m.TeamID, &m.UserID, &m.FirstName, &m.LastName, &m.Username, &m.JoinedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *TeamRepo) Join(ctx context.Context, teamID string, userID int64, profile schema.UserProfile) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var lockedID string
	if err := tx.QueryRow(ctx, `SELECT id::text FROM teams WHERE id = $1 FOR UPDATE;`, teamID).Scan(&lockedID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errorz.ErrNotFound
		}
		return err
	}

	var membersCount int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM team_members WHERE team_id = $1;`, teamID).Scan(&membersCount); err != nil {
		return err
	}
	if membersCount >= maxTeamMembers {
		return errorz.ErrLimitExceeded
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO team_members(team_id, user_id, first_name, last_name, username)
		VALUES($1, $2, $3, $4, $5);
	`, teamID, userID, profile.FirstName, profile.LastName, profile.Username); err != nil {
		return mapPgErr(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (r *TeamRepo) Leave(ctx context.Context, teamID string, userID int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var ownerID int64
	if err := tx.QueryRow(ctx, `SELECT owner_id FROM teams WHERE id = $1;`, teamID).Scan(&ownerID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errorz.ErrNotFound
		}
		return err
	}

	tag, err := tx.Exec(ctx, `DELETE FROM team_members WHERE team_id = $1 AND user_id = $2;`, teamID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errorz.ErrNotFound
	}

	if userID == ownerID {
		var nextOwnerID int64
		err := tx.QueryRow(ctx, `SELECT user_id FROM team_members WHERE team_id = $1 ORDER BY random() LIMIT 1;`, teamID).Scan(&nextOwnerID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				if _, err := tx.Exec(ctx, `DELETE FROM teams WHERE id = $1;`, teamID); err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			if _, err := tx.Exec(ctx, `UPDATE teams SET owner_id = $1 WHERE id = $2;`, nextOwnerID, teamID); err != nil {
				return err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (r *TeamRepo) Kick(ctx context.Context, teamID string, userID int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM team_members WHERE team_id = $1 AND user_id = $2;`, teamID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errorz.ErrNotFound
	}
	return nil
}

func (r *TeamRepo) TransferOwnership(ctx context.Context, teamID string, newOwnerID int64) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE teams t
		SET owner_id = $2
		WHERE t.id = $1
		  AND EXISTS (SELECT 1 FROM team_members tm WHERE tm.team_id = t.id AND tm.user_id = $2);
	`, teamID, newOwnerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errorz.ErrNotFound
	}
	return nil
}

func mapPgErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return errorz.ErrConflict
	}
	return err
}
