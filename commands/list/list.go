package list

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/subcommands"
	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/robertdolca/calendar-sync/clients/calendar"
)

type list struct {
	calendarManager *calendar.Manager
}

func New(calendarManager *calendar.Manager) subcommands.Command  {
	return &list{
		calendarManager: calendarManager,
	}
}

func (*list) Name() string {
	return "list"
}

func (*list) Synopsis() string {
	return "List authenticated accounts and the calendars they have access to"
}

func (*list) Usage() string {
	return ``
}

func (p *list) SetFlags(*flag.FlagSet) {}

func (p *list) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
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


