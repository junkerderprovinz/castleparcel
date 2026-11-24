package account

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/junkerderprovinz/parleyport/kit/signin"
	"github.com/junkerderprovinz/parleyport/kit/status"

	"github.com/junkerderprovinz/castleparcel/server-go/internal/config"
	"github.com/junkerderprovinz/castleparcel/server-go/internal/store"
	"github.com/junkerderprovinz/castleparcel/server-go/internal/upload"
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

// owned is app with a route behind OwnPages and Admins, and the status
// behind StatusReader.
func owned(auth *signin.Auth, cfg config.Config) http.Handler {
	mux := http.NewServeMux()
	mux.Handle(signin.APIPrefix, Contract(auth, cfg, auth.Handler()))
	mux.Handle("/owner", OwnPages(cfg, Admins(auth, cfg, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))))
	mux.Handle(status.StatusPath, status.Handler(quiet{}, StatusReader(cfg)))
	return auth.Guard(mux)
}

type quiet struct{}

func (quiet) Report(context.Context, string) (status.Report, error) { return status.Report{}, nil }
func (quiet) Log(context.Context, status.LogQuery) (status.LogPage, error) {
	return status.LogPage{}, nil
}

func TestOnlyTheOwnersSessionReachesTheOwnersRoutes(t *testing.T) {
	cfg := testConfig(t)
	h := owned(open(t, cfg), cfg)
	if w := call(h, "POST", "/owner", nil); w.Code != http.StatusNoContent {
		t.Errorf("while open: %d %s", w.Code, w.Body)
	}
	cookie := signIn(t, h)
	w := call(h, "POST", signin.APIPrefix+"me/tokens", map[string]any{"name": "all", "scopes": []string{"all"}}, "Cookie", cookie)
	var made struct{ Value string }
	_ = json.Unmarshal(w.Body.Bytes(), &made)

	for _, c := range []struct {
		name    string
		headers []string
		want    int
		code    string
	}{
		{"a stranger", nil, http.StatusUnauthorized, "signInFirst"},
		{"a token that may write", []string{"Authorization", "Bearer " + made.Value}, http.StatusForbidden, "notWithToken"},
		{"another site's page", []string{"Cookie", cookie, "Sec-Fetch-Site", "cross-site"}, http.StatusForbidden, "crossSite"},
		{"the owner", []string{"Cookie", cookie}, http.StatusNoContent, ""},
	} {
		if w := call(h, "POST", "/owner", nil, c.headers...); w.Code != c.want || errorOf(w) != c.code {
			t.Errorf("%s: %d %s, want %d %s", c.name, w.Code, w.Body, c.want, c.code)
		}
	}
	if w := call(h, "POST", "/owner", "plain", "Cookie", cookie, "Content-Type", "text/plain"); w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("a body that is no JSON: %d", w.Code)
	}
}

func TestWhileOpenTheUploadPasswordGuardsTheOwnersRoutesAndTheStatus(t *testing.T) {
	cfg := testConfig(t)
	cfg.UploadPassword, cfg.UploadProtected = "guest secret", true
	h := owned(open(t, cfg), cfg)
	for _, path := range []string{"/owner", status.StatusPath} {
		if w := call(h, "GET", path, nil); w.Code != http.StatusUnauthorized {
			t.Errorf("GET %s without the upload password: %d %s", path, w.Code, w.Body)
		}
	}
	if w := call(h, "GET", "/owner", nil, upload.UploadTokenHeader, "guest secret"); w.Code != http.StatusNoContent {
		t.Errorf("with the upload password: %d %s", w.Code, w.Body)
	}
}

func TestATokenOfEitherScopeReadsTheStatus(t *testing.T) {
	cfg := testConfig(t)
	h := owned(open(t, cfg), cfg)
	cookie := signIn(t, h)
	for _, scope := range []string{"read", "all"} {
		w := call(h, "POST", signin.APIPrefix+"me/tokens", map[string]any{"name": scope, "scopes": []string{scope}}, "Cookie", cookie)
		var made struct{ Value string }
		_ = json.Unmarshal(w.Body.Bytes(), &made)
		if w := call(h, "GET", status.StatusPath, nil, "Authorization", "Bearer "+made.Value); w.Code != http.StatusOK {
			t.Errorf("a %s token: %d %s", scope, w.Code, w.Body)
		}
	}
	if w := call(h, "GET", status.StatusPath, nil); w.Code != http.StatusUnauthorized {
		t.Errorf("a stranger: %d", w.Code)
	}
}
