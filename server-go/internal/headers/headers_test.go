package headers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// directives splits a policy into its directives, keyed by name.
func directives(t *testing.T, policy string) map[string]string {
	t.Helper()
	out := make(map[string]string)
	for _, d := range strings.Split(policy, ";") {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		name, value, _ := strings.Cut(d, " ")
		if _, dup := out[name]; dup {
			t.Fatalf("directive %q appears twice in %q", name, policy)
		}
		out[name] = value
	}
	return out
}

func TestContentSecurityPolicy(t *testing.T) {
	got := directives(t, ContentSecurityPolicy(""))
	want := map[string]string{
		"default-src":     "'self'",
		"script-src":      "'self' 'wasm-unsafe-eval'",
		"style-src":       "'self' 'unsafe-inline'",
		"img-src":         "'self' blob: data:",
		"font-src":        "'self' data:",
		"media-src":       "'self' blob:",
		"object-src":      "blob:",
		"worker-src":      "'self' blob:",
		"connect-src":     "'self'",
		"frame-ancestors": "'none'",
		"base-uri":        "'none'",
		"form-action":     "'self'",
	}
	for name, value := range want {
		if got[name] != value {
			t.Errorf("%s = %q, want %q", name, got[name], value)
		}
	}
}

func TestContentSecurityPolicy_LogoOriginJoinsImgSrcOnly(t *testing.T) {
	got := directives(t, ContentSecurityPolicy("https://cdn.example.com"))
	if want := "'self' blob: data: https://cdn.example.com"; got["img-src"] != want {
		t.Errorf("img-src = %q, want %q", got["img-src"], want)
	}
	for name, value := range got {
		if name != "img-src" && strings.Contains(value, "cdn.example.com") {
			t.Errorf("%s = %q must not allow the logo origin", name, value)
		}
	}
}

func TestMiddleware(t *testing.T) {
	csp := ContentSecurityPolicy("")
	h := Middleware(csp)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want the handler's 418", rec.Code)
	}
	want := map[string]string{
		"Content-Security-Policy":    csp,
		"Referrer-Policy":            "no-referrer",
		"Cross-Origin-Opener-Policy": "same-origin",
		"X-Frame-Options":            "DENY",
		"X-Content-Type-Options":     "nosniff",
		"Permissions-Policy":         PermissionsPolicy,
	}
	for name, value := range want {
		if got := rec.Header().Get(name); got != value {
			t.Errorf("%s = %q, want %q", name, got, value)
		}
	}
}

func TestPermissionsPolicyDeniesPowerfulFeatures(t *testing.T) {
	for _, feature := range []string{"camera", "microphone", "geolocation", "payment", "usb"} {
		if !strings.Contains(PermissionsPolicy, feature+"=()") {
			t.Errorf("Permissions-Policy %q does not deny %s", PermissionsPolicy, feature)
		}
	}
	// The share link copy button and fullscreen video keep their defaults.
	for _, feature := range []string{"clipboard-write", "fullscreen"} {
		if strings.Contains(PermissionsPolicy, feature+"=()") {
			t.Errorf("Permissions-Policy %q must not deny %s", PermissionsPolicy, feature)
		}
	}
}

func TestPermissionsPolicyLetsTheOwnerUsePasskeys(t *testing.T) {
	for _, feature := range []string{"publickey-credentials-get", "publickey-credentials-create"} {
		if !strings.Contains(PermissionsPolicy, feature+"=(self)") {
			t.Errorf("Permissions-Policy %q does not allow %s on the page itself", PermissionsPolicy, feature)
		}
	}
}

func TestMiddleware_HandlerCanOverride(t *testing.T) {
	h := Middleware(ContentSecurityPolicy(""))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Referrer-Policy", "same-origin")
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if got := rec.Header().Get("Referrer-Policy"); got != "same-origin" {
		t.Errorf("Referrer-Policy = %q, want the handler's own value", got)
	}
}

// The fortress's tab frames another member's page under its bar, at the same
// address. Our frame-ancestors 'none' and DENY would keep that page out, and
// the kit gives the bar and the page policies of their own.
func TestMiddleware_LeavesTheirOwnPathsToTheFortressTabs(t *testing.T) {
	h := Middleware(ContentSecurityPolicy(""), "/fortress/open/")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	for path, framed := range map[string]bool{"/fortress/open/abc/": true, "/fortress/openly": false, "/": false} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if got := rec.Header().Get("X-Frame-Options") == ""; got != framed {
			t.Errorf("%s: X-Frame-Options %q", path, rec.Header().Get("X-Frame-Options"))
		}
	}
}
