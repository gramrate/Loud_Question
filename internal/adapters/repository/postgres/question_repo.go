package postgres

import (
	"LoudQuestionBot/internal/domain/errorz"
	"LoudQuestionBot/internal/domain/repository"
	"LoudQuestionBot/internal/domain/schema"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type QuestionRepo struct {
	pool *pgxpool.Pool
}

func NewQuestionRepo(pool *pgxpool.Pool) *QuestionRepo {
	return &QuestionRepo{pool: pool}
}

func (r *QuestionRepo) Migrate(ctx context.Context) error {
	queries := []string{
		`CREATE EXTENSION IF NOT EXISTS pgcrypto;`,
		`CREATE TABLE IF NOT EXISTS questions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			question_text TEXT NOT NULL,
			answer_text TEXT NOT NULL,
			author_id BIGINT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`ALTER TABLE questions DROP COLUMN IF EXISTS example_text;`,
		`DO $$
		DECLARE
			id_udt text;
		BEGIN
			SELECT c.udt_name
			INTO id_udt
			FROM information_schema.columns c
			WHERE c.table_name = 'questions' AND c.column_name = 'id'
			LIMIT 1;

			IF id_udt IS NOT NULL AND id_udt <> 'uuid' THEN
				EXECUTE 'DROP TABLE IF EXISTS team_seen_questions';
				EXECUTE 'DROP TABLE IF EXISTS user_seen_questions';
				EXECUTE 'ALTER TABLE questions ALTER COLUMN id DROP DEFAULT';
				EXECUTE 'ALTER TABLE questions ALTER COLUMN id TYPE UUID USING gen_random_uuid()';
				EXECUTE 'ALTER TABLE questions ALTER COLUMN id SET DEFAULT gen_random_uuid()';
			END IF;
		END $$;`,
		`CREATE INDEX IF NOT EXISTS idx_questions_author_status ON questions(author_id, status);`,
		`CREATE TABLE IF NOT EXISTS user_seen_questions (
			user_id BIGINT NOT NULL,
			question_id UUID NOT NULL REFERENCES questions(id),
			seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY(user_id, question_id)
		);`,
		`CREATE TABLE IF NOT EXISTS team_seen_questions (
			team_id UUID NOT NULL,
			question_id UUID NOT NULL REFERENCES questions(id),
			seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY(team_id, question_id)
		);`,
		`CREATE TABLE IF NOT EXISTS user_answered_questions (
			user_id BIGINT NOT NULL,
			question_id UUID NOT NULL REFERENCES questions(id),
			answered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY(user_id, question_id)
		);`,
	}

	for _, q := range queries {
		if _, err := r.pool.Exec(ctx, q); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	return nil
}

func (r *QuestionRepo) Create(ctx context.Context, q schema.Question) (schema.Question, error) {
	const query = `
	INSERT INTO questions (question_text, answer_text, author_id, status)
	VALUES ($1, $2, $3, $4)
	RETURNING id::text, question_text, answer_text, author_id, status, created_at, updated_at;
	`
	var out schema.Question
	if err := r.pool.QueryRow(ctx, query, q.QuestionText, q.AnswerText, q.AuthorID, q.Status).Scan(
		&out.ID,
		&out.QuestionText,
		&out.AnswerText,
		&out.AuthorID,
		&out.Status,
		&out.CreatedAt,
		&out.UpdatedAt,
	); err != nil {
		return schema.Question{}, err
	}
	return out, nil
}

func (r *QuestionRepo) GetByID(ctx context.Context, id string) (schema.Question, error) {
	const query = `
	SELECT id::text, question_text, answer_text, author_id, status, created_at, updated_at
	FROM questions
	WHERE id = $1;
	`
	var out schema.Question
	if err := r.pool.QueryRow(ctx, query, id).Scan(
		&out.ID,
		&out.QuestionText,
		&out.AnswerText,
		&out.AuthorID,
		&out.Status,
		&out.CreatedAt,
		&out.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return schema.Question{}, errorz.ErrNotFound
		}
		return schema.Question{}, err
	}
	return out, nil
}

func (r *QuestionRepo) GetActiveUnseenByUser(ctx context.Context, userID int64) (schema.Question, error) {
	const query = `
	SELECT q.id::text, q.question_text, q.answer_text, q.author_id, q.status, q.created_at, q.updated_at
	FROM questions q
	WHERE q.status = 'active'
	  AND q.author_id <> $1
	  AND NOT EXISTS (
		SELECT 1
		FROM user_seen_questions usq
		WHERE usq.user_id = $1 AND usq.question_id = q.id
	)
	ORDER BY RANDOM()
	LIMIT 1;
	`

	var out schema.Question
	if err := r.pool.QueryRow(ctx, query, userID).Scan(
		&out.ID,
		&out.QuestionText,
		&out.AnswerText,
		&out.AuthorID,
		&out.Status,
		&out.CreatedAt,
		&out.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return schema.Question{}, errorz.ErrNotFound
		}
		return schema.Question{}, err
	}
	return out, nil
}

func (r *QuestionRepo) GetActiveUnseenByTeam(ctx context.Context, teamID string, userID int64) (schema.Question, error) {
	const query = `
	SELECT q.id::text, q.question_text, q.answer_text, q.author_id, q.status, q.created_at, q.updated_at
	FROM questions q
	WHERE q.status = 'active'
	  AND q.author_id <> $2
	  AND NOT EXISTS (
		SELECT 1
		FROM team_seen_questions tsq
		WHERE tsq.team_id = $1 AND tsq.question_id = q.id
	)
	ORDER BY RANDOM()
	LIMIT 1;
	`

	var out schema.Question
	if err := r.pool.QueryRow(ctx, query, teamID, userID).Scan(
		&out.ID,
		&out.QuestionText,
		&out.AnswerText,
		&out.AuthorID,
		&out.Status,
		&out.CreatedAt,
		&out.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return schema.Question{}, errorz.ErrNotFound
		}
		return schema.Question{}, err
	}
	return out, nil
}

func (r *QuestionRepo) MarkSeenByUser(ctx context.Context, userID int64, questionID string) error {
	const query = `
	INSERT INTO user_seen_questions (user_id, question_id)
	VALUES ($1, $2)
	ON CONFLICT (user_id, question_id) DO NOTHING;
	`
	_, err := r.pool.Exec(ctx, query, userID, questionID)
	return err
}

func (r *QuestionRepo) MarkSeenByTeam(ctx context.Context, teamID string, questionID string) error {
	const query = `
	INSERT INTO team_seen_questions (team_id, question_id)
	VALUES ($1, $2)
	ON CONFLICT (team_id, question_id) DO NOTHING;
	`
	_, err := r.pool.Exec(ctx, query, teamID, questionID)
	return err
}

func (r *QuestionRepo) CountSeenByTeam(ctx context.Context, teamID string) (int, error) {
	const query = `SELECT COUNT(*) FROM team_seen_questions WHERE team_id = $1;`
	var cnt int
	if err := r.pool.QueryRow(ctx, query, teamID).Scan(&cnt); err != nil {
		return 0, err
	}
	return cnt, nil
}

func (r *QuestionRepo) MarkAnsweredByUser(ctx context.Context, userID int64, questionID string) error {
	const query = `
	INSERT INTO user_answered_questions (user_id, question_id)
	VALUES ($1, $2)
	ON CONFLICT (user_id, question_id) DO NOTHING;
	`
	_, err := r.pool.Exec(ctx, query, userID, questionID)
	return err
}

func (r *QuestionRepo) CountAnsweredByUser(ctx context.Context, userID int64) (int, error) {
	const query = `SELECT COUNT(*) FROM user_answered_questions WHERE user_id = $1;`
	var cnt int
	if err := r.pool.QueryRow(ctx, query, userID).Scan(&cnt); err != nil {
		return 0, err
	}
	return cnt, nil
}

func (r *QuestionRepo) ListByAuthor(ctx context.Context, authorID int64, page, pageSize int) (repository.ListQuestionsResult, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	const countQuery = `SELECT COUNT(*) FROM questions WHERE author_id = $1 AND status = 'active';`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, authorID).Scan(&total); err != nil {
		return repository.ListQuestionsResult{}, err
	}

	const query = `
	SELECT id::text, question_text, answer_text, author_id, status, created_at, updated_at
	FROM questions
	WHERE author_id = $1 AND status = 'active'
	ORDER BY created_at DESC
	LIMIT $2 OFFSET $3;
	`
	rows, err := r.pool.Query(ctx, query, authorID, pageSize, offset)
	if err != nil {
		return repository.ListQuestionsResult{}, err
	}
	defer rows.Close()

	items := make([]schema.Question, 0, pageSize)
	for rows.Next() {
		var q schema.Question
		if err := rows.Scan(
			&q.ID,
			&q.QuestionText,
			&q.AnswerText,
			&q.AuthorID,
			&q.Status,
			&q.CreatedAt,
			&q.UpdatedAt,
		); err != nil {
			return repository.ListQuestionsResult{}, err
		}
		items = append(items, q)
	}
	if err := rows.Err(); err != nil {
		return repository.ListQuestionsResult{}, err
	}

	return repository.ListQuestionsResult{Items: items, Total: total}, nil
}

func (r *QuestionRepo) UpdateByAuthor(ctx context.Context, authorID int64, questionID string, draft schema.QuestionDraft) (schema.Question, error) {
	const query = `
	UPDATE questions
	SET question_text = $1,
		answer_text = $2,
		updated_at = NOW()
	WHERE id = $3 AND author_id = $4 AND status = 'active'
	RETURNING id::text, question_text, answer_text, author_id, status, created_at, updated_at;
	`

	var out schema.Question
	if err := r.pool.QueryRow(ctx, query, draft.QuestionText, draft.AnswerText, questionID, authorID).Scan(
		&out.ID,
		&out.QuestionText,
		&out.AnswerText,
		&out.AuthorID,
		&out.Status,
		&out.CreatedAt,
		&out.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return schema.Question{}, errorz.ErrForbidden
		}
		return schema.Question{}, err
	}
	return out, nil
}

func (r *QuestionRepo) SoftDeleteByAuthor(ctx context.Context, authorID int64, questionID string) error {
	const query = `
	UPDATE questions
	SET status = 'deleted', updated_at = NOW()
	WHERE id = $1 AND author_id = $2 AND status = 'active';
	`
	tag, err := r.pool.Exec(ctx, query, questionID, authorID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errorz.ErrForbidden
	}
	return nil
}
