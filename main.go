package main

import (
	"calendar/clients/calendar"
	"calendar/clients/tmanager"
	"calendar/clients/userinfo"
	"calendar/commands/auth"
	"calendar/commands/list"
	synccmd "calendar/commands/sync"
	"flag"
	"fmt"
	"github.com/google/subcommands"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"os"
)

func setup() error {
	tm, err := tmanager.New()
	if err != nil {
		return errors.Wrap(err, "failed to create token manager")
	}

	ui := userinfo.New(tm)
	cm := calendar.New(tm, ui)

	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(auth.New(tm), "")
	subcommands.Register(list.New(cm), "")
	subcommands.Register(synccmd.New(cm), "")
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
