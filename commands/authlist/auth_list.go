package authlist

import (
	"calendar/clients/userinfo"
	"context"
	"flag"
	"fmt"
	"github.com/google/subcommands"
)

type authListCmd struct {
	userInfo *userinfo.Manager
}

func New(userInfo *userinfo.Manager) subcommands.Command  {
	return &authListCmd{
		userInfo: userInfo,
	}
}

func (*authListCmd) Name() string {
	return "auth-list"
}

func (*authListCmd) Synopsis() string {
	return "List authenticated accounts"
}

func (*authListCmd) Usage() string {
	return ``
}

func (p *authListCmd) SetFlags(*flag.FlagSet) {}

func (p *authListCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	emails, err := p.userInfo.AllEmails()
	if err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}
	for _, email := range emails {
		fmt.Println(email)
	}
	return subcommands.ExitSuccess
}
