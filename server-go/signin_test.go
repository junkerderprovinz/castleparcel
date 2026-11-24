package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/junkerderprovinz/parleyport/kit/signin"
	"github.com/junkerderprovinz/parleyport/kit/signin/signintest"

	"github.com/junkerderprovinz/castleparcel/server-go/internal/config"
	"github.com/junkerderprovinz/castleparcel/server-go/internal/share"
	"github.com/junkerderprovinz/castleparcel/server-go/internal/store"
)

func withBaseURL(c *config.Config) { c.BaseURL = "https://parcel.example" }

func TestRouter_KeepsTheSignInContract(t *testing.T) {
	signintest.Run(t, newFullRouter(t, withBaseURL), signintest.Options{Password: "a password for the owner"})
}

func TestRouter_ServesTheSignInServiceRoutes(t *testing.T) {
	h := newFullRouter(t, withBaseURL)
	rec := get(h, signin.BrowserPrefix+"oidc/nobody")
	if rec.Code == http.StatusOK && strings.Contains(rec.Body.String(), `id="root"`) {
		t.Errorf("%soidc/nobody fell through to the page", signin.BrowserPrefix)
	}
}

func TestRouter_OnlyTheOwnerUploadsOnceThereIsAPassword(t *testing.T) {
	h := newFullRouter(t, withBaseURL)
	set := httptest.NewRequest(http.MethodPut, "https://parcel.example"+signin.APIPrefix+"me/password", strings.NewReader(`{"next":"a password for the owner"}`))
	set.Header.Set("Content-Type", "application/json")
	set.Header.Set("Sec-Fetch-Site", "same-origin")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, set)
	if rec.Code != http.StatusOK {
		t.Fatalf("first password: %d %s", rec.Code, rec.Body)
	}

	for _, path := range []string{"/files/", "/api/finalize", "/api/requests"} {
		req := httptest.NewRequest(http.MethodPost, "https://parcel.example"+path, bytes.NewReader([]byte(`{}`)))
		req.RemoteAddr = "192.0.2.1:1234"
		req.Header.Set("Tus-Resumable", "1.0.0")
		req.Header.Set("Upload-Length", "10")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "signInFirst") {
			t.Errorf("POST %s by a stranger: %d %s", path, rec.Code, rec.Body)
		}
	}
}

func send(h http.Handler, method, path, body string, headers ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "https://parcel.example"+path, strings.NewReader(body))
	req.RemoteAddr = "192.0.2.1:1234"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// ownerSession sets the first password and returns the cookie of a sign-in.
func ownerSession(t *testing.T, h http.Handler) string {
	t.Helper()
	if rec := send(h, http.MethodPut, signin.APIPrefix+"me/password", `{"next":"a password for the owner"}`); rec.Code != http.StatusOK {
		t.Fatalf("first password: %d %s", rec.Code, rec.Body)
	}
	rec := send(h, http.MethodPost, signin.APIPrefix+"login", `{"password":"a password for the owner"}`)
	for _, c := range rec.Result().Cookies() {
		if strings.HasSuffix(c.Name, "session") {
			return c.Name + "=" + c.Value
		}
	}
	t.Fatalf("login: %d %s", rec.Code, rec.Body)
	return ""
}

func TestRouter_ARequestRemembersTheAccountThatOpenedIt(t *testing.T) {
	h, db := newRouterAndStore(t, withBaseURL)
	cookie := ownerSession(t, h)
	body := `{"publicKey":"cHVibGljcHVibGljcHVibGljcHVibGljcHVibGljcHU","collectTokenHash":"` +
		share.HashCollectToken("Y29sbGVjdGNvbGxlY3Rjb2xsZWN0Y29sbGVjdGNvbGw") + `"}`
	rec := send(h, http.MethodPost, "/api/requests", body, "Cookie", cookie)
	var made struct{ Slug string }
	if _ = json.Unmarshal(rec.Body.Bytes(), &made); rec.Code != http.StatusOK || made.Slug == "" {
		t.Fatalf("open a request: %d %s", rec.Code, rec.Body)
	}
	me := send(h, http.MethodGet, signin.APIPrefix+"me", "", "Cookie", cookie)
	var who struct{ Account string }
	_ = json.Unmarshal(me.Body.Bytes(), &who)
	got, err := store.GetRequest(db, made.Slug)
	if err != nil || got == nil || who.Account == "" || got.Account != who.Account {
		t.Errorf("request account = %+v, %v, want %q", got, err, who.Account)
	}
}

func TestRouter_AnUploadStopsWithTheSessionThatStartedIt(t *testing.T) {
	h := newFullRouter(t, withBaseURL)
	cookie := ownerSession(t, h)
	start := send(h, http.MethodPost, "/files/", "", "Cookie", cookie, "Tus-Resumable", "1.0.0", "Upload-Length", "10")
	where := start.Header().Get("Location")
	if start.Code != http.StatusCreated || where == "" {
		t.Fatalf("start an upload: %d %s", start.Code, start.Body)
	}
	if rec := send(h, http.MethodPost, signin.APIPrefix+"logout", "", "Cookie", cookie); rec.Code != http.StatusOK {
		t.Fatalf("logout: %d %s", rec.Code, rec.Body)
	}
	path := where[strings.Index(where, "/files/"):]
	req := httptest.NewRequest(http.MethodPatch, "https://parcel.example"+path, strings.NewReader("0123456789"))
	req.RemoteAddr = "192.0.2.1:1234"
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Tus-Resumable", "1.0.0")
	req.Header.Set("Upload-Offset", "0")
	req.Header.Set("Content-Type", "application/offset+octet-stream")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("PATCH after the session ended: %d %s", rec.Code, rec.Body)
	}
}
