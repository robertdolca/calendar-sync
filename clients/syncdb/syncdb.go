package syncdb

import (
	"encoding/json"

	"github.com/dgraph-io/badger/v2"
	"github.com/nightlyone/lockfile"
	"github.com/pkg/errors"

	"calendar/clients/lockhelper"
)

var (
	ErrNotFound = errors.New("record not found")
)

type DB struct {
	fileMutex lockfile.Lockfile
	db *badger.DB
}

type Record struct {
	Src Event
	Dst Event
}

type Event struct {
	Id string
	AccountEmail string
	CalendarId string
}

func New() (*DB, error) {
	db, err := badger.Open(badger.DefaultOptions("sync.db").WithLogger(nil))
	if err != nil {
		return nil, err
	}

	fileMutex, err := lockfile.New(lockhelper.FilePath("syncdb.lock"))
	if err != nil {
		return nil, err
	}

	return &DB{
		fileMutex: fileMutex,
		db: db,
	}, nil
}

func (db *DB) Insert(r Record) error {
	return db.db.Update(func(txn *badger.Txn) error {
		key := buildKey(r.Src)

		value, err := json.Marshal(r)
		if err != nil {
			return errors.Wrap(err, "failed to serialize record")
		}

		if err := txn.SetEntry(badger.NewEntry(key, value)); err != nil {
			return errors.Wrapf(err, "failed to insert")
		}
		return nil
	})
}

func (db *DB) Find(e Event) (Record, error) {
	var r Record

	err := db.db.View(func(txn *badger.Txn) error {
		key := buildKey(e)

		item, err := txn.Get(key)
		if err == badger.ErrKeyNotFound {
			return ErrNotFound
		}
		if err != nil {
			return errors.Wrapf(err, "failed to insert")
		}

		data, err := item.ValueCopy(nil)
		if err != nil {
			return errors.Wrap(err, "failed to read record into buffer")
		}

		if err := json.Unmarshal(data, &r); err != nil {
			return errors.Wrap(err, "failed to serialize record")
		}

		return nil
	})

	return r, err
}

func (db *DB) Delete(e Event) error {
	return db.db.Update(func(txn *badger.Txn) error {
		if err := txn.Delete(buildKey(e)); err != nil {
			return errors.Wrapf(err, "failed to delete")
		}
		return nil
	})
}

func (db *DB) Close() error {
	return db.db.Close()
}

func buildKey(event Event) []byte {
	return []byte(event.AccountEmail + event.CalendarId + event.Id)
}
