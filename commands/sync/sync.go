package sync

import (
	"context"
	"flag"
	"fmt"
	"regexp"
	"time"

	"github.com/google/subcommands"
	"github.com/pkg/errors"

	"github.com/robertdolca/calendar-sync/clients/calendar"
	"github.com/robertdolca/calendar-sync/clients/calendar/sync"
)

type syncCmd struct {
	sync                *calendar.Manager
	srcAccountEmail     string
	srcCalendarID       string
	dstAccountEmail     string
	dstCalendarID       string
	copyDescription     bool
	copyLocation        bool
	copyColor           bool
	includeTentative    bool
	includeNotGoing     bool
	includeNotResponded bool
	includeOutOfOffice  bool
	titleOverride       string
	visibility          string
	excludeTitleRegex   string
	syncInterval        time.Duration
}

func New(syncManager *calendar.Manager) subcommands.Command {
	return &syncCmd{
		sync: syncManager,
	}
}

func (*syncCmd) Name() string {
	return "sync"
}

func (*syncCmd) Synopsis() string {
	return "Copies all events from the source calendar to the destination calendar"
}

func (*syncCmd) Usage() string {
	return "calendar sync\n"
}

func (p *syncCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.srcAccountEmail, "src-account", "", "Source account email address")
	f.StringVar(&p.srcCalendarID, "src-calendar", "", "Source calendar id")
	f.StringVar(&p.dstAccountEmail, "dst-account", "", "Destination account email address")
	f.StringVar(&p.dstCalendarID, "dst-calendar", "", "Destination calendar id")
	f.StringVar(&p.titleOverride, "title-override", "", "Is specified the title of all events will be replaced by this")
	f.StringVar(&p.visibility, "visibility", "default", "Event visibility (options: default / public / private)")
	f.StringVar(&p.excludeTitleRegex, "exclude-title-regex", "", "Am optional regular expression to exclude events when the title matches")

	f.BoolVar(&p.copyDescription, "copy-description", false, "Copy the event description (default: false)")
	f.BoolVar(&p.copyLocation, "copy-location", false, "Copy the event location (default: false)")
	f.BoolVar(&p.copyColor, "copy-color", false, "Copy the event color (default: false)")
	f.BoolVar(&p.includeTentative, "include-tentative", false, "Copy events RSVP'ed as Maybe (default: false)")
	f.BoolVar(&p.includeNotGoing, "include-not-going", false, "Copy events RSVP'ed as No (default: false)")
	f.BoolVar(&p.includeNotResponded, "include-not-responded", false, "Copy events without RSVP response (default: false)")
	f.BoolVar(&p.includeOutOfOffice, "include-out-of-office", false, "Copy out of office events (default: false)")

	f.DurationVar(&p.syncInterval, "interval", time.Hour, "The time window to look back for calendar changes (3h, 5d)")
}

func (p *syncCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if err := p.validateInput(); err != nil {
		fmt.Println(err)
		return subcommands.ExitUsageError
	}

	var excludeTitleRegex *regexp.Regexp
	if p.excludeTitleRegex != "" {
		var err error
		excludeTitleRegex, err = regexp.Compile(p.excludeTitleRegex)
		if err != nil {
			fmt.Println(fmt.Sprintf("regular expression comillation error: %s", err))
			return subcommands.ExitUsageError
		}
	}

	request := sync.Request{
		SrcAccountEmail:     p.srcAccountEmail,
		SrcCalendarID:       p.srcCalendarID,
		DstAccountEmail:     p.dstAccountEmail,
		DstCalendarID:       p.dstCalendarID,
		SyncInterval:        p.syncInterval,
		IncludeTentative:    p.includeTentative,
		IncludeNotGoing:     p.includeNotGoing,
		IncludeNotResponded: p.includeNotResponded,
		ExcludeTitleRegex:   excludeTitleRegex,
		IncludeOutOfOffice:  p.includeOutOfOffice,
		MappingOptions: sync.MappingOptions{
			CopyDescription: p.copyDescription,
			CopyLocation:    p.copyLocation,
			CopyColor:       p.copyColor,
			TitleOverride:   p.titleOverride,
			Visibility:      p.visibility,
		},
	}

	if err := p.sync.Sync(ctx, request); err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

func validateVisibility(visibility string) error {
	if visibility == "public" || visibility == "private" || visibility == "default" {
		return nil
	}
	return errors.Errorf("invalid visibility: %s", visibility)
}

func (p *syncCmd) validateInput() error {
	if p.srcAccountEmail == "" {
		return errors.New("source account email not specified")
	}
	if p.srcCalendarID == "" {
		return errors.New("source calendar id not specified")
	}
	if p.dstAccountEmail == "" {
		return errors.New("destination account email not specified")
	}
	if p.dstCalendarID == "" {
		return errors.New("destination calendar id not specified")
	}
	if err := validateVisibility(p.visibility); err != nil {
		return err
	}
	return nil
}
