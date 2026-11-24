package cleanup

import (
	"testing"
	"time"

	"github.com/junkerderprovinz/castleparcel/server-go/internal/store"
)

func TestAnExpiredShareStaysInTheLog(t *testing.T) {
	cfg, db := newTestEnv(t)
	addShare(t, cfg, db, "expired", 100, i64p(9_000))
	addShare(t, cfg, db, "future", 400, i64p(20_000))

	if _, err := Sweep(cfg, db, time.UnixMilli(10_000)); err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	rows, err := store.LogAfter(db, 0, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	var expired []store.LogRow
	for _, r := range rows {
		if r.Kind == store.LogExpired {
			expired = append(expired, r)
		}
	}
	if len(expired) != 1 || expired[0].Size != 100 || expired[0].At != 10_000 {
		t.Errorf("expired lines = %+v, want one of 100 bytes at 10000", expired)
	}
}
