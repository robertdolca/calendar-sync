# calendar-sync

Keep any pair of two Google Calendars in sync.

It can also use a public or a private (shared) calendar as source as long as
the source calendar is added to the Google account's secondary / subscribed
calendars list.

## Installation

```bash
go get github.com/robertdolca/calendar-sync
```

In order to use `calendar-sync` as a command the go bin folder (`~/go/bin`)
needs to be added to the `PATH`.

## Authenticate accounts

```bash
calendar-sync auth
```

This will print link that can be used to authorize access to a Google account.
If the calendars that need to be kept in sync are not owned by the same user
or edit rights are restricted, this step can be repeated for any number of
Google accounts.

## List accounts and calendars

```bash
calendar-sync list
```

This will print a list of accounts that are authorized. For each account, a list
of calendars will be displayed. The calendar list includes read only calendars
and calendars the account is subscribed to.

## Sync calendars

```bash
calendar-sync sync \
  -src-account accountA@gmail.com \
  -src-calendar dj3snc3c \
  -dst-account accountB@custom-domain.com \
  -dst-calendar jab1rgf \
  -interval 2h
```

This will create events on the destination calendar if they are not already
there. If a corresponding event exists it will be updated if necessary.

The sync action looks at events created or updated on the source calendar
within the last 2 hours.

## Delete synced events

```bash
calendar-sync clear \
  -account accountA@gmail.com \
  -calendar dj3snc3c
```

This will remove all synced events regardless of the source calendar.

For this operation to work well it requires the local sync DB to be preserved.

## Internals

### Google API app credentials

The CLI tool requires a Google API app that it will be used in the OAuth 2 flow.
To create an app go to [Developer Console](https://console.developers.google.com/).

Within the Google API app you need to create a
[OAuth 2.0 Client ID](https://console.developers.google.com/apis/credentials).
The application type that should be selected is `Desktop app`.

Once the OAuth 2.0 Client ID is created the JSON credentials file should be
downloaded and placed next to the binary under the name `credentials.json`.

The Google Calendar API needs to be
[enabled](https://console.developers.google.com/apis/api/calendar-json.googleapis.com/overview)
for the Google API app at.

### Local database

The mapping between the source event id and the synced (copy) event id is
maintained next to the binary in a folder called `sync.db`.

Removing the database can lead to duplicate events being created.

### Auth and refresh token

When a new account is authorized the auth token and the refresh token are stored
in a file called `tokens.json` next to the binary.

To remove an authorized account or re-trigger the OAuth flow simply delete the
tokens file.

When the file is read or update a lock file is created `tokens.lock` and it is
cleaned up automatically.
