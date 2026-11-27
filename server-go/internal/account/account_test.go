package account

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/junkerderprovinz/parleyport/kit/signin"

	"github.com/junkerderprovinz/castleparcel/server-go/internal/config"
	"github.com/junkerderprovinz/castleparcel/server-go/internal/notices"
	"github.com/junkerderprovinz/castleparcel/server-go/internal/store"
	"github.com/junkerderprovinz/castleparcel/server-go/internal/upload"
)

const ownerPassword = "a password for the owner"

func testConfig(t *testing.T) config.Config {
	t.Helper()
	dir := t.TempDir()
	return config.Config{ConfigDir: dir, DBPath: filepath.Join(dir, "db.sqlite"), BaseURL: "https://parcel.example"}
}

func open(t *testing.T, cfg config.Config) *signin.Auth {
	t.Helper()
	db, err := store.Open(cfg.DBPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	auth, err := Open(cfg, db, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = auth.Close() })
	return auth
}

// app is the sign-in with the upload gate in front of a handler that only
// says whether the request may upload.
func app(auth *signin.Auth, cfg config.Config) http.Handler {
	mux := http.NewServeMux()
	mux.Handle(signin.APIPrefix, Contract(auth, cfg, auth.Handler()))
	mux.Handle("/upload", Uploads(auth, cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if upload.Allowed(r, cfg) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	})))
	return auth.Guard(mux)
}

func call(h http.Handler, method, path string, body any, headers ...string) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	r := httptest.NewRequest(method, "https://parcel.example"+path, &buf)
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	for i := 0; i+1 < len(headers); i += 2 {
		r.Header.Set(headers[i], headers[i+1])
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func errorOf(w *httptest.ResponseRecorder) string {
	var body struct{ Error string }
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return body.Error
}

// signIn sets the owner's first password and returns the session cookie of
// a sign-in with it.
func signIn(t *testing.T, h http.Handler, headers ...string) string {
	t.Helper()
	if w := call(h, "PUT", signin.APIPrefix+"me/password", map[string]string{"next": ownerPassword}, headers...); w.Code != http.StatusOK {
		t.Fatalf("first password: %d %s", w.Code, w.Body)
	}
	w := call(h, "POST", signin.APIPrefix+"login", map[string]string{"password": ownerPassword})
	if w.Code != http.StatusOK {
		t.Fatalf("login: %d %s", w.Code, w.Body)
	}
	for _, c := range w.Result().Cookies() {
		if strings.HasSuffix(c.Name, "session") {
			return c.Name + "=" + c.Value
		}
	}
	t.Fatal("no session cookie")
	return ""
}

func TestTheFirstStartKeepsAKeyInTheConfigFolder(t *testing.T) {
	cfg := testConfig(t)
	open(t, cfg)
	path := filepath.Join(cfg.ConfigDir, "app.key")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read key file: %v", err)
	}
	if key, err := hex.DecodeString(strings.TrimSpace(string(raw))); err != nil || len(key) != 32 {
		t.Errorf("key file holds %q", raw)
	}
	if info, err := os.Stat(path); err == nil && runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Errorf("key file mode %v, want 0600", info.Mode().Perm())
	}
}

func TestTheKeySurvivesARestart(t *testing.T) {
	cfg := testConfig(t)
	signIn(t, app(open(t, cfg), cfg))

	again := app(open(t, cfg), cfg)
	if w := call(again, "POST", signin.APIPrefix+"login", map[string]string{"password": ownerPassword}); w.Code != http.StatusOK {
		t.Errorf("login after a restart: %d %s", w.Code, w.Body)
	}
}

func TestAKeyFromTheEnvironmentWins(t *testing.T) {
	cfg := testConfig(t)
	cfg.AppKey = strings.Repeat("ab", 32)
	open(t, cfg)
	if _, err := os.Stat(filepath.Join(cfg.ConfigDir, "app.key")); !os.IsNotExist(err) {
		t.Errorf("a key file was written beside CASTLEPARCEL_APP_KEY: %v", err)
	}
}

