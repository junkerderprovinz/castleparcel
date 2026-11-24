package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	kitfortress "github.com/junkerderprovinz/parleyport/kit/fortress"
	"github.com/junkerderprovinz/parleyport/kit/group"
	"github.com/junkerderprovinz/parleyport/kit/member"
	"github.com/junkerderprovinz/parleyport/kit/phrase"
	"github.com/junkerderprovinz/parleyport/kit/relay"
	"github.com/junkerderprovinz/parleyport/kit/signin"
	"github.com/junkerderprovinz/parleyport/kit/status"
	"github.com/junkerderprovinz/parleyport/kit/statustest"

	"github.com/junkerderprovinz/castleparcel/server-go/internal/config"
	"github.com/junkerderprovinz/castleparcel/server-go/internal/store"
)

// send makes a request of CastleParcel's own page.
func send(h http.Handler, method, path string, body any, headers ...string) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, "https://parcel.example"+path, &buf)
	req.RemoteAddr = "192.0.2.1:1234"
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// ownerToken sets the owner's password and returns an API token of scope.
func ownerToken(t *testing.T, h http.Handler, scope string) string {
	t.Helper()
	if w := send(h, "PUT", signin.APIPrefix+"me/password", map[string]string{"next": "a password for the owner"}); w.Code != http.StatusOK {
		t.Fatalf("first password: %d %s", w.Code, w.Body)
	}
	w := send(h, "POST", signin.APIPrefix+"login", map[string]string{"password": "a password for the owner"})
	var cookie string
	for _, c := range w.Result().Cookies() {
		if strings.HasSuffix(c.Name, "session") {
			cookie = c.Name + "=" + c.Value
		}
	}
	w = send(h, "POST", signin.APIPrefix+"me/tokens", map[string]any{"name": "Homarr", "scopes": []string{scope}}, "Cookie", cookie)
	var made struct{ Value string }
	if err := json.Unmarshal(w.Body.Bytes(), &made); err != nil || made.Value == "" {
		t.Fatalf("token: %d %s", w.Code, w.Body)
	}
	return made.Value
}

// The status is checked where a password is set, on two parcels that fill
// the storage quota past nine tenths.
func TestTheStatusAnswersTheFortressContract(t *testing.T) {
	app := newFullApp(t, func(c *config.Config) { withBaseURL(c); c.StorageQuota = 100 })
	for i, slug := range []string{"one", "two"} {
		rec := store.FileRecord{ID: slug, Slug: slug, Size: 48, CreatedAt: int64(1000 + i), Format: 4}
		if err := store.CreateFileRecord(app.db, rec); err != nil {
			t.Fatal(err)
		}
	}
	token := ownerToken(t, app.h, "read")
	statustest.Run(t, app.h, statustest.Options{Authenticate: func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+token)
	}})
}

func TestTheFortressIsTheOwnersAlone(t *testing.T) {
	h := newFullRouter(t, withBaseURL)
	token := ownerToken(t, h, "all")
	if w := send(h, "GET", "/api/fortress", nil); w.Code != http.StatusUnauthorized {
		t.Errorf("a stranger reads the fortress: %d %s", w.Code, w.Body)
	}
	if w := send(h, "GET", "/api/fortress", nil, "Authorization", "Bearer "+token); w.Code != http.StatusForbidden {
		t.Errorf("a token reads the fortress: %d %s", w.Code, w.Body)
	}
	if w := send(h, "GET", status.StatusPath, nil, "Authorization", "Bearer "+token); w.Code != http.StatusOK {
		t.Errorf("a token reads the status: %d %s", w.Code, w.Body)
	}
}

func TestAnUpgradedInstallIsInNoFortress(t *testing.T) {
	h := newFullRouter(t, nil)
	w := send(h, "GET", "/api/fortress", nil)
	var v struct {
		Joined  bool
		Members []any
	}
	if err := json.Unmarshal(w.Body.Bytes(), &v); w.Code != http.StatusOK || err != nil || v.Joined || len(v.Members) != 0 {
		t.Errorf("GET /api/fortress = %d %s", w.Code, w.Body)
	}
	if w := send(h, "POST", group.DirectPath, nil); w.Code != http.StatusNotFound {
		t.Errorf("the direct calls answer outside a fortress: %d", w.Code)
	}
}

