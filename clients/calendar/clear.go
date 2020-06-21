package calendar

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

func (s *Manager) Clear(ctx context.Context, accountEmail, calendarID string) error {
	calendarTokens, err := s.usersCalendarsTokens(ctx)
	if err != nil {
		return err
	}

	token := findToken(calendarTokens, accountEmail)
	if token == nil {
		return errors.New("source account not authenticated")
	}

	config := s.tokenManager.Config()
	service, err := calendar.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx, token)))
	if err != nil {
		return errors.Wrap(err, "unable to create calendar client")
	}


	records, err := s.syncDB.ListDst(accountEmail, calendarID)
	if err != nil {
		return errors.New("failed to list records")
	}

	for _, record := range records {
		if err := s.deleteExistingEvent(service, record); err != nil {
			return err
		}
	}

	return nil
}
