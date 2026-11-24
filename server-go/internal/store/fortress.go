package store

import (
	"cmp"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// The fortress keeps two documents, both sealed by the caller: this
// instance's membership and the kit's own state. The log notes what happened
// in the fortress, and the shares that ran out, whose rows are gone, beside
// the shares and requests the other tables hold.
func applyFortressSchema(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS fortress (
			name  TEXT PRIMARY KEY,
			value BLOB NOT NULL
		);
		CREATE TABLE IF NOT EXISTS fortress_log (
			id     INTEGER PRIMARY KEY AUTOINCREMENT,
			at     INTEGER NOT NULL,
			code   TEXT NOT NULL,
			app    TEXT NOT NULL DEFAULT '',
			device TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE IF NOT EXISTS expired_shares (
			id   INTEGER PRIMARY KEY AUTOINCREMENT,
			at   INTEGER NOT NULL,
			size INTEGER NOT NULL
		);
	`); err != nil {
		return fmt.Errorf("create fortress tables: %w", err)
	}
	return nil
}

// FortressDoc returns the document name, or nil when there is none.
func FortressDoc(db *sql.DB, name string) ([]byte, error) {
	var value []byte
	err := db.QueryRow(`SELECT value FROM fortress WHERE name = ?`, name).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read fortress %s: %w", name, err)
	}
	return value, nil
}

// SaveFortressDoc replaces the document name.
func SaveFortressDoc(db *sql.DB, name string, value []byte) error {
	if _, err := db.Exec(`INSERT INTO fortress (name, value) VALUES (?, ?)
		ON CONFLICT (name) DO UPDATE SET value = excluded.value`, name, value); err != nil {
		return fmt.Errorf("save fortress %s: %w", name, err)
	}
	return nil
}

// DropFortressDoc forgets the document name.
func DropFortressDoc(db *sql.DB, name string) error {
	if _, err := db.Exec(`DELETE FROM fortress WHERE name = ?`, name); err != nil {
		return fmt.Errorf("drop fortress %s: %w", name, err)
	}
	return nil
}

// FortressEvent is a line of the fortress's own log. App and Device name the
// other member it concerns, empty for none.
type FortressEvent struct {
	At     int64 // unix ms
	Code   string
	App    string
	Device string
}

// NoteExpired notes in the log that a share of size bytes ran out at at, by
// its expiry or its last download.
func NoteExpired(db *sql.DB, at, size int64) error {
	if _, err := db.Exec(`INSERT INTO expired_shares (at, size) VALUES (?, ?)`, at, size); err != nil {
		return fmt.Errorf("note an expired share: %w", err)
	}
	return nil
}

// AddFortressEvent notes e in the log.
func AddFortressEvent(db *sql.DB, e FortressEvent) error {
	if _, err := db.Exec(`INSERT INTO fortress_log (at, code, app, device) VALUES (?, ?, ?, ?)`,
		e.At, e.Code, e.App, e.Device); err != nil {
		return fmt.Errorf("add fortress event: %w", err)
	}
	return nil
}

// The kinds of LogRow.
const (
	LogShare    = "share"
	LogRequest  = "request"
	LogReceived = "received"
	LogExpired  = "expired"
	LogFortress = "fortress"
)

// LogRow is one line of what this instance did and still keeps: a share
// made or run out, a file request opened, a file that arrived for one, or an
// event of the fortress. Key orders the rows of one moment and names the row for as
// long as it lasts, without the slug or the file's name, which open a share.
type LogRow struct {
	Key  string
	Kind string
	At   int64 // unix ms
	Size int64
	// Code, App and Device are those of a fortress event.
	Code, App, Device string
}

// logSources each list the rows after a cursor as (key, at, size, code, app,
// device), ordered by at and key. The keys pad the row id, so that they sort
// as the rows were made.
var logSources = []string{
	`SELECT '` + LogShare + `:' || printf('%020d', rowid) AS k, created_at, size, '', '', '' FROM files`,
	`SELECT '` + LogRequest + `:' || printf('%020d', rowid) AS k, created_at, 0, '', '', '' FROM requests`,
	`SELECT '` + LogReceived + `:' || printf('%020d', rowid) AS k, created_at, size, '', '', '' FROM request_uploads`,
	`SELECT '` + LogExpired + `:' || printf('%020d', id) AS k, at AS created_at, size, '', '', '' FROM expired_shares`,
	`SELECT '` + LogFortress + `:' || printf('%020d', id) AS k, at AS created_at, 0, code, app, device FROM fortress_log`,
}

// LogAfter returns up to limit rows that come after the row at and key, the
// oldest first. A zero at and an empty key start at the oldest row.
func LogAfter(db *sql.DB, at int64, key string, limit int) ([]LogRow, error) {
	var out []LogRow
	for _, src := range logSources {
		rows, err := db.Query(`SELECT * FROM (`+src+`) WHERE created_at > ?1 OR (created_at = ?1 AND k > ?2)
			ORDER BY created_at, k LIMIT ?3`, at, key, limit)
		if err != nil {
			return nil, fmt.Errorf("read the log: %w", err)
		}
		for rows.Next() {
			var r LogRow
			if err := rows.Scan(&r.Key, &r.At, &r.Size, &r.Code, &r.App, &r.Device); err != nil {
				rows.Close()
				return nil, fmt.Errorf("read the log: %w", err)
			}
			r.Kind, _, _ = strings.Cut(r.Key, ":")
			out = append(out, r)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, fmt.Errorf("read the log: %w", err)
		}
	}
	slices.SortFunc(out, func(a, b LogRow) int { return cmp.Or(cmp.Compare(a.At, b.At), strings.Compare(a.Key, b.Key)) })
	return out[:min(len(out), limit)], nil
}

// Upload is a tus upload in progress, as the status lists it: Key names it
// without its id, which would let anyone write on.
type Upload struct {
	ID         string
	Key        string
	Size       int64
	ForRequest bool
	CreatedAt  int64 // unix ms
}

// UploadsInProgress lists the uploads not yet stored, the oldest first.
func UploadsInProgress(db *sql.DB) ([]Upload, error) {
	rows, err := db.Query(`SELECT id, rowid, size, request_slug IS NOT NULL, created_at FROM pending_uploads ORDER BY created_at, rowid`)
	if err != nil {
		return nil, fmt.Errorf("list uploads in progress: %w", err)
	}
	defer rows.Close()
	var out []Upload
	for rows.Next() {
		var u Upload
		var row int64
		if err := rows.Scan(&u.ID, &row, &u.Size, &u.ForRequest, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("list uploads in progress: %w", err)
		}
		u.Key = strconv.FormatInt(row, 10)
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list uploads in progress: %w", err)
	}
	return out, nil
}
