package sync

import (
	"context"
	"log"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"calendar/clients/calendar/ccommon"
	"calendar/clients/syncdb"
	"calendar/clients/tmanager"
	"calendar/clients/userinfo"
)


type Request struct {
	SrcCalendarID   string
	DstCalendarID   string
	SrcAccountEmail string
	DstAccountEmail string
	SyncInterval    time.Duration
}

type job struct {
	request   Request
	srcService *calendar.Service
	dstService *calendar.Service
	syncDB     *syncdb.DB
}

func RunSync(
	ctx context.Context,
	syncDB *syncdb.DB,
	tokenManager *tmanager.Manager,
	userInfo *userinfo.Manager,
	request Request,
) error {
	calendarTokens, err := ccommon.UsersCalendarsTokens(ctx, userInfo, tokenManager)
	if err != nil {
		return err
	}

	srcToken := ccommon.FindToken(calendarTokens, request.SrcAccountEmail)
	if srcToken == nil {
		return errors.New("source account not authenticated")
	}

	dstToken := ccommon.FindToken(calendarTokens, request.DstAccountEmail)
	if dstToken == nil {
		return errors.New("source account not authenticated")
	}

	config := tokenManager.Config()

	srcService, err := calendar.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx, srcToken)))
	if err != nil {
		return errors.Wrap(err, "unable to create calendar client")
	}

	dstService, err := calendar.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx, dstToken)))
	if err != nil {
		return errors.Wrap(err, "unable to create calendar client")
	}

	job := &job{
		request: request,
		syncDB: syncDB,
		srcService: srcService,
		dstService: dstService,
	}

	return job.run(ctx)
}


func (s *job) run(ctx context.Context) error {
	err := s.srcService.Events.
		List(s.request.SrcCalendarID).
		UpdatedMin(time.Now().Add(-s.request.SyncInterval).Format(time.RFC3339)).
		Pages(ctx, s.syncEvents)

	return errors.Wrap(err, "unable to sync events")
}

func (s *job) syncEvents(events *calendar.Events) error {
	for _, srcEvent := range events.Items {
		if err := s.syncEvent(srcEvent); err != nil {
			return err
		}
	}
	return nil
}


func (s *job) syncEvent(srcEvent *calendar.Event) error {
	r, err := s.syncDB.Find(
		syncdb.Event{
			EventID:      srcEvent.Id,
			AccountEmail: s.request.SrcAccountEmail,
			CalendarID:   s.request.SrcCalendarID,
		},
		s.request.DstAccountEmail,
		s.request.DstCalendarID,
	)
	if err == syncdb.ErrNotFound {
		if srcEvent.Status == ccommon.EventStatusCancelled {
			if srcEvent.RecurringEventId != "" {
				return s.deleteRecurringEventInstance(srcEvent)
			}
			return nil
		}
		return s.createEvent(srcEvent)
	}
	if err != nil {
		return err
	}
	return s.syncExistingEvent(srcEvent, r)
}

func (s *job) createEvent(srcEvent *calendar.Event) error {
	log.Println("create event")

	recurringEventId, err := s.mapRecurringEventId(srcEvent)
	if err != nil {
		return errors.Wrap(err, "failed to map recurring event id")
	}

	dstEvent := mapEvent(srcEvent)
	dstEvent.RecurringEventId = recurringEventId

	dstEvent, err = s.dstService.Events.Insert(s.request.DstCalendarID, dstEvent).Do()
	if err != nil {
		return errors.Wrapf(err, "failed to create event")
	}

	record := syncdb.Record{
		Src: syncdb.Event{
			EventID:      srcEvent.Id,
			AccountEmail: s.request.SrcAccountEmail,
			CalendarID:   s.request.SrcCalendarID,
		},
		Dst: syncdb.Event{
			EventID:      dstEvent.Id,
			AccountEmail: s.request.DstAccountEmail,
			CalendarID:   s.request.DstCalendarID,
		},
	}

	if err = s.syncDB.Insert(record); err != nil {
		return errors.Wrapf(err, "failed to save sync mapping")
	}

	log.Printf("created event: %s, %s\n", srcEvent.Id, srcEvent.RecurringEventId)
	return nil
}

func (s *job) syncExistingEvent(srcEvent *calendar.Event, r syncdb.Record) error {
	log.Printf("existing event: %s\n", r.Src.EventID)

	if srcEvent.Status == "cancelled" {
		return s.deleteDstEvent(r)
	}

	recurringEventId, err := s.mapRecurringEventId(srcEvent)
	if err != nil {
		return errors.Wrap(err, "failed to map recurring event id")
	}

	dstEvent := mapEvent(srcEvent)
	dstEvent.RecurringEventId = recurringEventId

	if dstEvent, err = s.dstService.Events.Update(s.request.DstCalendarID, r.Dst.EventID, dstEvent).Do(); err != nil {
		return errors.Wrapf(err, "failed to update event")
	}

	log.Printf("updated event: %s\n", srcEvent.Id)
	return nil
}

func (s *job) deleteDstEvent(r syncdb.Record) error {
	log.Printf("delete event: %s\n", r.Src.EventID)
	return ccommon.DeleteDstEvent(s.syncDB, s.dstService, r)
}

func (s *job) mapRecurringEventId(srcEvent *calendar.Event) (string, error) {
	r, err := s.syncDB.Find(
		syncdb.Event{
			EventID:      srcEvent.RecurringEventId,
			AccountEmail: s.request.SrcAccountEmail,
			CalendarID:   s.request.SrcCalendarID,
		},
		s.request.DstAccountEmail,
		s.request.DstCalendarID,
	)
	if err == syncdb.ErrNotFound {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return r.Dst.EventID, nil
}

func (s *job) deleteRecurringEventInstance(srcEvent *calendar.Event) error {
	log.Printf("skipping event: %s\n", srcEvent.Id)

	recurringEventId, err := s.mapRecurringEventId(srcEvent)
	if err != nil {
		return errors.Wrap(err, "failed to map recurring event id")
	}

	// recurring event already deleted
	if recurringEventId == "" {
		return nil
	}

	start := srcEvent.OriginalStartTime.DateTime
	if start == "" {
		start = srcEvent.OriginalStartTime.Date
	}

	dstInstances, err := s.dstService.Events.
		Instances(s.request.DstCalendarID, recurringEventId).
		OriginalStart(start).
		MaxResults(1).
		Do()
	if err != nil {
		return err
	}

	if len(dstInstances.Items) == 0 {
		return nil
	}

	dstEvent := dstInstances.Items[0]
	if err := s.dstService.Events.Delete(s.request.DstCalendarID, dstEvent.Id).Do(); err != nil {
		return errors.Wrapf(err, "failed to delete event")
	}

	log.Printf("deleted recurring event instance: %s\n", srcEvent.Id)
	return nil
}
