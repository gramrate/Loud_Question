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
		`CREATE TABLE IF NOT EXISTS questions (
			id BIGSERIAL PRIMARY KEY,
			question_text TEXT NOT NULL,
			answer_text TEXT NOT NULL,
			author_id BIGINT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`ALTER TABLE questions DROP COLUMN IF EXISTS example_text;`,
		`CREATE INDEX IF NOT EXISTS idx_questions_author_status ON questions(author_id, status);`,
		`CREATE TABLE IF NOT EXISTS user_seen_questions (
			user_id BIGINT NOT NULL,
			question_id BIGINT NOT NULL REFERENCES questions(id),
			seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
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
	RETURNING id, question_text, answer_text, author_id, status, created_at, updated_at;
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

func (r *QuestionRepo) GetByID(ctx context.Context, id int64) (schema.Question, error) {
	const query = `
	SELECT id, question_text, answer_text, author_id, status, created_at, updated_at
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
	SELECT q.id, q.question_text, q.answer_text, q.author_id, q.status, q.created_at, q.updated_at
	FROM questions q
	WHERE q.status = 'active'
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

func (r *QuestionRepo) MarkSeen(ctx context.Context, userID, questionID int64) error {
	const query = `
	INSERT INTO user_seen_questions (user_id, question_id)
	VALUES ($1, $2)
	ON CONFLICT (user_id, question_id) DO NOTHING;
	`
	_, err := r.pool.Exec(ctx, query, userID, questionID)
	return err
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
	SELECT id, question_text, answer_text, author_id, status, created_at, updated_at
	FROM questions
	WHERE author_id = $1 AND status = 'active'
	ORDER BY id DESC
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

func (r *QuestionRepo) UpdateByAuthor(ctx context.Context, authorID, questionID int64, draft schema.QuestionDraft) (schema.Question, error) {
	const query = `
	UPDATE questions
	SET question_text = $1,
		answer_text = $2,
		updated_at = NOW()
	WHERE id = $3 AND author_id = $4 AND status = 'active'
	RETURNING id, question_text, answer_text, author_id, status, created_at, updated_at;
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

func (r *QuestionRepo) SoftDeleteByAuthor(ctx context.Context, authorID, questionID int64) error {
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
