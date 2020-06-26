package auth

import (
	"context"
	"flag"
	"fmt"

	"github.com/google/subcommands"

	"github.com/robertdolca/calendar-sync/clients/tmanager"
)

type auth struct {
	tokenManager *tmanager.Manager
}

func New(tokenManager *tmanager.Manager) subcommands.Command {
	return &auth{
		tokenManager: tokenManager,
	}
}

func (*auth) Name() string {
	return "auth"
}

func (*auth) Synopsis() string {
	return "Authenticated a new account"
}

func (*auth) Usage() string {
	return ``
}

func (a *auth) SetFlags(*flag.FlagSet) {}

func (a *auth) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if err := a.tokenManager.Auth(ctx); err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}
