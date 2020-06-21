package calendar

import (
	"context"

	"github.com/pkg/errors"
)

func (s *Manager) Clear(ctx context.Context, account, calendar string) error {
	calendarTokens, err := s.usersCalendarsTokens(ctx)
	if err != nil {
		return err
	}

	token := findToken(calendarTokens, account)
	if token == nil {
		return errors.New("source account not authenticated")
	}


	return nil
}
