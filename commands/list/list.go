package list

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/subcommands"
	"github.com/jedib0t/go-pretty/v6/table"

	"calendar/clients/calendar"
)

type authListCmd struct {
	calendarManager *calendar.Manager
}

func New(calendarManager *calendar.Manager) subcommands.Command  {
	return &authListCmd{
		calendarManager: calendarManager,
	}
}

func (*authListCmd) Name() string {
	return "list"
}

func (*authListCmd) Synopsis() string {
	return "List authenticated accounts and the calendars they have access to"
}

func (*authListCmd) Usage() string {
	return ``
}

func (p *authListCmd) SetFlags(*flag.FlagSet) {}

func (p *authListCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	usersCalendars, err := p.calendarManager.UsersCalendars(ctx)
	if err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}
	for _, userCalendars := range usersCalendars {
		fmt.Println(userCalendars.Email)
		fmt.Println()

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Summary", "ID"})

		for _, cal := range userCalendars.Calendars {
			if cal.Deleted {
				continue
			}
			t.AppendRow([]interface{}{cal.Summary, cal.Id})
			t.AppendSeparator()
		}

		t.Render()
		fmt.Println()
	}
	return subcommands.ExitSuccess
}


