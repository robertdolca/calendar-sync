package ccommon

import (
	"context"

	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/robertdolca/calendar-sync/clients/syncdb"
	"github.com/robertdolca/calendar-sync/clients/tmanager"
	"github.com/robertdolca/calendar-sync/clients/userinfo"
)

const (
	EventStatusCancelled = "cancelled"
)

type CalendarInfo struct {
	Summary string
	Id      string
	Deleted bool
}

type TokenCalendars struct {
	Email     string
	Calendars []CalendarInfo
	Token     oauth2.Token
}

func calendarListEntryToCalendar(entry *calendar.CalendarListEntry) CalendarInfo {
	summary := entry.SummaryOverride
	if summary == "" {
		summary = entry.Summary
	}
	return CalendarInfo{
		Summary: summary,
		Id:      entry.Id,
		Deleted: entry.Deleted,
	}
}

func FindToken(tokensEmail []TokenCalendars, email string) *oauth2.Token {
	for _, tokeEmail := range tokensEmail {
		if tokeEmail.Email == email {
			return &tokeEmail.Token
		}
	}
	return nil
}

func UsersCalendarsTokens(
	ctx context.Context,
	userInfo *userinfo.Manager,
	tokenMgr *tmanager.Manager,
) ([]TokenCalendars, error) {
	tokens := tokenMgr.List()
	config := tokenMgr.Config()

	result := make([]TokenCalendars, 0, len(tokens))
	for _, token := range tokens {
		email, err := userInfo.Email(ctx, &token)
		if err != nil {
			return nil, err
		}

		calendars, err := calendars(ctx, config, &token)
		if err != nil {
			return nil, err
		}

		result = append(result, TokenCalendars{
			Email:     email,
			Calendars: calendars,
			Token:     token,
		})
	}

	return result, nil
}

func calendars(ctx context.Context, config *oauth2.Config, token *oauth2.Token) ([]CalendarInfo, error) {
	srv, err := calendar.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx, token)))
	if err != nil {
		return nil, errors.Wrap(err, "unable to create calendar client")
	}

	calendarList, err := srv.CalendarList.List().Do()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get calendar list")
	}

	var calendars []CalendarInfo
	for _, entry := range calendarList.Items {
		calendars = append(calendars, calendarListEntryToCalendar(entry))
	}

	return calendars, nil
}

func DeleteDstEvent(syncDB *syncdb.DB, dstService *calendar.Service, r syncdb.Record) error {
	dstEvent, err := dstService.Events.Get(r.Dst.CalendarID, r.Dst.EventID).Do()
	if err != nil {
		return errors.Wrapf(err, "failed to get event before deletion")
	}

	if dstEvent.Status == EventStatusCancelled {
		return syncDB.Delete(r)
	}

	if err := dstService.Events.Delete(r.Dst.CalendarID, r.Dst.EventID).Do(); err != nil {
		return errors.Wrapf(err, "failed to delete event")
	}
	return syncDB.Delete(r)
}