func TestABrokenKeyStopsTheStart(t *testing.T) {
	cfg := testConfig(t)
	cfg.AppKey = "not hex"
	db, err := store.Open(cfg.DBPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if auth, err := Open(cfg, db, nil, nil); err == nil {
		_ = auth.Close()
		t.Fatal("Open took a key that is no key")
	}
}

// database is CastleParcel's database for cfg, as a start opens it.
func database(t *testing.T, cfg config.Config) *sql.DB {
	t.Helper()
	db, err := store.Open(cfg.DBPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestAKeyFileAStartLeftUnfinishedIsCompletedByTheNext(t *testing.T) {
	for name, body := range map[string]string{"empty": "", "cut short": "3f9a0c71"} {
		t.Run(name, func(t *testing.T) {
			cfg := testConfig(t)
			if err := os.WriteFile(filepath.Join(cfg.ConfigDir, keyFile), []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			db := database(t, cfg)
			key, err := Key(cfg, db)
			if err != nil {
				t.Fatal(err)
			}
			open(t, cfg)
			again, err := Key(cfg, db)
			if err != nil {
				t.Fatal(err)
			}
			if len(key) != 32 || !bytes.Equal(key, again) {
				t.Fatalf("the key changed between two starts: %x, %x", key, again)
			}
		})
	}
}

func TestAnUnfinishedKeyFileStaysWhenSomethingIsSealedWithTheKey(t *testing.T) {
	for name, seal := range map[string]func(t *testing.T, cfg config.Config){
		"the sign-in": func(t *testing.T, cfg config.Config) {
			signIn(t, app(open(t, cfg), cfg))
		},
		"the fortress": func(t *testing.T, cfg config.Config) {
			if err := store.SaveFortressDoc(database(t, cfg), "membership", []byte("sealed")); err != nil {
				t.Fatal(err)
			}
		},
		"the notifications": func(t *testing.T, cfg config.Config) {
			if err := os.WriteFile(filepath.Join(cfg.ConfigDir, notices.FileName), []byte("sealed"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := testConfig(t)
			seal(t, cfg)
			path := filepath.Join(cfg.ConfigDir, keyFile)
			if err := os.WriteFile(path, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Key(cfg, database(t, cfg)); err == nil {
				t.Error("a fresh key was drawn although something was sealed with the lost one")
			}
			if body, err := os.ReadFile(path); err != nil || len(body) != 0 {
				t.Errorf("the key file was touched: %q, %v", body, err)
			}
		})
	}
}

func TestAKeyFileWithSomethingElseInItIsAnError(t *testing.T) {
	cfg := testConfig(t)
	path := filepath.Join(cfg.ConfigDir, keyFile)
	if err := os.WriteFile(path, []byte("not a key at all\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Key(cfg, database(t, cfg)); err == nil {
		t.Error("a file that holds no key was taken")
	}
	if body, _ := os.ReadFile(path); string(body) != "not a key at all\n" {
		t.Errorf("the file was replaced: %q", body)
	}
}

func TestStartsAtOnceAllGetTheKeyThatIsKept(t *testing.T) {
	for range 50 {
		cfg := testConfig(t)
		db := database(t, cfg)
		const starts = 8
		keys := make([][]byte, starts)
		errs := make([]error, starts)
		gate := make(chan struct{})
		var wg sync.WaitGroup
		for i := range starts {
			wg.Go(func() {
				<-gate
				keys[i], errs[i] = Key(cfg, db)
			})
		}
		close(gate)
		wg.Wait()
		kept, err := os.ReadFile(filepath.Join(cfg.ConfigDir, keyFile))
		if err != nil {
			t.Fatal(err)
		}
		for i := range starts {
			if errs[i] != nil {
				t.Fatalf("start %d: %v", i, errs[i])
			}
			if hex.EncodeToString(keys[i]) != strings.TrimSpace(string(kept)) {
				t.Fatalf("start %d got %x, the file keeps %s", i, keys[i], kept)
			}
		}
	}
}

func TestTheKeyFileNeverShowsPartOfAKey(t *testing.T) {
	for range 200 {
		cfg := testConfig(t)
		db := database(t, cfg)
		path := filepath.Join(cfg.ConfigDir, keyFile)
		seen := make(chan string, 1)
		go func() {
			for {
				if body, err := os.ReadFile(path); err == nil {
					seen <- string(body)
					return
				}
			}
		}()
		if _, err := Key(cfg, db); err != nil {
			t.Fatal(err)
		}
		if body := <-seen; len(strings.TrimSpace(body)) != 64 {
			t.Fatalf("a start reading at the wrong moment found %q", body)
		}
	}
}

func TestAnOpenCastleParcelUploadsAsBefore(t *testing.T) {
	cfg := testConfig(t)
	h := app(open(t, cfg), cfg)
	if w := call(h, "POST", "/upload", nil); w.Code != http.StatusNoContent {
		t.Errorf("upload while open: %d", w.Code)
	}
}

func TestAnOpenCastleParcelStillWantsTheUploadPassword(t *testing.T) {
	cfg := testConfig(t)
	cfg.UploadPassword, cfg.UploadProtected = "guest secret", true
	h := app(open(t, cfg), cfg)
	if w := call(h, "POST", "/upload", nil); w.Code != http.StatusUnauthorized {
		t.Errorf("upload without the password: %d", w.Code)
	}
	if w := call(h, "POST", "/upload", nil, upload.UploadTokenHeader, "guest secret"); w.Code != http.StatusNoContent {
		t.Errorf("upload with the password: %d", w.Code)
	}
}

func TestOnceTheOwnerHasAPasswordOnlyTheOwnerUploads(t *testing.T) {
	cfg := testConfig(t)
	h := app(open(t, cfg), cfg)
	cookie := signIn(t, h)

	w := call(h, "POST", "/upload", nil)
	if w.Code != http.StatusUnauthorized || errorOf(w) != "signInFirst" {
		t.Errorf("stranger uploads: %d %s", w.Code, w.Body)
	}
	if w := call(h, "POST", "/upload", nil, "Cookie", cookie); w.Code != http.StatusNoContent {
		t.Errorf("owner uploads: %d", w.Code)
	}
}

func TestGuestsUploadWithTheUploadPasswordAndTheOwnerWithout(t *testing.T) {
	cfg := testConfig(t)
	cfg.UploadPassword, cfg.UploadProtected = "guest secret", true
	h := app(open(t, cfg), cfg)
	cookie := signIn(t, h, upload.UploadTokenHeader, "guest secret")

	if w := call(h, "POST", "/upload", nil, upload.UploadTokenHeader, "guest secret"); w.Code != http.StatusNoContent {
		t.Errorf("guest with the password: %d", w.Code)
	}
	if w := call(h, "POST", "/upload", nil, upload.UploadTokenHeader, "wrong"); w.Code != http.StatusUnauthorized {
		t.Errorf("guest with a wrong password: %d", w.Code)
	}
	if w := call(h, "POST", "/upload", nil, "Cookie", cookie); w.Code != http.StatusNoContent {
		t.Errorf("owner without the password: %d", w.Code)
	}
}

func TestAFileRequestUploadIsLeftToItsLink(t *testing.T) {
	cfg := testConfig(t)
	h := app(open(t, cfg), cfg)
	signIn(t, h)
	// The tus gate checks the request link itself, so the sign-in lets it by.
	if w := call(h, "POST", "/upload", nil, upload.RequestHeader, "some-request"); errorOf(w) == "signInFirst" {
		t.Errorf("a request upload was sent to sign in")
	}
}

func TestAReadTokenDoesNotUpload(t *testing.T) {
	cfg := testConfig(t)
	h := app(open(t, cfg), cfg)
	cookie := signIn(t, h)
	tokens := map[string]string{}
	for _, scope := range []string{"read", "all"} {
		w := call(h, "POST", signin.APIPrefix+"me/tokens", map[string]any{"name": scope, "scopes": []string{scope}}, "Cookie", cookie)
		var made struct{ Value string }
		if _ = json.Unmarshal(w.Body.Bytes(), &made); made.Value == "" {
			t.Fatalf("token %s: %d %s", scope, w.Code, w.Body)
		}
		tokens[scope] = made.Value
	}
	if w := call(h, "POST", "/upload", nil, "Authorization", "Bearer "+tokens["read"]); w.Code != http.StatusUnauthorized {
		t.Errorf("read token uploads: %d", w.Code)
	}
	if w := call(h, "POST", "/upload", nil, "Authorization", "Bearer "+tokens["all"]); w.Code != http.StatusNoContent {
		t.Errorf("read-and-write token: %d", w.Code)
	}
}

func TestWhileOpenTheUploadPasswordGuardsTheSignIn(t *testing.T) {
	cfg := testConfig(t)
	cfg.UploadPassword, cfg.UploadProtected = "guest secret", true
	h := app(open(t, cfg), cfg)

	if w := call(h, "PUT", signin.APIPrefix+"me/password", map[string]string{"next": ownerPassword}); w.Code != http.StatusUnauthorized || errorOf(w) != "uploadPassword" {
		t.Errorf("first password without the upload password: %d %s", w.Code, w.Body)
	}
	if w := call(h, "GET", signin.APIPrefix+"options", nil); w.Code != http.StatusOK {
		t.Errorf("options: %d", w.Code)
	}
	if w := call(h, "PUT", signin.APIPrefix+"me/password", map[string]string{"next": ownerPassword}, upload.UploadTokenHeader, "guest secret"); w.Code != http.StatusOK {
		t.Errorf("first password with the upload password: %d %s", w.Code, w.Body)
	}
	if w := call(h, "POST", signin.APIPrefix+"login", map[string]string{"password": ownerPassword}); w.Code != http.StatusOK {
		t.Errorf("login once closed needs no upload password: %d %s", w.Code, w.Body)
	}
}

func adminApp(auth *signin.Auth, cfg config.Config) http.Handler {
	mux := http.NewServeMux()
	mux.Handle(signin.APIPrefix, Contract(auth, cfg, auth.Handler()))
	mux.Handle("/settings", Admins(auth, cfg, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })))
	return auth.Guard(mux)
}

func TestOnlyTheOwnerChangesTheSettingsOnceThereIsAPassword(t *testing.T) {
	cfg := testConfig(t)
	h := adminApp(open(t, cfg), cfg)
	if w := call(h, "PUT", "/settings", nil); w.Code != http.StatusNoContent {
		t.Errorf("open castle: %d", w.Code)
	}
	cookie := signIn(t, h)
	if w := call(h, "PUT", "/settings", nil); w.Code != http.StatusUnauthorized {
		t.Errorf("stranger: %d", w.Code)
	}
	if w := call(h, "PUT", "/settings", nil, "Cookie", cookie); w.Code != http.StatusNoContent {
		t.Errorf("owner: %d", w.Code)
	}
	w := call(h, "POST", signin.APIPrefix+"me/tokens", map[string]any{"name": "all", "scopes": []string{"all"}}, "Cookie", cookie)
	var made struct{ Value string }
	_ = json.Unmarshal(w.Body.Bytes(), &made)
	if w := call(h, "PUT", "/settings", nil, "Authorization", "Bearer "+made.Value); w.Code != http.StatusForbidden {
		t.Errorf("API token: %d", w.Code)
	}
}

func TestWhileOpenTheUploadPasswordGuardsTheSettings(t *testing.T) {
	cfg := testConfig(t)
	cfg.UploadPassword, cfg.UploadProtected = "guest secret", true
	h := adminApp(open(t, cfg), cfg)
	if w := call(h, "GET", "/settings", nil); w.Code != http.StatusUnauthorized || errorOf(w) != "uploadPassword" {
		t.Errorf("without the upload password: %d %s", w.Code, w.Body)
	}
	if w := call(h, "GET", "/settings", nil, upload.UploadTokenHeader, "guest secret"); w.Code != http.StatusNoContent {
		t.Errorf("with the upload password: %d", w.Code)
	}
}

// CastleParcel serves its pages, its API and the sign-in from the root of its
// host, so a path in BASE_URL never moves the sign-in's addresses.
func TestTheSignInStaysAtTheRootWhateverPathBaseURLCarries(t *testing.T) {
	for base, want := range map[string]string{
		"https://Parcel.Example:443/sub/": "https://parcel.example",
		"http://parcel.example:8080/a/b":  "http://parcel.example:8080",
		"https://[::1]/x":                 "https://[::1]",
	} {
		if got := public(base); got == nil || got.String() != want {
			t.Errorf("public(%q) = %v, want %s", base, got, want)
		}
	}
	if public("not a url") != nil {
		t.Error("an address without a host counts")
	}
}