// bravoPages is another app of the fortress whose pages open from
// CastleParcel, with a policy that forbids every frame but its own bar's.
func bravoPages(t *testing.T, secret []byte, relayURL string) string {
	t.Helper()
	id, err := member.NewIdentity(member.KindInstance)
	if err != nil {
		t.Fatal(err)
	}
	pages := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		_, _ = w.Write([]byte("<!doctype html><p>hello from bravo</p>"))
	})
	f, err := kitfortress.Open(kitfortress.Config{
		App: kitfortress.App{ID: "bravo", Name: "Bravo", Version: "1.0.0"}, Secret: secret, Identity: id,
		Self: group.Self{Name: "Bravo box", Deployment: "container"}, Relay: group.RelayOwn, RelayURL: relayURL,
		Store: &memStore{}, UI: &kitfortress.UI{Handler: pages, Opened: func(kitfortress.Caller) {}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return id.ID()
}

type memStore struct{ raw []byte }

func (m *memStore) Load() ([]byte, error) { return m.raw, nil }
func (m *memStore) Save(raw []byte) error { m.raw = slices.Clone(raw); return nil }

// The tab carries the kit's policies, not CastleParcel's: the bar frames the
// member's page at the same address, which frame-ancestors 'none' and DENY
// would forbid, and the page runs in a sandbox of its own.
func TestAnotherAppsPagesOpenInATabWithTheKitsPolicies(t *testing.T) {
	srv := httptest.NewServer(relay.New())
	t.Cleanup(srv.CloseClientConnections)
	t.Cleanup(srv.Close)
	secret, words, err := phrase.New()
	if err != nil {
		t.Fatal(err)
	}
	bravo := bravoPages(t, secret, srv.URL)
	h := newFullRouter(t, withBaseURL)

	if w := send(h, "PUT", "/api/fortress/network", map[string]any{"relay": "own", "url": srv.URL, "lan": false}); w.Code != http.StatusOK {
		t.Fatalf("network: %d %s", w.Code, w.Body)
	}
	if w := send(h, "POST", "/api/fortress/join", map[string]string{"phrase": words}); w.Code != http.StatusOK {
		t.Fatalf("join: %d %s", w.Code, w.Body)
	}
	deadline := time.Now().Add(15 * time.Second)
	for !strings.Contains(send(h, "GET", "/api/fortress", nil).Body.String(), `"pages":true`) {
		if time.Now().After(deadline) {
			t.Fatal("Bravo's pages never showed")
		}
		time.Sleep(50 * time.Millisecond)
	}

	w := send(h, "POST", "/api/fortress/open/"+bravo, nil)
	var opened struct{ Path string }
	if err := json.Unmarshal(w.Body.Bytes(), &opened); err != nil || w.Code != http.StatusOK || !strings.HasPrefix(opened.Path, kitfortress.OpenPath) {
		t.Fatalf("open: %d %s", w.Code, w.Body)
	}
	var visitor string
	for _, c := range w.Result().Cookies() {
		visitor = c.Name + "=" + c.Value
	}

	bar := send(h, "GET", opened.Path, nil, "Cookie", visitor)
	if bar.Code != http.StatusOK || !strings.Contains(bar.Body.String(), "<iframe") {
		t.Fatalf("the tab: %d %s", bar.Code, bar.Body)
	}
	if csp := bar.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "frame-src 'self'") || bar.Header().Get("X-Frame-Options") != "" {
		t.Errorf("the tab's headers: %v", bar.Header())
	}
	page := send(h, "GET", opened.Path+"/", nil, "Cookie", visitor, "Sec-Fetch-Dest", "iframe")
	policies := strings.Join(page.Header().Values("Content-Security-Policy"), " | ")
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "hello from bravo") {
		t.Fatalf("the page: %d %s", page.Code, page.Body)
	}
	if !strings.Contains(policies, "sandbox") || strings.Contains(policies, "frame-ancestors 'none'") || page.Header().Get("X-Frame-Options") != "" {
		t.Errorf("the page's policies: %s, X-Frame-Options %q", policies, page.Header().Get("X-Frame-Options"))
	}
	if w := send(h, "GET", opened.Path, nil); w.Code != http.StatusSeeOther {
		t.Errorf("another browser opens the tab: %d", w.Code)
	}
}
