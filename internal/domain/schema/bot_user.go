package schema

import "time"

type BotUser struct {
	UserID            int64
	FirstName         string
	LastName          string
	Username          string
	LanguageCode      string
	IsBot             bool
	RegisteredAt      time.Time
	LastInteractionAt time.Time
}
