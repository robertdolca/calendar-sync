package auth

import (
	"context"
	"flag"
	"fmt"

	"github.com/google/subcommands"

	"calendar/clients/tmanager"
)

type authAddCmd struct {
	tokenManager *tmanager.Manager
}

func New(tokenManager *tmanager.Manager) subcommands.Command  {
	return &authAddCmd{
		tokenManager: tokenManager,
	}
}

func (*authAddCmd) Name() string {
	return "auth"
}

func (*authAddCmd) Synopsis() string {
	return "Authenticated a new account"
}

func (*authAddCmd) Usage() string {
	return ``
}

func (a *authAddCmd) SetFlags(*flag.FlagSet) {}

func (a *authAddCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if err := a.tokenManager.Auth(ctx); err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}
