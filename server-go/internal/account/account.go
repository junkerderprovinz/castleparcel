// Package account is the owner's sign-in: the kit's sign-in on CastleParcel's
// database, and who may upload once the owner has a password. Downloads never
// ask, the key in the link is their proof.
package account

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/junkerderprovinz/parleyport/kit/signin"
	"github.com/junkerderprovinz/parleyport/kit/signin/sqlstore"

	"github.com/junkerderprovinz/castleparcel/server-go/internal/config"
	"github.com/junkerderprovinz/castleparcel/server-go/internal/store"
	"github.com/junkerderprovinz/castleparcel/server-go/internal/upload"
)

// keyFile holds the key the sign-in is sealed with when CASTLEPARCEL_APP_KEY
// is not set.
const keyFile = "app.key"

// Open opens the sign-in on db, sealed with the key from
// CASTLEPARCEL_APP_KEY or from the key file in CONFIG_DIR, which the first
// start writes. onReset, unless nil, hears of a reset through the log.
func Open(cfg config.Config, db *sql.DB, log *slog.Logger, onReset func()) (*signin.Auth, error) {
	key, err := loadKey(cfg)
	if err != nil {
		return nil, err
	}
	seal, err := signin.NewSealer(key)
	if err != nil {
		return nil, err
	}
	docs, err := sqlstore.New(context.Background(), db, store.SignInTable)
	if err != nil {
		return nil, err
	}
	sc := signin.Config{
		Domain:      "cp",
		Name:        cfg.Branding().AppName,
		Store:       docs,
		Seal:        seal,
		Owner:       true,
		TokenPrefix: "cp",
		Public:      func() *url.URL { return public(cfg.BaseURL) },
		Proxies:     cfg.TrustedProxies.Prefixes(),
		Logger:      log,
		Hooks:       signin.Hooks{Ended: ended(db, onReset)},
	}
	if cfg.TrustedProxies.All() {
		sc.TrustedPeer = func(*http.Request) bool { return true }
	}
	return signin.Open(sc)
}

// shutOut are the endings after which an account keeps nothing it opened.
var shutOut = map[signin.Ending]bool{
	signin.EndBlocked: true, signin.EndRemoved: true, signin.EndRole: true, signin.EndRefused: true,
	signin.EndRevoked: true, signin.EndReset: true, signin.EndResetOther: true,
}

// ended closes the file requests of an account that was shut out, since a
// request link lets anybody keep uploading in its name. Shares stay: their
// link belongs to the recipient now, and the management link deletes one
// early.
func ended(db *sql.DB, onReset func()) func(context.Context, string, signin.Ending) error {
	return func(_ context.Context, account string, what signin.Ending) error {
		if what == signin.EndReset && onReset != nil {
			onReset()
		}
		if !shutOut[what] {
			return nil
		}
		_, err := store.CloseRequestsOf(db, account)
		return err
	}
}

// Uploads guards the routes that start an upload. The signed-in owner passes
// and needs no upload password. Once the owner has a password, a stranger
// uploads only with UPLOAD_PASSWORD, and without one not at all. While
// CastleParcel is open, everything stays as it was without a sign-in. An
// upload for a file request carries its link as the proof, which the tus gate
// checks.
func Uploads(auth *signin.Auth, cfg config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if v, ok := signin.From(r.Context()); ok && owner(v) {
				next.ServeHTTP(w, upload.Admit(r))
				return
			}
			if auth.Anyone() || cfg.UploadProtected || r.Method == http.MethodOptions || r.Header.Get(upload.RequestHeader) != "" {
				next.ServeHTTP(w, r)
				return
			}
			refuse(w, "signInFirst")
		})
	}
}

// owner reports whether v is the owner signed in, through a session or a
// token that may write.
func owner(v signin.Visitor) bool {
	switch v.Via {
	case signin.ViaAnyone:
		return false
	case signin.ViaToken:
		return v.AtLeast("admin") && slices.Contains(v.Scopes, "all")
	}
	return v.AtLeast("admin")
}

// Contract guards the sign-in's own routes while CastleParcel is open and an
// upload password is set: only who knows that password may make changes, such
// as setting the first password, so a stranger cannot become the owner of a
// CastleParcel whose uploads were closed to them.
func Contract(auth *signin.Auth, cfg config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfg.UploadProtected && auth.Anyone() && r.Method != http.MethodGet && r.Method != http.MethodHead &&
			!upload.IsUploadAuthorized(r.Header.Get(upload.UploadTokenHeader), true, cfg.UploadPassword) {
			refuse(w, "uploadPassword")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Admins lets only the owner, signed in through the browser, reach the
// settings behind next. While CastleParcel is open everyone may, as with the
// sign-in itself, and an upload password then guards them as it guards the
// sign-in. An API token never may, since the settings hold other services'
// secrets.
func Admins(auth *signin.Auth, cfg config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v, ok := signin.From(r.Context())
		switch {
		case !ok:
			refuse(w, "signInFirst")
		case v.Via == signin.ViaToken:
			fail(w, http.StatusForbidden, "notWithToken")
		case !v.AtLeast("admin"):
			fail(w, http.StatusForbidden, "forbidden")
		case v.Via == signin.ViaAnyone && cfg.UploadProtected && auth.Anyone() &&
			!upload.IsUploadAuthorized(r.Header.Get(upload.UploadTokenHeader), true, cfg.UploadPassword):
			refuse(w, "uploadPassword")
		default:
			next.ServeHTTP(w, r)
		}
	})
}

func refuse(w http.ResponseWriter, code string) { fail(w, http.StatusUnauthorized, code) }

func fail(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, "{\"error\":%q}\n", code)
}

// Key returns the key the sign-in is sealed with, for the other secrets
// CastleParcel keeps.
func Key(cfg config.Config) ([]byte, error) { return loadKey(cfg) }

// loadKey returns the key from CASTLEPARCEL_APP_KEY, or from the key file,
// drawing and writing one on the first start.
func loadKey(cfg config.Config) ([]byte, error) {
	if cfg.AppKey != "" {
		key, err := parseKey(cfg.AppKey)
		if err != nil {
			return nil, fmt.Errorf("CASTLEPARCEL_APP_KEY: %w", err)
		}
		return key, nil
	}
	path := filepath.Join(cfg.ConfigDir, keyFile)
	body, err := os.ReadFile(path)
	if err == nil {
		key, err := parseKey(strings.TrimSpace(string(body)))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		return key, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	// O_EXCL, so two processes starting at once cannot each write their own.
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return loadKey(cfg)
	}
	if err != nil {
		return nil, fmt.Errorf("write %s: %w", path, err)
	}
	_, werr := f.WriteString(hex.EncodeToString(key) + "\n")
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		_ = os.Remove(path)
		return nil, fmt.Errorf("write %s: %w", path, werr)
	}
	return key, nil
}

func parseKey(raw string) ([]byte, error) {
	key, err := hex.DecodeString(raw)
	if err != nil || len(key) != 32 {
		return nil, errors.New("the key has to be 64 hexadecimal characters, as openssl rand -hex 32 prints")
	}
	return key, nil
}

// public is BASE_URL as browsers write it: a lower-case host without the
// default port, and no path. Passkeys and sign-in services need it.
func public(base string) *url.URL {
	u, err := url.Parse(base)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return nil
	}
	host, port := strings.ToLower(u.Hostname()), u.Port()
	if (u.Scheme == "https" && port == "443") || (u.Scheme == "http" && port == "80") {
		port = ""
	}
	if port != "" {
		host = net.JoinHostPort(host, port)
	} else if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	return &url.URL{Scheme: u.Scheme, Host: host}
}
