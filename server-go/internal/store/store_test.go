package store

import (
	"database/sql"
	"path/filepath"
	"testing"
)

var wantColumns = []string{
	"id", "slug", "original_name", "size", "mime", "password_hash",
	"expires_at", "created_at", "download_count",
	"encrypted", "enc_mode", "enc_key_wrapped",
	"max_downloads",
	"format", "wrapped_key", "kdf_salt",
	"key_verifier",
	"manage_token_hash",
}

func tableColumns(t *testing.T, db *sql.DB) map[string]struct{} {
	t.Helper()
	cols, err := fileColumns(db)
	if err != nil {
		t.Fatalf("fileColumns: %v", err)
	}
	return cols
}

func assertAllColumns(t *testing.T, db *sql.DB) {
	t.Helper()
	cols := tableColumns(t, db)
	for _, name := range wantColumns {
		if _, ok := cols[name]; !ok {
			t.Errorf("missing column %q", name)
		}
	}
}

func assertIndexExists(t *testing.T, db *sql.DB) {
	t.Helper()
	var name string
	err := db.QueryRow(
		"SELECT name FROM sqlite_master WHERE type='index' AND name=?",
		"idx_files_expires_at",
	).Scan(&name)
	if err != nil {
		t.Fatalf("index idx_files_expires_at not found: %v", err)
	}
	if name != "idx_files_expires_at" {
		t.Errorf("index name = %q, want idx_files_expires_at", name)
	}
}

func TestOpenAppliesSchemaIdempotently(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "db.sqlite")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	if err := ApplySchema(db); err != nil {
		t.Fatalf("second ApplySchema: %v", err)
	}

	assertAllColumns(t, db)
	assertIndexExists(t, db)
}

func TestApplySchemaUpgradesOldDatabase(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "db.sqlite")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	// The base table as it was before the first migration.
	if _, err := db.Exec(`
		CREATE TABLE files (
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
	`); err != nil {
		t.Fatalf("create legacy table: %v", err)
	}

	if _, err := db.Exec(
		`INSERT INTO files (id, slug, original_name, size, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		"id1", "slug1", "name.bin", 123, 1000,
	); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	if err := ApplySchema(db); err != nil {
		t.Fatalf("ApplySchema on old DB: %v", err)
	}

	assertAllColumns(t, db)
	assertIndexExists(t, db)

	var (
		slug      string
		encrypted int
		format    int
	)
	if err := db.QueryRow(
		"SELECT slug, encrypted, format FROM files WHERE id=?", "id1",
	).Scan(&slug, &encrypted, &format); err != nil {
		t.Fatalf("read upgraded row: %v", err)
	}
	if slug != "slug1" {
		t.Errorf("slug = %q, want slug1", slug)
	}
	if encrypted != 0 {
		t.Errorf("encrypted default = %d, want 0", encrypted)
	}
	if format != 1 {
		t.Errorf("format default = %d, want 1", format)
	}
}

func version(t *testing.T, db *sql.DB) int {
	t.Helper()
	var v int
	if err := db.QueryRow("SELECT version FROM schema_version").Scan(&v); err != nil {
		t.Fatalf("read schema_version: %v", err)
	}
	return v
}

func TestOpenRecordsTheSchemaVersion(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	if got := version(t, db); got != SchemaVersion {
		t.Errorf("schema_version = %d, want %d", got, SchemaVersion)
	}
	if _, err := db.Exec("SELECT kind, id, data FROM " + SignInTable); err != nil {
		t.Errorf("sign-in table: %v", err)
	}
	var rows int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_version").Scan(&rows); err != nil || rows != 1 {
		t.Errorf("schema_version has %d rows (%v), want 1", rows, err)
	}
}

func TestADatabaseWithoutVersionIsCountedFromTheStart(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "db.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if err := applyBase(db); err != nil {
		t.Fatalf("base schema: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO files (id, slug, original_name, size, created_at) VALUES ('id1', 'slug1', '', 1, 1)`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	db.Close()

	db, err = Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()
	if got := version(t, db); got != SchemaVersion {
		t.Errorf("schema_version = %d, want %d", got, SchemaVersion)
	}
	var slug string
	if err := db.QueryRow("SELECT slug FROM files WHERE id = 'id1'").Scan(&slug); err != nil || slug != "slug1" {
		t.Errorf("share after the upgrade: %q, %v", slug, err)
	}
}

func TestOpenRefusesADatabaseOfANewerVersion(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "db.sqlite")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := db.Exec("UPDATE schema_version SET version = ?", SchemaVersion+1); err != nil {
		t.Fatalf("bump version: %v", err)
	}
	db.Close()

	if db, err := Open(dbPath); err == nil {
		db.Close()
		t.Fatal("Open took a database a newer CastleParcel wrote")
	}
}
