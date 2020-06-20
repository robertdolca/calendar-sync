package authadd

import (
	"calendar/clients/tmanager"
	"context"
	"flag"
	"fmt"
	"github.com/google/subcommands"
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
	return "auth-add"
}

func (*authAddCmd) Synopsis() string {
	return "Authenticated a new account"
}

func (*authAddCmd) Usage() string {
	return ``
}

func (a *authAddCmd) SetFlags(*flag.FlagSet) {}

func (a *authAddCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if err := a.tokenManager.Auth(); err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}
