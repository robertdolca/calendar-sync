package sync

import (
	"calendar/clients/calendar"
	"context"
	"flag"
	"fmt"
	"github.com/google/subcommands"
)

type authListCmd struct {
	sync *calendar.Manager
}

func New(sync *calendar.Manager) subcommands.Command  {
	return &authListCmd{
		sync: sync,
	}
}

func (*authListCmd) Name() string {
	return "calendar-sync"
}

func (*authListCmd) Synopsis() string {
	return "Copies all events from the source calendar to the destination calendar"
}

func (*authListCmd) Usage() string {
	return ``
}

func (p *authListCmd) SetFlags(*flag.FlagSet) {}

func (p *authListCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if err := p.sync.Sync("", "", "", ""); err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}
