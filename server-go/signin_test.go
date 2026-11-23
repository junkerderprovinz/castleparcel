package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/junkerderprovinz/parleyport/kit/signin"
	"github.com/junkerderprovinz/parleyport/kit/signin/signintest"

	"github.com/junkerderprovinz/castleparcel/server-go/internal/config"
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
