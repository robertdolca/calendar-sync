package sync

import (
	"context"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/time/rate"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/robertdolca/calendar-sync/clients/calendar/ccommon"
	"github.com/robertdolca/calendar-sync/clients/syncdb"
	"github.com/robertdolca/calendar-sync/clients/tmanager"
	"github.com/robertdolca/calendar-sync/clients/userinfo"
)

type Request struct {
	SrcCalendarID       string
	DstCalendarID       string
	SrcAccountEmail     string
	DstAccountEmail     string
	IncludeTentative    bool
	IncludeNotGoing     bool
	IncludeNotResponded bool
	IncludeOutOfOffice  bool
	ExcludeTitleRegex   *regexp.Regexp
	SyncInterval        time.Duration
	MappingOptions      MappingOptions
}

type MappingOptions struct {
	CopyDescription bool
	CopyLocation    bool
	TitleOverride   string
	Visibility      string
	CopyColor       bool
}

type job struct {
	ctx          context.Context
	request      Request
	srcService   *calendar.Service
	dstService   *calendar.Service
	syncDB       *syncdb.DB
	rateLLimiter *rate.Limiter
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
		ctx:          ctx,
		request:      request,
		syncDB:       syncDB,
		srcService:   srcService,
		dstService:   dstService,
		rateLLimiter: rate.NewLimiter(rate.Every(250*time.Millisecond), 1),
	}

	return job.run()
}

func (s *job) run() error {
	call := s.srcService.Events.
		List(s.request.SrcCalendarID).
		OrderBy("updated")

	if s.request.SyncInterval != 0 {
		call = call.UpdatedMin(time.Now().Add(-s.request.SyncInterval).Format(time.RFC3339))
	}

	err := call.Pages(s.ctx, s.syncEvents)

	return errors.Wrap(err, "unable to sync events")
}

func (s *job) syncEvents(events *calendar.Events) error {
	for _, srcEvent := range events.Items {
		if err := s.syncEvent(srcEvent); err != nil {
			return err
		}
	}
	// wait for a slot before next page
	if err := s.rateLLimiter.Wait(s.ctx); err != nil {
		return err
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
		if srcEvent.Status == ccommon.EventStatusCancelled || s.shouldExclude(srcEvent) {
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

func (s *job) shouldExclude(event *calendar.Event) bool {
	responseStatus := eventResponseStatus(event)
	if !s.request.IncludeNotGoing && responseStatus == "declined" {
		return true
	}
	if !s.request.IncludeTentative && responseStatus == "tentative" {
		return true
	}
	if !s.request.IncludeNotResponded && responseStatus == "needsAction" {
		return true
	}
	if !s.request.IncludeOutOfOffice && strings.HasPrefix(event.Description, "This is an out-of-office event") {
		return true
	}
	if s.request.ExcludeTitleRegex != nil && s.request.ExcludeTitleRegex.MatchString(event.Summary) {
		return true
	}
	return false
}

func (s *job) createEvent(srcEvent *calendar.Event) error {
	log.Printf("creating event: %s, %s\n", srcEvent.Id, srcEvent.RecurringEventId)

	mappedRecurringEventId, err := s.mapRecurringEventId(srcEvent.RecurringEventId)
	if err != nil {
		return errors.Wrap(err, "failed to map recurring event id")
	}

	if srcEvent.RecurringEventId != "" && mappedRecurringEventId == "" {
		log.Println("skipping recurring event instance for recurring event that does not exist")
		return nil
	}

	dstEvent := mapEvent(srcEvent, s.request.MappingOptions)
	dstEvent.RecurringEventId = mappedRecurringEventId

	if err := s.rateLLimiter.Wait(s.ctx); err != nil {
		return err
	}
	dstEvent, err = s.dstService.Events.Insert(s.request.DstCalendarID, dstEvent).Do()
	if err != nil {
		return errors.Wrapf(err, "failed to create event")
	}

	if err = s.createMapping(srcEvent.Id, dstEvent.Id); err != nil {
		return err
	}

	log.Printf("created event: %s, %s\n", srcEvent.Id, srcEvent.RecurringEventId)
	return nil
}

func (s *job) createMapping(srcEventId, dstEventId string) error {
	record := syncdb.Record{
		Src: syncdb.Event{
			EventID:      srcEventId,
			AccountEmail: s.request.SrcAccountEmail,
			CalendarID:   s.request.SrcCalendarID,
		},
		Dst: syncdb.Event{
			EventID:      dstEventId,
			AccountEmail: s.request.DstAccountEmail,
			CalendarID:   s.request.DstCalendarID,
		},
	}
	if err := s.syncDB.Insert(record); err != nil {
		return errors.Wrapf(err, "failed to save sync mapping")
	}
	return nil
}

func (s *job) syncExistingEvent(srcEvent *calendar.Event, r syncdb.Record) error {
	log.Printf("existing event: %s\n", r.Src.EventID)

	if srcEvent.Status == "cancelled" || s.shouldExclude(srcEvent) {
		return s.deleteDstEvent(r)
	}

	mappedRecurringEventId, err := s.mapRecurringEventId(srcEvent.RecurringEventId)
	if err != nil {
		return errors.Wrap(err, "failed to map recurring event id")
	}

	if srcEvent.RecurringEventId != "" && mappedRecurringEventId == "" {
		return errors.New("cannot sync recurring event instance when recurring event id mapping not found")
	}

	dstEvent := mapEvent(srcEvent, s.request.MappingOptions)
	dstEvent.RecurringEventId = mappedRecurringEventId

	if err := s.rateLLimiter.Wait(s.ctx); err != nil {
		return err
	}
	if dstEvent, err = s.dstService.Events.Update(s.request.DstCalendarID, r.Dst.EventID, dstEvent).Do(); err != nil {
		return errors.Wrapf(err, "failed to update event")
	}

	log.Printf("updated event: %s\n", srcEvent.Id)
	return nil
}

func (s *job) deleteDstEvent(r syncdb.Record) error {
	log.Printf("delete event: %s\n", r.Src.EventID)
	if err := s.rateLLimiter.Wait(s.ctx); err != nil {
		return err
	}
	return ccommon.DeleteDstEvent(s.syncDB, s.dstService, r)
}

func (s *job) mapRecurringEventId(recurringEventId string) (string, error) {
	r, err := s.syncDB.Find(
		syncdb.Event{
			EventID:      recurringEventId,
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

	recurringEventId, err := s.mapRecurringEventId(srcEvent.RecurringEventId)
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

	if err := s.rateLLimiter.Wait(s.ctx); err != nil {
		return err
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
	if err := s.rateLLimiter.Wait(s.ctx); err != nil {
		return err
	}
	if err := s.dstService.Events.Delete(s.request.DstCalendarID, dstEvent.Id).Do(); err != nil {
		return errors.Wrapf(err, "failed to delete event")
	}

	log.Printf("deleted recurring event instance: %s\n", srcEvent.Id)
	return nil
}

func eventResponseStatus(event *calendar.Event) string {
	for _, attendee := range event.Attendees {
		if !attendee.Self {
			continue
		}
		return attendee.ResponseStatus
	}
	return "unknown"
}
