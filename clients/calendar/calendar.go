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

func New(tokenManager *tmanager.Manager, userInfo *userinfo.Manager) *Manager {
	return &Manager{
		tokenManager: tokenManager,
		userInfo: userInfo,
	}
}

func (s *Manager) UsersCalendars() ([]UserCalendars, error) {
	tokens := s.tokenManager.List()

	result := make([]UserCalendars, 0, len(tokens))
	for _, token := range tokens {
		email, err := s.userInfo.Email(&token)
		if err != nil {
			return nil, err
		}

		calendars, err := s.calendars(&token)
		if err != nil {
			return nil, err
		}

		result = append(result, UserCalendars{
			Email: email,
			Calendars: calendars,
		})
	}

	return result, nil
}

func (s *Manager) calendars(token *oauth2.Token) ([]Into, error) {
	ctx := context.Background()
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

func (s *Manager) Sync() error {
	config := s.tokenManager.Config()
	tokens := s.tokenManager.List()

	for _, token := range tokens {
		if err := s.syncAccount(config, &token); err != nil {
			return err
		}
	}

	return nil
}

func (s *Manager) syncAccount(config *oauth2.Config, token *oauth2.Token) error {
	ctx := context.Background()
	srv, err := calendar.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx, token)))
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
