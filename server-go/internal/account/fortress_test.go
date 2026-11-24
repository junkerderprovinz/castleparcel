package account

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/junkerderprovinz/castleparcel/server-go/internal/store"
)

func TestAnAccountShutOutLeavesTheFortressMembership(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := store.SaveFortressDoc(db, "membership", []byte("sealed")); err != nil {
		t.Fatal(err)
	}
	for what := range shutOut {
		if err := ended(db, nil)(context.Background(), "acc1", what); err != nil {
			t.Fatalf("%s: %v", what, err)
		}
	}
	if got, err := store.FortressDoc(db, "membership"); err != nil || !bytes.Equal(got, []byte("sealed")) {
		t.Errorf("membership after the account was shut out = %q, %v", got, err)
	}
}
