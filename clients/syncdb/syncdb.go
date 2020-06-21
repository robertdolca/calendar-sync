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
	EventID      string
	AccountEmail string
	CalendarID   string
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
		key := buildKeyRecord(r)

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

func (db *DB) Find(e Event, dstAccountEmail, dstCalendarID string) (Record, error) {
	var r Record

	err := db.db.View(func(txn *badger.Txn) error {
		key := buildKey(e, dstAccountEmail, dstCalendarID)

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

func (db *DB) ListDst(accountEmail, calendarID string) ([]Record, error) {
	var result []Record

	err := db.db.View(func(txn *badger.Txn) error {

		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()

			data, err := item.ValueCopy(nil)
			if err != nil {
				return errors.Wrap(err, "failed to read record into buffer")
			}

			var record Record
			if err := json.Unmarshal(data, &record); err != nil {
				return errors.Wrap(err, "failed to serialize record")
			}

			if record.Dst.CalendarID != calendarID || record.Dst.AccountEmail != accountEmail {
				continue
			}

			result = append(result, record)
		}
		return nil
	})

	return result, err
}

func (db *DB) Delete(r Record) error {
	return db.db.Update(func(txn *badger.Txn) error {
		if err := txn.Delete(buildKeyRecord(r)); err != nil {
			return errors.Wrapf(err, "failed to delete")
		}
		return nil
	})
}

func (db *DB) Close() error {
	return db.db.Close()
}

func buildKey(event Event, dstAccountEmail, dstCalendarId string) []byte {
	return []byte(
		event.AccountEmail + event.CalendarID +
		dstAccountEmail + dstCalendarId +
		event.EventID,
	)
}

func buildKeyRecord(r Record) []byte {
	return buildKey(r.Src, r.Dst.AccountEmail, r.Dst.CalendarID)
}
