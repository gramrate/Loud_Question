package postgres

import (
	"LoudQuestionBot/internal/domain/schema"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) Migrate(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS bot_users (
			user_id BIGINT PRIMARY KEY,
			first_name TEXT NOT NULL DEFAULT '',
			last_name TEXT NOT NULL DEFAULT '',
			username TEXT NOT NULL DEFAULT '',
			language_code TEXT NOT NULL DEFAULT '',
			is_bot BOOLEAN NOT NULL DEFAULT FALSE,
			registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			last_interaction_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`ALTER TABLE bot_users ADD COLUMN IF NOT EXISTS registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW();`,
		`ALTER TABLE bot_users ADD COLUMN IF NOT EXISTS last_interaction_at TIMESTAMPTZ NOT NULL DEFAULT NOW();`,
		`DO $$
		BEGIN
			IF EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'bot_users' AND column_name = 'first_started_at'
			) THEN
				EXECUTE 'UPDATE bot_users SET registered_at = COALESCE(registered_at, first_started_at)';
			END IF;

			IF EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'bot_users' AND column_name = 'last_started_at'
			) THEN
				EXECUTE 'UPDATE bot_users SET last_interaction_at = GREATEST(last_interaction_at, last_started_at)';
			END IF;
		END $$;`,
		`ALTER TABLE bot_users DROP COLUMN IF EXISTS start_count;`,
		`ALTER TABLE bot_users DROP COLUMN IF EXISTS first_started_at;`,
		`ALTER TABLE bot_users DROP COLUMN IF EXISTS last_started_at;`,
	}

	for _, q := range queries {
		if _, err := r.pool.Exec(ctx, q); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	return nil
}

func (r *UserRepo) RegisterStart(ctx context.Context, user schema.BotUser) (schema.BotUser, bool, error) {
	existing, ok, err := r.GetByID(ctx, user.UserID)
	if err != nil {
		return schema.BotUser{}, false, err
	}
	if ok {
		return existing, false, nil
	}

	const insertQuery = `
	INSERT INTO bot_users (
		user_id, first_name, last_name, username, language_code, is_bot
	) VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING user_id, first_name, last_name, username, language_code, is_bot, registered_at, last_interaction_at;
	`
	var out schema.BotUser
	if err := r.pool.QueryRow(ctx, insertQuery,
		user.UserID, user.FirstName, user.LastName, user.Username, user.LanguageCode, user.IsBot,
	).Scan(
		&out.UserID, &out.FirstName, &out.LastName, &out.Username, &out.LanguageCode, &out.IsBot,
		&out.RegisteredAt, &out.LastInteractionAt,
	); err != nil {
		return schema.BotUser{}, false, err
	}
	return out, true, nil
}

func (r *UserRepo) GetByID(ctx context.Context, userID int64) (schema.BotUser, bool, error) {
	const query = `
	SELECT user_id, first_name, last_name, username, language_code, is_bot, registered_at, last_interaction_at
	FROM bot_users
	WHERE user_id = $1;
	`
	var out schema.BotUser
	if err := r.pool.QueryRow(ctx, query, userID).Scan(
		&out.UserID, &out.FirstName, &out.LastName, &out.Username, &out.LanguageCode, &out.IsBot,
		&out.RegisteredAt, &out.LastInteractionAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return schema.BotUser{}, false, nil
		}
		return schema.BotUser{}, false, err
	}
	return out, true, nil
}

func (r *UserRepo) TouchInteraction(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx, `UPDATE bot_users SET last_interaction_at = NOW() WHERE user_id = $1;`, userID)
	return err
}
