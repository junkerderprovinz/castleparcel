package fortress

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	kitfortress "github.com/junkerderprovinz/parleyport/kit/fortress"
	"github.com/junkerderprovinz/parleyport/kit/group"
	"github.com/junkerderprovinz/parleyport/kit/member"
	"github.com/junkerderprovinz/parleyport/kit/phrase"
	"github.com/junkerderprovinz/parleyport/kit/relay"
	"github.com/junkerderprovinz/parleyport/kit/status"

	"github.com/junkerderprovinz/castleparcel/server-go/internal/account"
	"github.com/junkerderprovinz/castleparcel/server-go/internal/config"
	"github.com/junkerderprovinz/castleparcel/server-go/internal/report"
	"github.com/junkerderprovinz/castleparcel/server-go/internal/store"
)

func relayServer(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(relay.New())
	t.Cleanup(srv.CloseClientConnections)
	t.Cleanup(srv.Close)
	return srv.URL
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// castle is a CastleParcel with its fortress on the relay at relayURL and
// its database in dir.
type castle struct {
	*Service
	dir string
	h   http.Handler
}

func openCastle(t *testing.T, dir, relayURL string) *castle {
	t.Helper()
	cfg := config.Config{ConfigDir: dir, DBPath: filepath.Join(dir, "db.sqlite"), TmpDir: dir}
	db, err := store.Open(cfg.DBPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	auth, err := account.Open(cfg, db, log, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = auth.Close() })
	seal, err := account.Seal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	app := kitfortress.App{ID: "castleparcel", Name: "CastleParcel", Version: report.Version, Icon: report.Icon}
	s := New(Config{
		DB: db, Seal: seal, Auth: auth, App: app,
		Self:   group.Self{Name: "Atlas", Deployment: "container"},
		Status: report.Source{DB: db, TmpDir: dir, App: status.App{ID: app.ID, Name: app.Name}, Instance: func() status.Instance { return status.Instance{ID: "i", Name: "Atlas"} }, Now: time.Now},
		Log:    log,
	})
	s.Start()
	t.Cleanup(s.Close)
	if relayURL != "" {
		if err := s.SetNetwork(relayURL, false); err != nil {
			t.Fatal(err)
		}
	}
	return &castle{Service: s, dir: dir, h: auth.Guard(account.OwnPages(cfg, account.Admins(auth, cfg, s.Routes())))}
}

// bravo is another app of the fortress, with a status of its own.
func bravo(t *testing.T, secret []byte, relayURL string) (*kitfortress.Fortress, string) {
	t.Helper()
	id, err := member.NewIdentity(member.KindInstance)
	if err != nil {
		t.Fatal(err)
	}
	f, err := kitfortress.Open(kitfortress.Config{
		App: kitfortress.App{ID: "bravo", Name: "Bravo", Version: "1.0.0"}, Secret: secret, Identity: id,
		Self: group.Self{Name: "Bravo box", Deployment: "container"}, Relay: group.RelayOwn, RelayURL: relayURL,
		Store: &memoryStore{}, Status: bravoStatus{},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f, id.ID()
}

type bravoStatus struct{}

func (bravoStatus) Report(_ context.Context, lang string) (status.Report, error) {
	return status.Report{App: status.App{ID: "bravo", Name: "Bravo"}, State: status.StateWarning,
		NeedsYou: []status.Need{{ID: "n", Severity: status.SeverityWarn, Code: "x", Text: "Look at me " + lang}}}, nil
}

func (bravoStatus) Log(context.Context, status.LogQuery) (status.LogPage, error) {
	return status.LogPage{Entries: []status.LogEntry{{ID: "1", At: time.Unix(1, 0), Level: status.SeverityInfo, Code: "x", Text: "Something"}}, Next: "1"}, nil
}

func newSecret(t *testing.T) ([]byte, string) {
	t.Helper()
	secret, words, err := phrase.New()
	if err != nil {
		t.Fatal(err)
	}
	return secret, words
}

func (c *castle) call(t *testing.T, method, path string, body any) (int, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	r := httptest.NewRequest(method, "http://parcel.example"+path, &buf)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	w := httptest.NewRecorder()
	c.h.ServeHTTP(w, r)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

// sees reports whether c lists member id and knows its app.
func (c *castle) sees(id string) bool {
	return slices.ContainsFunc(c.view().Members, func(m memberView) bool { return m.ID == id && m.App != "" })
}

func TestJoiningWithThePhraseShowsTheOtherApps(t *testing.T) {
	url := relayServer(t)
	secret, words := newSecret(t)
	_, bravoID := bravo(t, secret, url)
	c := openCastle(t, t.TempDir(), url)

	if code, out := c.call(t, "POST", "/api/fortress/join", map[string]string{"phrase": words}); code != http.StatusOK || out["joined"] != true {
		t.Fatalf("join = %d %v", code, out)
	}
	waitFor(t, "Bravo in the list", func() bool { return c.sees(bravoID) })
	m := c.view().Members[0]
	if m.App != "bravo" || m.Name != "Bravo" || m.Device != "Bravo box" || !m.Relay || m.Pages {
		t.Errorf("member = %+v", m)
	}
	rows, _ := store.LogAfter(c.cfg.DB, 0, "", 10)
	if len(rows) != 1 || rows[0].Code != report.FortressJoined {
		t.Errorf("log = %+v", rows)
	}
}

func TestAPhraseThatDoesNotDecodeIsRefusedWithItsCode(t *testing.T) {
	c := openCastle(t, t.TempDir(), "")
	code, out := c.call(t, "POST", "/api/fortress/join", map[string]string{"phrase": "not a phrase"})
	if code != http.StatusBadRequest || out["error"] != "phrase" {
		t.Errorf("join = %d %v", code, out)
	}
	if c.view().Joined {
		t.Error("joined anyway")
	}
}

func TestTheMembershipSurvivesARestartAndIsSealed(t *testing.T) {
	url := relayServer(t)
	secret, words := newSecret(t)
	_, bravoID := bravo(t, secret, url)
	dir := t.TempDir()
	c := openCastle(t, dir, url)
	if err := c.Join(secret); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "Bravo", func() bool { return c.sees(bravoID) })
	raw, err := store.FortressDoc(c.cfg.DB, docMembership)
	if err != nil || raw == nil || bytes.Contains(raw, secret) || bytes.Contains(raw, []byte("secret")) {
		t.Errorf("the membership is kept as %q, %v", raw, err)
	}
	c.Close()

	again := openCastle(t, dir, "")
	if !again.view().Joined {
		t.Fatal("not in the fortress after a restart")
	}
	if got, _, err := again.Phrase(); err != nil || got != words {
		t.Errorf("phrase after a restart = %q, %v", got, err)
	}
	waitFor(t, "Bravo after the restart", func() bool { return again.sees(bravoID) })
}

func TestAnotherAppsStatusAndLogAreReadForItsWindow(t *testing.T) {
	url := relayServer(t)
	secret, _ := newSecret(t)
	_, bravoID := bravo(t, secret, url)
	c := openCastle(t, t.TempDir(), url)
	if err := c.Join(secret); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "Bravo", func() bool { return c.sees(bravoID) })
	code, out := c.call(t, "GET", "/api/fortress/members/"+bravoID+"/status?lang=de", nil)
	if code != http.StatusOK {
		t.Fatalf("status = %d %v", code, out)
	}
	rep, _ := out["report"].(map[string]any)
	needs, _ := rep["needsYou"].([]any)
	log, _ := out["log"].([]any)
	if rep["state"] != "warning" || len(needs) != 1 || !strings.Contains(needs[0].(map[string]any)["text"].(string), "de") || len(log) != 1 {
		t.Errorf("status = %v", out)
	}
	if code, out := c.call(t, "POST", "/api/fortress/open/"+bravoID, nil); code != http.StatusNotFound || out["error"] != "no_ui" {
		t.Errorf("opening a member without pages = %d %v", code, out)
	}
}

func TestAfterARemovalTheFortressTakesNoNewAppUntilThePhraseChanges(t *testing.T) {
	url := relayServer(t)
	secret, words := newSecret(t)
	_, bravoID := bravo(t, secret, url)
	c := openCastle(t, t.TempDir(), url)
	if err := c.Join(secret); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "Bravo", func() bool { return c.sees(bravoID) })

	if code, out := c.call(t, "DELETE", "/api/fortress/members/"+bravoID, nil); code != http.StatusOK {
		t.Fatalf("remove = %d %v", code, out)
	}
	if got := c.view().Removed; len(got) != 1 || got[0] != (Removed{App: "Bravo", Device: "Bravo box"}) {
		t.Errorf("removed = %+v", got)
	}

	other := openCastle(t, t.TempDir(), url)
	if code, out := other.call(t, "POST", "/api/fortress/join", map[string]string{"phrase": words}); code != http.StatusConflict || out["error"] != group.ClosedUntilNewPhrase {
		t.Errorf("a newcomer's join = %d %v", code, out)
	}
	if other.view().Joined {
		t.Error("the newcomer joined")
	}

	newWords, _, err := c.ChangePhrase(t.Context())
	if err != nil || len(newWords) != phrase.WordCount || strings.Join(newWords, " ") == words {
		t.Fatalf("change = %v, %v", newWords, err)
	}
	if len(c.view().Removed) != 0 {
		t.Error("the removal is still named after the phrase changed")
	}
}

func TestLeavingForgetsTheMembershipButNotTheNetwork(t *testing.T) {
	url := relayServer(t)
	secret, _ := newSecret(t)
	dir := t.TempDir()
	c := openCastle(t, dir, url)
	if err := c.Join(secret); err != nil {
		t.Fatal(err)
	}
	if code, out := c.call(t, "DELETE", "/api/fortress", nil); code != http.StatusOK || out["joined"] != false {
		t.Fatalf("leave = %d %v", code, out)
	}
	for _, doc := range []string{docMembership, docBook} {
		if raw, _ := store.FortressDoc(c.cfg.DB, doc); raw != nil {
			t.Errorf("%s is still kept", doc)
		}
	}
	if v := c.view(); v.Relay.Mode != group.RelayOwn || v.Relay.URL != url || v.LAN {
		t.Errorf("network after leaving = %+v", v.Relay)
	}
	if code, out := c.call(t, "DELETE", "/api/fortress", nil); code != http.StatusConflict || out["error"] != "noFortress" {
		t.Errorf("leaving twice = %d %v", code, out)
	}
}

func TestAnOwnRelayNeedsAnAddress(t *testing.T) {
	c := openCastle(t, t.TempDir(), "")
	for _, bad := range []string{"", "relay.example", "ftp://relay.example", "https://user@relay.example"} {
		if code, out := c.call(t, "PUT", "/api/fortress/network", map[string]any{"relay": "own", "url": bad, "lan": true}); code != http.StatusBadRequest || out["error"] != "relayAddress" {
			t.Errorf("%q = %d %v", bad, code, out)
		}
	}
	if code, out := c.call(t, "PUT", "/api/fortress/network", map[string]any{"relay": "off", "lan": true}); code != http.StatusOK || out["lan"] != true {
		t.Errorf("relay off = %d %v", code, out)
	}
	if v := c.view(); v.Relay.Mode != group.RelayOff {
		t.Errorf("relay = %+v", v.Relay)
	}
}

func TestAnInstallThatNeverJoinedIsInNoFortress(t *testing.T) {
	c := openCastle(t, t.TempDir(), "")
	v := c.view()
	if v.Joined || v.RemovedHere || v.Relay.Mode != group.RelayProject || !v.LAN || len(v.Members) != 0 {
		t.Errorf("view = %+v", v)
	}
	if c.current() != nil {
		t.Error("a fortress is open")
	}
}

func TestTakenOutByAnotherAppItLeavesAndSaysSo(t *testing.T) {
	url := relayServer(t)
	secret, _ := newSecret(t)
	other, bravoID := bravo(t, secret, url)
	c := openCastle(t, t.TempDir(), url)
	if err := c.Join(secret); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "Bravo", func() bool { return c.sees(bravoID) })
	m, _, _ := c.membership()
	id, err := member.FromSeed(member.KindInstance, m.Seed)
	if err != nil {
		t.Fatal(err)
	}
	if err := other.Remove(id.ID()); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "CastleParcel to leave", func() bool { v := c.view(); return !v.Joined && v.RemovedHere })
	rows, _ := store.LogAfter(c.cfg.DB, 0, "", 10)
	if len(rows) != 2 || rows[1].Code != report.FortressRemovedHere {
		t.Errorf("log = %+v", rows)
	}
}
