// Package store owns the SQLite metadata database. It uses modernc.org/sqlite,
// which needs no cgo, so the server builds as a static binary.
package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/junkerderprovinz/parleyport/kit/signin/sqlstore"
	_ "modernc.org/sqlite"
)

// SignInTable keeps the documents of the owner's sign-in.
const SignInTable = "signin_docs"

// migrations bring a database from one version to the next, the first from a
// database of a CastleParcel that kept no version. Each one is safe to run
// again, so a start cut off between a migration and its version runs it once
// more.
var migrations = []func(*sql.DB) error{
	applyBase,
	func(db *sql.DB) error {
		_, err := db.Exec(sqlstore.Schema(SignInTable))
		return err
	},
	func(db *sql.DB) error {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('requests') WHERE name = 'account'`).Scan(&n); err != nil || n > 0 {
			return err
		}
		_, err := db.Exec(`ALTER TABLE requests ADD COLUMN account TEXT`)
		return err
	},
	applyFortressSchema,
}

// SchemaVersion is the version of the databases this server writes.
var SchemaVersion = len(migrations)

// Open opens or creates the database at dbPath and applies the schema.
//
// SQLite allows one writer at a time, and without a busy timeout a second
// writer fails at once with "database is locked", which showed up as spurious
// 404s from the download count under overlapping downloads. WAL keeps readers
// from blocking the writer, busy_timeout makes a contending writer wait, and a
// single pooled connection serializes the writes.
func Open(dbPath string) (*sql.DB, error) {
	dsn := "file:" + dbPath +
		"?_pragma=busy_timeout(5000)" +
		"&_pragma=journal_mode(WAL)" +
		"&_pragma=foreign_keys(on)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", dbPath, err)
	}
	db.SetMaxOpenConns(1)
	if err := ApplySchema(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// ApplySchema brings the database to SchemaVersion. A database a newer
// CastleParcel wrote stops the start, since this one would not know what its
// tables mean.
func ApplySchema(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`); err != nil {
		return fmt.Errorf("create schema_version: %w", err)
	}
	var at int
	err := db.QueryRow(`SELECT version FROM schema_version`).Scan(&at)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if _, err := db.Exec(`INSERT INTO schema_version (version) VALUES (0)`); err != nil {
			return fmt.Errorf("start schema_version: %w", err)
		}
	case err != nil:
		return fmt.Errorf("read schema_version: %w", err)
	}
	if at > SchemaVersion {
		return fmt.Errorf("the database is version %d and this CastleParcel knows up to %d: start the newer CastleParcel again", at, SchemaVersion)
	}
	for v := at; v < SchemaVersion; v++ {
		if err := migrations[v](db); err != nil {
			return fmt.Errorf("migrate the database to version %d: %w", v+1, err)
		}
		if _, err := db.Exec(`UPDATE schema_version SET version = ?`, v+1); err != nil {
			return fmt.Errorf("record version %d: %w", v+1, err)
		}
	}
	return nil
}

// applyBase creates the tables and indexes of the CastleParcel that kept no
// version and adds each later column of files when it is missing.
func applyBase(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS files (
			id             TEXT PRIMARY KEY,
			slug           TEXT UNIQUE NOT NULL,
			original_name  TEXT NOT NULL,
			size           INTEGER NOT NULL,
			mime           TEXT,
			password_hash  TEXT,
			expires_at     INTEGER,
			created_at     INTEGER NOT NULL,
			download_count INTEGER NOT NULL DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_files_expires_at ON files (expires_at);
	`); err != nil {
		return fmt.Errorf("create files table: %w", err)
	}

	cols, err := fileColumns(db)
	if err != nil {
		return err
	}

	addColumn := func(name, ddl string) error {
		if _, ok := cols[name]; ok {
			return nil
		}
		if _, err := db.Exec("ALTER TABLE files ADD COLUMN " + ddl); err != nil {
			return fmt.Errorf("add column %s: %w", name, err)
		}
		return nil
	}

	migrations := []struct{ name, ddl string }{
		{"encrypted", "encrypted INTEGER NOT NULL DEFAULT 0"},
		{"enc_mode", "enc_mode TEXT"},
		{"enc_key_wrapped", "enc_key_wrapped TEXT"},
		{"max_downloads", "max_downloads INTEGER"},
		// Rows from before browser encryption are format 1.
		{"format", "format INTEGER NOT NULL DEFAULT 1"},
		{"wrapped_key", "wrapped_key BLOB"},
		{"kdf_salt", "kdf_salt BLOB"},
		{"key_verifier", "key_verifier TEXT"},
		{"manage_token_hash", "manage_token_hash TEXT"},
		{"verifier_failures", "verifier_failures INTEGER NOT NULL DEFAULT 0"},
		{"verifier_locked_until", "verifier_locked_until INTEGER"},
	}
	for _, m := range migrations {
		if err := addColumn(m.name, m.ddl); err != nil {
			return err
		}
	}

	// A counted download's resume token, by hash. until is when it runs out
	// unless used again, deadline the latest it may ever run out.
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS resumes (
			token_hash TEXT PRIMARY KEY,
			file_id    TEXT NOT NULL REFERENCES files (id) ON DELETE CASCADE,
			until      INTEGER NOT NULL,
			deadline   INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_resumes_file_id ON resumes (file_id);
	`); err != nil {
		return fmt.Errorf("create resumes table: %w", err)
	}
	if err := applyRequestSchema(db); err != nil {
		return err
	}
	return applyPendingSchema(db)
}

func fileColumns(db *sql.DB) (map[string]struct{}, error) {
	rows, err := db.Query("PRAGMA table_info(files)")
	if err != nil {
		return nil, fmt.Errorf("pragma table_info(files): %w", err)
	}
	defer rows.Close()

	cols := make(map[string]struct{})
	for rows.Next() {
		var (
			cid       int
			name      string
			typ       string
			notNull   int
			dfltValue sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dfltValue, &pk); err != nil {
			return nil, fmt.Errorf("scan table_info: %w", err)
		}
		cols[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate table_info: %w", err)
	}
	return cols, nil
}
