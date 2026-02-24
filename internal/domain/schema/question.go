package schema

import "time"

type QuestionStatus string

const (
	QuestionStatusActive  QuestionStatus = "active"
	QuestionStatusDeleted QuestionStatus = "deleted"
	QuestionStatusDraft   QuestionStatus = "draft"
)

type Question struct {
	ID           int64
	QuestionText string
	AnswerText   string
	AuthorID     int64
	Status       QuestionStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
