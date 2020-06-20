package calendar

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

func (s *Manager) Sync(ctx context.Context, srcAccount, srcCalendar, dstAccount, dstCalendar string) error {
	calendarTokens, err := s.usersCalendarsTokens(ctx)
	if err != nil {
		return err
	}

	srcToken := findToken(calendarTokens, srcAccount)
	if srcToken == nil {
		return errors.New("source account not authenticated")
	}

	dstToken := findToken(calendarTokens, dstAccount)
	if dstToken == nil {
		return errors.New("source account not authenticated")
	}

	return s.sync(ctx, srcToken, dstToken, srcCalendar, dstCalendar)
}

func (s *Manager) sync(ctx context.Context, srcToken, dstToken *oauth2.Token, srcCalendar, dstCalendar string) error {
	config := s.tokenManager.Config()

	srcService, err := calendar.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx, srcToken)))
	if err != nil {
		return errors.Wrap(err, "unable to create calendar client")
	}

	dstService, err := calendar.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx, dstToken)))
	if err != nil {
		return errors.Wrap(err, "unable to create calendar client")
	}

	err = srcService.Events.
		List(srcCalendar).
		UpdatedMin(updateInterval).
		Pages(ctx, func(events *calendar.Events) error {
			return s.syncEvents(dstService, dstCalendar, events)
		})

	if err != nil {
		return errors.Wrap(err, "unable to sync events")
	}

	return nil
}

func (s *Manager) syncEvents(dstService *calendar.Service, dstCalendar string, events *calendar.Events) error {
	for _, event := range events.Items {
		if event.Status == "cancelled" {

		} else {
			_, err := dstService.Events.Insert(dstCalendar, mapEvent(event)).Do()
			if err != nil {
				prettyPrint(event)
				return err
			}
			println(".")
		}
	}
	return nil
}

func prettyPrint(i interface{}) {
	s, _ := json.MarshalIndent(i, "", "\t")
	fmt.Println(string(s))
}
