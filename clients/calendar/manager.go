package calendar

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/robertdolca/calendar-sync/clients/calendar/ccommon"
	"github.com/robertdolca/calendar-sync/clients/calendar/sync"
	"github.com/robertdolca/calendar-sync/clients/syncdb"
	"github.com/robertdolca/calendar-sync/clients/tmanager"
	"github.com/robertdolca/calendar-sync/clients/userinfo"
)

type Manager struct {
	tokenManager *tmanager.Manager
	userInfo     *userinfo.Manager
	syncDB       *syncdb.DB
}

type UserCalendars struct {
	Email     string
	Calendars []ccommon.CalendarInfo
}

func New(tokenManager *tmanager.Manager, userInfo *userinfo.Manager, syncDB *syncdb.DB) *Manager {
	return &Manager{
		tokenManager: tokenManager,
		userInfo:     userInfo,
		syncDB:       syncDB,
	}
}

func (s *Manager) UsersCalendars(ctx context.Context) ([]UserCalendars, error) {
	calendarTokens, err := ccommon.UsersCalendarsTokens(ctx, s.userInfo, s.tokenManager)
	if err != nil {
		return nil, err
	}

	result := make([]UserCalendars, 0, len(calendarTokens))
	for _, calendarToken := range calendarTokens {
		result = append(result, UserCalendars{
			Email:     calendarToken.Email,
			Calendars: calendarToken.Calendars,
		})
	}

	return result, nil
}

func (s *Manager) Clear(ctx context.Context, accountEmail, calendarID string) error {
	calendarTokens, err := ccommon.UsersCalendarsTokens(ctx, s.userInfo, s.tokenManager)
	if err != nil {
		return err
	}

	token := ccommon.FindToken(calendarTokens, accountEmail)
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
		if err := ccommon.DeleteDstEvent(s.syncDB, service, record); err != nil {
			return err
		}
	}

	return nil
}

func (s *Manager) Sync(ctx context.Context, request sync.Request) error {
	return sync.RunSync(ctx, s.syncDB, s.tokenManager, s.userInfo, request)
}
