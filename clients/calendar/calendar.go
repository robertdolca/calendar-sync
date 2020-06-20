package calendar

import (
	"calendar/clients/tmanager"
	"calendar/clients/userinfo"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
	"time"
)

type Into struct {
	Summary string
	Id string
	Deleted bool
}

type Manager struct {
	tokenManager *tmanager.Manager
	userInfo *userinfo.Manager
}

type UserCalendars struct {
	Email string
	Calendars []Into
}

type TokenCalendars struct {
	Email string
	Calendars []Into
	Token oauth2.Token
}

func New(tokenManager *tmanager.Manager, userInfo *userinfo.Manager) *Manager {
	return &Manager{
		tokenManager: tokenManager,
		userInfo: userInfo,
	}
}

func (s *Manager) UsersCalendars(ctx context.Context) ([]UserCalendars, error) {
	calendarTokens, err := s.usersCalendarsTokens(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]UserCalendars, 0, len(calendarTokens))
	for _, calendarToken := range calendarTokens {
		result = append(result, UserCalendars{
			Email: calendarToken.Email,
			Calendars: calendarToken.Calendars,
		})
	}

	return result, nil
}

func (s *Manager) usersCalendarsTokens(ctx context.Context, ) ([]TokenCalendars, error) {
	tokens := s.tokenManager.List()

	result := make([]TokenCalendars, 0, len(tokens))
	for _, token := range tokens {
		email, err := s.userInfo.Email(ctx, &token)
		if err != nil {
			return nil, err
		}

		calendars, err := s.calendars(ctx, &token)
		if err != nil {
			return nil, err
		}

		result = append(result, TokenCalendars{
			Email: email,
			Calendars: calendars,
			Token: token,
		})
	}

	return result, nil
}

func (s *Manager) calendars(ctx context.Context, token *oauth2.Token) ([]Into, error) {
	srv, err := calendar.NewService(ctx, option.WithTokenSource(s.tokenManager.Config().TokenSource(ctx, token)))
	if err != nil {
		return nil, errors.Wrap(err, "unable to create calendar client")
	}

	calendarList, err := srv.CalendarList.List().Do()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get calendar list")
	}

	var calendars []Into
	for _, entry := range calendarList.Items {
		calendars = append(calendars, calendarListEntryToCalendar(entry))
	}

	return calendars, nil
}

func (s *Manager) Sync(ctx context.Context, srcAccount, srcCalendar, dstAccount, dstCalendar string) error {

	return nil
}

func (s *Manager) sync(ctx context.Context, token *oauth2.Token) error {
	srv, err := calendar.NewService(ctx, option.WithTokenSource(s.tokenManager.Config().TokenSource(ctx, token)))
	if err != nil {
		return errors.Wrap(err, "unable to create calendar client")
	}

	err = srv.Events.
		List("primary").
		UpdatedMin(time.Now().AddDate(0, 0, -4).Format(time.RFC3339)).
		Pages(ctx, s.syncEvents)

	if err != nil {
		return errors.Wrap(err, "unable to retrieve events")
	}

	return nil
}

func (s *Manager) syncEvents(events *calendar.Events) error {
	for _, event := range events.Items {
		if event.Start == nil {
			continue
		}
		date := event.Start.DateTime
		if date == "" {
			date = event.Start.Date
		}
		fmt.Printf("%v (%v) %s\n", event.Summary, date, event.Id)
	}
	return nil
}

func calendarListEntryToCalendar(entry *calendar.CalendarListEntry) Into {
	summary := entry.SummaryOverride
	if summary == "" {
		summary = entry.Summary
	}
	return Into{
		Summary: summary,
		Id: entry.Id,
		Deleted: entry.Deleted,
	}
}
