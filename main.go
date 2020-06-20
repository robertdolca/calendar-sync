package main

import (
	"calendar/clients/tmanager"
	"calendar/clients/userinfo"
	"calendar/commands/authadd"
	"calendar/commands/authlist"
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

	ui, err := userinfo.New(tm)
	if err != nil {
		return errors.Wrap(err, "failed to create user info")
	}

	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(authadd.New(tm), "")
	subcommands.Register(authlist.New(ui), "")
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
