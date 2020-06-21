package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/google/subcommands"
	"github.com/pkg/errors"
	"golang.org/x/net/context"

	"calendar/clients/calendar"
	"calendar/clients/syncdb"
	"calendar/clients/tmanager"
	"calendar/clients/userinfo"
	"calendar/commands/auth"
	"calendar/commands/clear"
	"calendar/commands/list"
	synccmd "calendar/commands/sync"
)

func run() subcommands.ExitStatus {
	tm, err := tmanager.New()
	if err != nil {
		fmt.Println(errors.Wrap(err, "failed to create token manager"))
		return subcommands.ExitFailure
	}

	sdb, err := syncdb.New()
	if err != nil {
		fmt.Println(errors.Wrap(err, "failed to create sync records database"))
		return subcommands.ExitFailure
	}
	defer func() {
		err := sdb.Close()
		if err != nil {
			fmt.Println(errors.Wrap(err, "failed to close db connection"))
		}
	}()

	ui := userinfo.New(tm)
	cm := calendar.New(tm, ui, sdb)

	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(auth.New(tm), "")
	subcommands.Register(list.New(cm), "")
	subcommands.Register(synccmd.New(cm), "")
	subcommands.Register(clear.New(cm), "")
	flag.Parse()

	return subcommands.Execute(context.Background())
}

func main() {
	os.Exit(int(run()))
}
