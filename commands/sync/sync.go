package sync

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/google/subcommands"
	"github.com/pkg/errors"

	"calendar/clients/calendar"
)

type sync struct {
	sync            *calendar.Manager
	srcAccountEmail string
	srcCalendarID   string
	dstAccountEmail string
	dstCalendarID   string
	syncInterval    time.Duration
}

func New(syncManager *calendar.Manager) subcommands.Command  {
	return &sync{
		sync: syncManager,
	}
}

func (*sync) Name() string {
	return "sync"
}

func (*sync) Synopsis() string {
	return "Copies all events from the source calendar to the destination calendar"
}

func (*sync) Usage() string {
	return "calendar sync\n"
}

func (p *sync) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.srcAccountEmail, "src-account", "", "Source account email address")
	f.StringVar(&p.srcCalendarID, "src-calendar", "", "Source calendar id")
	f.StringVar(&p.dstAccountEmail, "dst-account", "", "Destination account email address")
	f.StringVar(&p.dstCalendarID, "dst-calendar", "", "Destination calendar id")
	f.DurationVar(&p.syncInterval, "interval", time.Hour, "The time window to look back for calendar changes (3h, 5d)")
}

func (p *sync) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if err := p.validateInput(); err != nil {
		fmt.Println(err)
		return subcommands.ExitUsageError
	}

	if err := p.sync.Sync(ctx, p.srcAccountEmail, p.srcCalendarID, p.dstAccountEmail, p.dstCalendarID, p.syncInterval); err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

func (p *sync) validateInput() error {
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
	return nil
}
