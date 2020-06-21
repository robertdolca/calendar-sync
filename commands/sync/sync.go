package sync

import (
	"context"
	"flag"
	"fmt"

	"github.com/google/subcommands"
	"github.com/pkg/errors"

	"calendar/clients/calendar"
)

type authListCmd struct {
	sync *calendar.Manager
	srcAccount string
	srcCalendar string
	dstAccount string
	dstCalendar string
}

func New(sync *calendar.Manager) subcommands.Command  {
	return &authListCmd{
		sync: sync,
	}
}

func (*authListCmd) Name() string {
	return "sync"
}

func (*authListCmd) Synopsis() string {
	return "Copies all events from the source calendar to the destination calendar"
}

func (*authListCmd) Usage() string {
	return "calendar sync\n"
}

func (p *authListCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.srcAccount, "src-account", "", "Source account email address")
	f.StringVar(&p.srcCalendar, "src-calendar", "", "Source calendar id")
	f.StringVar(&p.dstAccount, "dst-account", "", "Destination account email address")
	f.StringVar(&p.dstCalendar, "dst-calendar", "", "Destination calendar id")
}

func (p *authListCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if err := p.validateInput(); err != nil {
		fmt.Println(err)
		return subcommands.ExitUsageError
	}

	if err := p.sync.Sync(ctx, p.srcAccount, p.srcCalendar, p.dstAccount, p.dstCalendar); err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

func (p *authListCmd) validateInput() error {
	if p.srcAccount == "" {
		return errors.New("source account email not specified")
	}
	if p.srcCalendar == "" {
		return errors.New("source calendar id not specified")
	}
	if p.dstAccount == "" {
		return errors.New("destination account email not specified")
	}
	if p.dstCalendar == "" {
		return errors.New("destination calendar id not specified")
	}
	return nil
}
