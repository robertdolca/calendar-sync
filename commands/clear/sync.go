package clear

import (
	"context"
	"flag"
	"fmt"

	"github.com/google/subcommands"
	"github.com/pkg/errors"

	"github.com/robertdolca/calendar-sync/clients/calendar"
)

type clearCmd struct {
	sync *calendar.Manager
	account string
	calendar string
}

func New(sync *calendar.Manager) subcommands.Command  {
	return &clearCmd{
		sync: sync,
	}
}

func (*clearCmd) Name() string {
	return "clear"
}

func (*clearCmd) Synopsis() string {
	return "Remove all synced events from a calendar"
}

func (*clearCmd) Usage() string {
	return "calendar clear\n"
}

func (p *clearCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.account, "account", "", "Account email address")
	f.StringVar(&p.calendar, "calendar", "", "Calendar id")
}

func (p *clearCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if err := p.validateInput(); err != nil {
		fmt.Println(err)
		return subcommands.ExitUsageError
	}

	if err := p.sync.Clear(ctx, p.account, p.calendar); err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

func (p *clearCmd) validateInput() error {
	if p.account == "" {
		return errors.New("account email not specified")
	}
	if p.calendar == "" {
		return errors.New("calendar id not specified")
	}
	return nil
}
