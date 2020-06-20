package userinfo

import (
	"calendar/clients/tmanager"
	"context"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	goauth2 "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

type Manager struct {
	tokenManager *tmanager.Manager
}

func New(tokenManager *tmanager.Manager) *Manager {
	return &Manager{
		tokenManager: tokenManager,
	}
}

func (i *Manager) Email(token *oauth2.Token) (string, error) {
	config := i.tokenManager.Config()

	email, err := userEmail(context.Background(), config, token)
	if err != nil {
		return "", errors.Wrap(err, "failed to get user email")
	}

	return email, nil
}

func userEmail(ctx context.Context, config *oauth2.Config, token *oauth2.Token) (string, error) {
	authService, err := goauth2.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx, token)))
	if err != nil {
		return "", errors.Wrap(err, "failed to create ouath2 client")
	}

	info, err := authService.Userinfo.Get().Do()
	if err != nil {
		return "", errors.Wrap(err, "user info call failed")
	}

	return info.Email, nil
}
