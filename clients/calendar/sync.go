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
	srcCalendarID, dstCalendarID     string
	srcAccountEmail, dstAccountEmail string
}

func (s *Manager) Sync(
	ctx context.Context,
	srcAccountEmail, srcCalendarID,
	dstAccountEmail, dstCalendarID string,
	syncInterval time.Duration,
) error {
	calendarTokens, err := s.usersCalendarsTokens(ctx)
	if err != nil {
		return err
	}

	srcToken := findToken(calendarTokens, srcAccountEmail)
	if srcToken == nil {
		return errors.New("source account not authenticated")
	}

	dstToken := findToken(calendarTokens, dstAccountEmail)
	if dstToken == nil {
		return errors.New("source account not authenticated")
	}

	return s.sync(ctx, srcToken, dstToken, syncInterval, syncMetadata{
		srcCalendarID:   srcCalendarID,
		dstCalendarID:   dstCalendarID,
		srcAccountEmail: srcAccountEmail,
		dstAccountEmail: dstAccountEmail,
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
		List(syncMetadata.srcCalendarID).
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
		EventID:      srcEvent.Id,
		AccountEmail: syncMetadata.srcAccountEmail,
		CalendarID:   syncMetadata.srcCalendarID,
	}, syncMetadata.dstAccountEmail, syncMetadata.dstCalendarID)
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

	dstEvent, err = dstService.Events.Insert(syncMetadata.dstCalendarID, dstEvent).Do()
	if err != nil {
		return errors.Wrapf(err, "failed to create event")
	}

	record := syncdb.Record{
		Src: syncdb.Event{
			EventID:      srcEvent.Id,
			AccountEmail: syncMetadata.srcAccountEmail,
			CalendarID:   syncMetadata.srcCalendarID,
		},
		Dst: syncdb.Event{
			EventID:      dstEvent.Id,
			AccountEmail: syncMetadata.dstAccountEmail,
			CalendarID:   syncMetadata.dstCalendarID,
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
	log.Printf("existing event: %s\n", r.Dst.EventID)

	if srcEvent.Status == "cancelled" {
		return s.deleteDstEvent(dstService, r)
	}

	recurringEventId, err := s.mapRecurringEventId(syncMetadata, srcEvent)
	if err != nil {
		return errors.Wrap(err, "failed to map recurring event id")
	}

	dstEvent := mapEvent(srcEvent)
	dstEvent.RecurringEventId = recurringEventId

	if dstEvent, err = dstService.Events.Update(syncMetadata.dstCalendarID, r.Dst.EventID, dstEvent).Do(); err != nil {
		return errors.Wrapf(err, "failed to update event")
	}

	log.Printf("created event: %s\n", dstEvent.Id)
	return nil
}

func (s *Manager) deleteDstEvent(service *calendar.Service, r syncdb.Record) error {
	log.Printf("delete event: %s\n", r.Dst.EventID)

	dstEvent, err := service.Events.Get(r.Dst.CalendarID, r.Dst.EventID).Do()
	if err != nil {
		return errors.Wrapf(err, "failed to get event before deletion")
	}

	if dstEvent.Status == eventStatusCancelled {
		log.Printf("event already deleted: %s\n", dstEvent.Id)
		return s.syncDB.Delete(r)
	}

	if err := service.Events.Delete(r.Dst.CalendarID, r.Dst.EventID).Do(); err != nil {
		return errors.Wrapf(err, "failed to delete event")
	}
	log.Printf("deleted event: %s\n", dstEvent.Id)
	return s.syncDB.Delete(r)
}

func (s *Manager) mapRecurringEventId(syncMetadata syncMetadata, srcEvent *calendar.Event) (string, error) {
	r, err := s.syncDB.Find(syncdb.Event{
		EventID:      srcEvent.RecurringEventId,
		AccountEmail: syncMetadata.srcAccountEmail,
		CalendarID:   syncMetadata.srcCalendarID,
	}, syncMetadata.dstAccountEmail, syncMetadata.dstCalendarID)
	if err == syncdb.ErrNotFound {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return r.Dst.EventID, nil
}
