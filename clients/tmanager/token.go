package tmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/nightlyone/lockfile"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	goauth2 "google.golang.org/api/oauth2/v2"

	"github.com/robertdolca/calendar-sync/clients/lockhelper"
)

const (
	tokensPath = "tokens.json"
	credentialsPath = "credentials.json"
)

var (
	scope = []string{
		calendar.CalendarReadonlyScope,
		calendar.CalendarEventsScope,
		goauth2.UserinfoEmailScope,
	}
)

type Manager struct {
	tokens []oauth2.Token
	config oauth2.Config
	tokensMutex lockfile.Lockfile
}

func New() (*Manager, error) {
	tokensMutex, err := lockfile.New(lockhelper.FilePath("tokens.lock"))
	if err != nil {
		return nil, err
	}

	tokens, err := readTokens(tokensMutex)
	if err != nil {
		return nil, err
	}

	config, err := readConfig()
	if err != nil {
		return nil, err
	}

	return &Manager{
		tokens: tokens,
		config: *config,
		tokensMutex: tokensMutex,
	}, nil
}

func (m *Manager) List() []oauth2.Token {
	return m.tokens
}

func (m *Manager) add(token oauth2.Token) error {
	tokens := append([]oauth2.Token{}, m.tokens...)
	tokens = append(tokens, token)
	if err := saveTokens(m.tokensMutex, tokens); err != nil {
		return err
	}
	m.tokens = tokens
	return nil
}

func (m *Manager) Config() *oauth2.Config {
	return &m.config
}

// GetTokenFromWeb requests a token from the web, then returns the retrieved token.
func (m *Manager) Auth(ctx context.Context, ) error {
	config := m.Config()

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return errors.Wrap(err, "unable to readTokens authorization code")
	}

	token, err := config.Exchange(ctx, authCode)
	if err != nil {
		return errors.Wrap(err, "Unable to retrieve token from web")
	}

	return m.add(*token)
}

func readConfig() (*oauth2.Config, error) {
	credentials, err := ioutil.ReadFile(credentialsPath)
	if err != nil {
		return nil, errors.Wrap(err, "unable to readTokens client secret file")
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(credentials, scope...)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse client secret file to config")
	}

	return config, nil
}

// Retrieves a token from a local file.
func readTokens(mutex lockfile.Lockfile) ([]oauth2.Token, error) {
	err := mutex.TryLock()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to acquire tokens file lock")
	}

	tokens, err := readTokenUnsafe()
	return tokens, lockhelper.MutexUnlock(mutex, err)
}

func readTokenUnsafe() ([]oauth2.Token, error) {
	tokens := make([]oauth2.Token, 0)

	file, err := os.Open(tokensPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if err = json.NewDecoder(file).Decode(&tokens); err != nil {
		return nil, err
	}

	return tokens, file.Close()
}

// Saves a token to a file tokensPath.
func saveTokens(mutex lockfile.Lockfile, tokens []oauth2.Token) error {
	err := mutex.TryLock()
	if err != nil {
		return errors.Wrapf(err, "failed to acquire tokens file lock")
	}
	return lockhelper.MutexUnlock(mutex, saveTokensUnsafe(tokens))
}

func saveTokensUnsafe(tokens []oauth2.Token) error {
	file, err := os.OpenFile(tokensPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return errors.Wrapf(err, "unable to open tokens file in write mode")
	}

	if err := json.NewEncoder(file).Encode(tokens); err != nil {
		return errors.Wrapf(err, "unable to save oauth token")
	}

	return file.Close()
}
