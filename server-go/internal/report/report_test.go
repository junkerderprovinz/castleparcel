package report

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/junkerderprovinz/parleyport/kit/status"

	"github.com/junkerderprovinz/castleparcel/server-go/internal/store"
)

func testSource(t *testing.T, quota int64) Source {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return Source{
		DB: db, TmpDir: t.TempDir(), Quota: quota,
		App:      status.App{ID: "castleparcel", Name: "CastleParcel", Version: Version, Icon: Icon},
		Instance: func() status.Instance { return status.Instance{ID: "i", Name: "Atlas"} },
		Now:      time.Now,
	}
}

func TestAnUploadInProgressIsWorkWithItsProgress(t *testing.T) {
	s := testSource(t, 0)
	if err := store.TrackPendingUpload(s.DB, store.PendingUpload{ID: "tus1", Size: 4000, CreatedAt: 1000}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.TmpDir, "tus1"), make([]byte, 1000), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := s.Report(context.Background(), "de")
	if err != nil || len(r.Work) != 1 {
		t.Fatalf("report = %+v, %v", r, err)
	}
	w := r.Work[0]
	if w.Kind != status.KindTransfer || w.BlocksRestart || w.Progress == nil || *w.Progress != 0.25 || w.Title != "Upload, 4,0 KB" {
		t.Errorf("work = %+v", w)
	}
	if strings.Contains(w.ID, "tus1") {
		t.Errorf("the task id %q names the upload", w.ID)
	}
	if len(r.NeedsYou) != 0 {
		t.Errorf("needsYou without a quota = %+v", r.NeedsYou)
	}
}

func TestAQuotaAlmostFullNeedsALook(t *testing.T) {
	for _, c := range []struct {
		used int64
		want status.Severity
	}{{80, ""}, {95, status.SeverityWarn}, {100, status.SeverityError}} {
		s := testSource(t, 100)
		if err := store.CreateFileRecord(s.DB, store.FileRecord{ID: "f", Slug: "s", Size: c.used, CreatedAt: 1, Format: 4}); err != nil {
			t.Fatal(err)
		}
		r, err := s.Report(context.Background(), "en")
		if err != nil {
			t.Fatal(err)
		}
		var got status.Severity
		if len(r.NeedsYou) > 0 {
			got = r.NeedsYou[0].Severity
		}
		if got != c.want {
			t.Errorf("%d of 100 bytes: severity %q, want %q (%+v)", c.used, got, c.want, r.NeedsYou)
		}
	}
}

func TestTheLogNamesFortressEventsInTheReadersWords(t *testing.T) {
	s := testSource(t, 0)
	if err := store.AddFortressEvent(s.DB, store.FortressEvent{At: 5, Code: FortressRemoved, App: "ArrowLoop", Device: "Atlas"}); err != nil {
		t.Fatal(err)
	}
	page, err := s.Log(context.Background(), status.LogQuery{Limit: 10, Lang: "de"})
	if err != nil || len(page.Entries) != 1 {
		t.Fatalf("log = %+v, %v", page, err)
	}
	e := page.Entries[0]
	if e.Code != "fortress.removed" || e.Text != "ArrowLoop auf Atlas aus der Festung herausgenommen." {
		t.Errorf("entry = %+v", e)
	}
	again, err := s.Log(context.Background(), status.LogQuery{After: page.Next, Limit: 10})
	if err != nil || len(again.Entries) != 0 || again.Next != page.Next {
		t.Errorf("the log after its end = %+v, %v", again, err)
	}
}

func TestACursorTheLogNeverHandedOutStartsAtTheBeginning(t *testing.T) {
	for _, after := range []string{"", "garbage", "12", "x.share:1"} {
		if at, key := cursor(after); at != 0 || key != "" {
			t.Errorf("cursor(%q) = %d, %q", after, at, key)
		}
	}
}

// The client's language list, read from its source so a language added
// there without texts here fails this test.
func clientLanguages(t *testing.T) []string {
	t.Helper()
	src, err := os.ReadFile("../../../lib/i18n/languages.ts")
	if err != nil {
		t.Fatal(err)
	}
	var codes []string
	for _, m := range regexp.MustCompile(`\{ code: "([a-z]+)"`).FindAllStringSubmatch(string(src), -1) {
		codes = append(codes, m[1])
	}
	if len(codes) != 42 {
		t.Fatalf("found %d languages: %v", len(codes), codes)
	}
	return codes
}

func TestEveryLanguageOfTheClientHasEveryText(t *testing.T) {
	placeholders := regexp.MustCompile(`\{[a-z]+\}`)
	for _, lang := range clientLanguages(t) {
		table, ok := texts[lang]
		if !ok {
			t.Errorf("no texts for %s", lang)
			continue
		}
		for key, en := range texts["en"] {
			text := table[key]
			if text == "" {
				t.Errorf("%s has no %s", lang, key)
				continue
			}
			want, got := placeholders.FindAllString(en, -1), placeholders.FindAllString(text, -1)
			slices.Sort(want)
			slices.Sort(got)
			if !slices.Equal(want, got) {
				t.Errorf("%s.%s has %v, English %v", lang, key, got, want)
			}
			if lang != "en" && text == en && !strings.HasPrefix(key, "work.upload") {
				t.Errorf("%s.%s is still English", lang, key)
			}
		}
	}
	if len(texts) != 42 {
		t.Errorf("texts has %d languages", len(texts))
	}
}

func TestTheVersionIsTheClients(t *testing.T) {
	raw, err := os.ReadFile("../../../package.json")
	if err != nil {
		t.Fatal(err)
	}
	var pkg struct{ Version string }
	if err := json.Unmarshal(raw, &pkg); err != nil {
		t.Fatal(err)
	}
	if pkg.Version != Version {
		t.Errorf("package.json says %s, the server %s", pkg.Version, Version)
	}
}

func TestTheIconIsCastleParcelsMark(t *testing.T) {
	if err := status.CheckIcon(Icon); err != nil {
		t.Fatal(err)
	}
}
