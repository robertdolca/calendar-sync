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
	}, syncMetadata.dstAccount, syncMetadata.dstCalendar)
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
	log.Println("create event")

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
	log.Printf("existing event: %s\n", r.Dst.Id)

	if srcEvent.Status == "cancelled" {
		return s.deleteExistingEvent(dstService, r)
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

	log.Printf("created event: %s\n", dstEvent.Id)
	return nil
}

func (s *Manager) deleteExistingEvent(service *calendar.Service, r syncdb.Record) error {
	log.Printf("delete event: %s\n", r.Dst.Id)

	dstEvent, err := service.Events.Get(r.Dst.CalendarId, r.Dst.Id).Do()
	if err != nil {
		return errors.Wrapf(err, "failed to get event before deletion")
	}

	if dstEvent.Status == eventStatusCancelled {
		log.Printf("event already deleted: %s\n", dstEvent.Id)
		return s.syncDB.Delete(r)
	}

	if err := service.Events.Delete(r.Dst.CalendarId, r.Dst.Id).Do(); err != nil {
		return errors.Wrapf(err, "failed to delete event")
	}
	log.Printf("deleted event: %s\n", dstEvent.Id)
	return s.syncDB.Delete(r)
}

func (s *Manager) mapRecurringEventId(syncMetadata syncMetadata, srcEvent *calendar.Event) (string, error) {
	r, err := s.syncDB.Find(syncdb.Event{
		Id:           srcEvent.RecurringEventId,
		AccountEmail: syncMetadata.srcAccount,
		CalendarId:   syncMetadata.srcCalendar,
	}, syncMetadata.dstAccount, syncMetadata.dstCalendar)
	if err == syncdb.ErrNotFound {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return r.Dst.Id, nil
}
