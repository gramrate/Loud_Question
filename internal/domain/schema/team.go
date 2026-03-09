package schema

import "time"

type Team struct {
	ID        string
	OwnerID   int64
	CreatedAt time.Time
}

type TeamMember struct {
	TeamID    string
	UserID    int64
	FirstName string
	LastName  string
	Username  string
	JoinedAt  time.Time
}

type TeamWithMembers struct {
	Team        Team
	Members     []TeamMember
	AnsweredCnt int
}

type UserProfile struct {
	FirstName string
	LastName  string
	Username  string
}
