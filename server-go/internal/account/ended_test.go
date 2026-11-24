package account

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/junkerderprovinz/parleyport/kit/signin"

	"github.com/junkerderprovinz/castleparcel/server-go/internal/store"
)

func TestAnAccountShutOutLosesItsRequests(t *testing.T) {
	for what, closes := range map[signin.Ending]bool{
		signin.EndBlocked: true, signin.EndRemoved: true, signin.EndRole: true, signin.EndRefused: true,
		signin.EndRevoked: true, signin.EndReset: true, signin.EndResetOther: true,
		signin.EndPassword: false, signin.EndEverywhere: false, signin.EndTwoFactor: false,
	} {
		db, err := store.Open(filepath.Join(t.TempDir(), "db.sqlite"))
		if err != nil {
			t.Fatal(err)
		}
		rec := store.RequestRecord{Slug: "r1", PublicKey: "k", CollectTokenHash: "h", CreatedAt: 1, Account: "acc1"}
		if err := store.CreateRequest(db, rec); err != nil {
			t.Fatal(err)
		}
		if err := ended(db)(context.Background(), "acc1", what); err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		got, _ := store.GetRequest(db, "r1")
		if got.Expired(2) != closes {
			t.Errorf("%s: request closed = %v, want %v", what, got.Expired(2), closes)
		}
		db.Close()
	}
}
