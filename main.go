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

func setup() error {
	tm, err := tmanager.New()
	if err != nil {
		return errors.Wrap(err, "failed to create token manager")
	}

	sdb, err := syncdb.New()
	if err != nil {
		return errors.Wrap(err, "failed to create sync records database")
	}

	ui := userinfo.New(tm)
	cm := calendar.New(tm, ui, sdb)

	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(auth.New(tm), "")
	subcommands.Register(list.New(cm), "")
	subcommands.Register(synccmd.New(cm), "")
	subcommands.Register(clear.New(cm), "")
	flag.Parse()

	return nil
}

func main() {
	if err := setup(); err != nil {
		fmt.Println(err)
		os.Exit(int(subcommands.ExitFailure))
	}
	os.Exit(int(subcommands.Execute(context.Background())))
}
