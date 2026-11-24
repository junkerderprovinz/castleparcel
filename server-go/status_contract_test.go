package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/junkerderprovinz/parleyport/kit/signin"
	"github.com/junkerderprovinz/parleyport/kit/status"
	"github.com/junkerderprovinz/parleyport/kit/statustest"

	"github.com/junkerderprovinz/castleparcel/server-go/internal/config"
	"github.com/junkerderprovinz/castleparcel/server-go/internal/store"
)

// ownerToken sets the owner's password and returns an API token of scope.
func ownerToken(t *testing.T, h http.Handler, scope string) string {
	t.Helper()
	cookie := ownerSession(t, h)
	rec := send(h, http.MethodPost, signin.APIPrefix+"me/tokens", `{"name":"Homarr","scopes":["`+scope+`"]}`, "Cookie", cookie)
	var made struct{ Value string }
	if err := json.Unmarshal(rec.Body.Bytes(), &made); err != nil || made.Value == "" {
		t.Fatalf("token: %d %s", rec.Code, rec.Body)
	}
	return made.Value
}

// The status is checked where a password is set, on two parcels that fill
// the storage quota past nine tenths.
func TestRouter_KeepsTheStatusContract(t *testing.T) {
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

func TestRouter_TheStatusIsClosedToStrangersOnceThereIsAPassword(t *testing.T) {
	h := newFullRouter(t, withBaseURL)
	if rec := send(h, http.MethodGet, status.StatusPath, ""); rec.Code != http.StatusOK {
		t.Errorf("the status of an open CastleParcel: %d %s", rec.Code, rec.Body)
	}
	token := ownerToken(t, h, "all")
	for _, path := range []string{status.StatusPath, status.LogPath} {
		if rec := send(h, http.MethodGet, path, ""); rec.Code != http.StatusUnauthorized {
			t.Errorf("a stranger reads %s: %d %s", path, rec.Code, rec.Body)
		}
		if rec := send(h, http.MethodGet, path, "", "Authorization", "Bearer cp_not-a-token"); rec.Code != http.StatusUnauthorized {
			t.Errorf("a made-up token reads %s: %d %s", path, rec.Code, rec.Body)
		}
		if rec := send(h, http.MethodGet, path, "", "Authorization", "Bearer "+token); rec.Code != http.StatusOK {
			t.Errorf("a token reads %s: %d %s", path, rec.Code, rec.Body)
		}
	}
}

func TestRouter_TheStatusWantsTheUploadPasswordWhileCastleParcelIsOpen(t *testing.T) {
	h := newFullRouter(t, func(c *config.Config) {
		withBaseURL(c)
		c.UploadProtected, c.UploadPassword = true, "an upload password"
	})
	if rec := send(h, http.MethodGet, status.StatusPath, ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("a stranger reads the status: %d %s", rec.Code, rec.Body)
	}
}
