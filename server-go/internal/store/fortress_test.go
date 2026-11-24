package store

import (
	"bytes"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

func TestADatabaseOfTheSignInGainsTheFortressAndKeepsItsRows(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "db.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE schema_version (version INTEGER NOT NULL); INSERT INTO schema_version VALUES (2)`); err != nil {
		t.Fatal(err)
	}
	for _, m := range migrations[:2] {
		if err := m(db); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO files (id, slug, original_name, size, created_at) VALUES ('id1', 'slug1', '', 1, 1)`); err != nil {
		t.Fatal(err)
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
	if doc, err := FortressDoc(db, "membership"); err != nil || doc != nil {
		t.Errorf("an upgraded install holds a membership: %q, %v", doc, err)
	}
	rows, err := LogAfter(db, 0, "", 10)
	if err != nil || len(rows) != 1 || rows[0].Kind != LogShare {
		t.Errorf("the log after the upgrade = %+v, %v", rows, err)
	}
}

func TestAFortressDocumentIsReplacedAndDropped(t *testing.T) {
	db := openTestDB(t)
	for _, v := range []string{"one", "two"} {
		if err := SaveFortressDoc(db, "membership", []byte(v)); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := FortressDoc(db, "membership"); err != nil || !bytes.Equal(got, []byte("two")) {
		t.Errorf("document = %q, %v, want two", got, err)
	}
	if err := DropFortressDoc(db, "membership"); err != nil {
		t.Fatal(err)
	}
	if got, err := FortressDoc(db, "membership"); err != nil || got != nil {
		t.Errorf("document after dropping = %q, %v", got, err)
	}
}

func TestTheLogPagesThroughSharesRequestsArrivalsAndTheFortressInOrder(t *testing.T) {
	db := openTestDB(t)
	for _, f := range []FileRecord{{ID: "a", Slug: "sa", Size: 10, CreatedAt: 5, Format: 4}, {ID: "b", Slug: "sb", Size: 20, CreatedAt: 5, Format: 4}} {
		if err := CreateFileRecord(db, f); err != nil {
			t.Fatal(err)
		}
	}
	req := newRequest("zq9", nil, nil)
	req.CreatedAt = 3
	if err := CreateRequest(db, req); err != nil {
		t.Fatal(err)
	}
	if ok, err := AddRequestUpload(db, "zq9", upload("u1", 7), 4); err != nil || !ok {
		t.Fatalf("request upload: %v, %v", ok, err)
	}
	if err := AddFortressEvent(db, FortressEvent{At: 6, Code: "joined"}); err != nil {
		t.Fatal(err)
	}

	var kinds []string
	at, key := int64(0), ""
	for range 10 {
		page, err := LogAfter(db, at, key, 2)
		if err != nil {
			t.Fatal(err)
		}
		if len(page) == 0 {
			break
		}
		for _, r := range page {
			kinds = append(kinds, r.Kind)
			if strings.Contains(r.Key, "zq9") || strings.Contains(r.Key, "sa") {
				t.Errorf("the key %q names a slug", r.Key)
			}
		}
		last := page[len(page)-1]
		at, key = last.At, last.Key
	}
	want := "request received share share fortress"
	if got := strings.Join(kinds, " "); got != want {
		t.Errorf("the log runs %q, want %q", got, want)
	}
}

func TestUploadsInProgressAreListedWithoutTheirIDsInTheirKeys(t *testing.T) {
	db := openTestDB(t)
	for _, up := range []PendingUpload{{ID: "tus-a", Size: 30, CreatedAt: 2}, {ID: "tus-b", Size: 5, RequestSlug: "r", CreatedAt: 1}} {
		if err := TrackPendingUpload(db, up); err != nil {
			t.Fatal(err)
		}
	}
	got, err := UploadsInProgress(db)
	if err != nil || len(got) != 2 {
		t.Fatalf("uploads = %+v, %v", got, err)
	}
	if got[0].ID != "tus-b" || !got[0].ForRequest || got[1].ForRequest || got[1].Size != 30 {
		t.Errorf("uploads = %+v", got)
	}
	for _, u := range got {
		if strings.Contains(u.Key, "tus") {
			t.Errorf("the key %q names the upload", u.Key)
		}
	}
}
