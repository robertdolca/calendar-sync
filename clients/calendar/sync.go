package calendar

import (
	"context"
	"log"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"calendar/clients/syncdb"
)

const (
	eventStatusCancelled = "cancelled"
)

type syncMetadata struct {
	srcCalendar, dstCalendar string
	srcAccount, dstAccount string
}

func (s *Manager) Sync(
	ctx context.Context,
	srcAccount, srcCalendar,
	dstAccount, dstCalendar string,
	syncInterval time.Duration,
) error {
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

	return s.sync(ctx, srcToken, dstToken, syncInterval, syncMetadata{
		srcCalendar: srcCalendar,
		dstCalendar: dstCalendar,
		srcAccount:  srcAccount,
		dstAccount:  dstAccount,
	})
}

func (s *Manager) sync(
	ctx context.Context,
	srcToken, dstToken *oauth2.Token,
	syncInterval time.Duration,
	syncMetadata syncMetadata,
) error {
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
		List(syncMetadata.srcCalendar).
		UpdatedMin(time.Now().Add(-syncInterval).Format(time.RFC3339)).
		Pages(ctx, func(events *calendar.Events) error {
			return s.syncEvents(dstService, syncMetadata, events)
		})

	if err != nil {
		return errors.Wrap(err, "unable to sync events")
	}

	return nil
}

func (s *Manager) syncEvents(dstService *calendar.Service, syncMetadata syncMetadata, events *calendar.Events) error {
	for _, srcEvent := range events.Items {
		if err := s.syncEvent(dstService, syncMetadata, srcEvent); err != nil {
			return err
		}
	}
	return nil
}


func (s *Manager) syncEvent(dstService *calendar.Service, syncMetadata syncMetadata, srcEvent *calendar.Event) error {
	r, err := s.syncDB.Find(syncdb.Event{
		Id:           srcEvent.Id,
		AccountEmail: syncMetadata.srcAccount,
		CalendarId:   syncMetadata.srcCalendar,
	})
	if err == syncdb.ErrNotFound {
		if srcEvent.Status == eventStatusCancelled {
			return nil
		}
		return s.createEvent(dstService, syncMetadata, srcEvent)
	}
	if err != nil {
		return err
	}
	return s.syncExistingEvent(dstService, syncMetadata, srcEvent, r)
}

func (s *Manager) createEvent(dstService *calendar.Service, syncMetadata syncMetadata, srcEvent *calendar.Event) error {
	log.Printf("create event")

	recurringEventId, err := s.mapRecurringEventId(syncMetadata, srcEvent)
	if err != nil {
		return errors.Wrap(err, "failed to map recurring event id")
	}

	dstEvent := mapEvent(srcEvent)
	dstEvent.RecurringEventId = recurringEventId

	dstEvent, err = dstService.Events.Insert(syncMetadata.dstCalendar, dstEvent).Do()
	if err != nil {
		return errors.Wrapf(err, "failed to create event")
	}

	record := syncdb.Record{
		Src: syncdb.Event{
			Id:           srcEvent.Id,
			AccountEmail: syncMetadata.srcAccount,
			CalendarId:   syncMetadata.srcCalendar,
		},
		Dst: syncdb.Event{
			Id:           dstEvent.Id,
			AccountEmail: syncMetadata.dstAccount,
			CalendarId:   syncMetadata.dstCalendar,
		},
	}

	if err = s.syncDB.Insert(record); err != nil {
		return errors.Wrapf(err, "failed to save sync mapping")
	}
	return nil
}

func (s *Manager) syncExistingEvent(
	dstService *calendar.Service,
	syncMetadata syncMetadata,
	srcEvent *calendar.Event,
	r syncdb.Record,
) error {
	log.Println("existing event")

	if srcEvent.Status == "cancelled" {
		return s.deleteExistingEvent(dstService, syncMetadata, r)
	}

	recurringEventId, err := s.mapRecurringEventId(syncMetadata, srcEvent)
	if err != nil {
		return errors.Wrap(err, "failed to map recurring event id")
	}

	dstEvent := mapEvent(srcEvent)
	dstEvent.RecurringEventId = recurringEventId

	if dstEvent, err = dstService.Events.Update(syncMetadata.dstCalendar, r.Dst.Id, dstEvent).Do(); err != nil {
		return errors.Wrapf(err, "failed to update event")
	}
	return nil
}

func (s *Manager) deleteExistingEvent(
	dstService *calendar.Service,
	syncMetadata syncMetadata,
	r syncdb.Record,
) error {
	log.Println("delete event")

	dstEvent, err := dstService.Events.Get(syncMetadata.dstCalendar, r.Dst.Id).Do()
	if err != nil {
		return errors.Wrapf(err, "failed to get event before deletion")
	}

	if dstEvent.Status == eventStatusCancelled {
		log.Println("event already deleted")
		return s.syncDB.Delete(r.Src)
	}

	if err := dstService.Events.Delete(syncMetadata.dstCalendar, r.Dst.Id).Do(); err != nil {
		return errors.Wrapf(err, "failed to delete event")
	}
	return s.syncDB.Delete(r.Src)
}

func (s *Manager) mapRecurringEventId(syncMetadata syncMetadata, srcEvent *calendar.Event) (string, error) {
	r, err := s.syncDB.Find(syncdb.Event{
		Id:           srcEvent.RecurringEventId,
		AccountEmail: syncMetadata.srcAccount,
		CalendarId:   syncMetadata.srcCalendar,
	})
	if err == syncdb.ErrNotFound {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return r.Dst.Id, nil
}
