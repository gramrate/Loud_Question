package access

type Service struct {
	admins map[int64]struct{}
}

func New(admins map[int64]struct{}) *Service {
	if admins == nil {
		admins = map[int64]struct{}{}
	}
	return &Service{admins: admins}
}

func (s *Service) IsAdmin(userID int64) bool {
	_, ok := s.admins[userID]
	return ok
}
