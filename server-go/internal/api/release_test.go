package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

// newReleaseServer stands in for GitHub's release API and counts the calls.
func newReleaseServer(t *testing.T, status int) (*Releases, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(status)
		tag := strings.TrimPrefix(r.URL.Path, "/tags/")
		if tag == "/latest" {
			tag = "v1.4.0"
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": tag, "body": "## Fixed\n\n- " + tag, "html_url": "https://example.org/" + tag})
	}))
	t.Cleanup(srv.Close)
	return &Releases{URL: srv.URL, Client: srv.Client()}, &calls
}

func notesRequest(r *Releases, tag string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/release-notes/"+tag, nil)
	route := chi.NewRouteContext()
	route.URLParams.Add("tag", tag)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, route))
	rec := httptest.NewRecorder()
	ReleaseNotesHandler(r)(rec, req)
	return rec
}

func TestLatestReleaseAnswersWithTheTag(t *testing.T) {
	r, _ := newReleaseServer(t, http.StatusOK)
	rec := httptest.NewRecorder()
	LatestReleaseHandler(r)(rec, httptest.NewRequest(http.MethodGet, "/api/latest-release", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body)
	}
	var got latestReleaseResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Latest != "v1.4.0" || time.Since(got.CheckedAt) > time.Minute {
		t.Fatalf("got %+v", got)
	}
}

func TestLatestReleaseAsksGitHubOncePerWindow(t *testing.T) {
	r, calls := newReleaseServer(t, http.StatusOK)
	for range 5 {
		if _, _, err := r.get("latest"); err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("GitHub asked %d times", calls.Load())
	}
	e := r.entries["latest"]
	e.checked = time.Now().Add(-releaseFresh - time.Second)
	r.entries["latest"] = e
	if _, _, err := r.get("latest"); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("a stale answer was kept, GitHub asked %d times", calls.Load())
	}
}

func TestLatestReleaseRemembersAFailureForAShorterWhile(t *testing.T) {
	r, calls := newReleaseServer(t, http.StatusForbidden)
	rec := httptest.NewRecorder()
	LatestReleaseHandler(r)(rec, httptest.NewRequest(http.MethodGet, "/api/latest-release", nil))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status %d", rec.Code)
	}
	_, _, _ = r.get("latest")
	if calls.Load() != 1 {
		t.Fatalf("GitHub asked %d times", calls.Load())
	}
	e := r.entries["latest"]
	e.checked = time.Now().Add(-releaseFailed - time.Second)
	r.entries["latest"] = e
	_, _, _ = r.get("latest")
	if calls.Load() != 2 {
		t.Fatalf("a failure was kept too long, GitHub asked %d times", calls.Load())
	}
}

func TestReleaseNotesAnswerWithTheNotesOfTheTag(t *testing.T) {
	r, calls := newReleaseServer(t, http.StatusOK)
	for range 3 {
		rec := notesRequest(r, "v2.2.0")
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d, body %s", rec.Code, rec.Body)
		}
		var got releaseNotesResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.Tag != "v2.2.0" || got.Notes != "## Fixed\n\n- v2.2.0" || got.URL != "https://example.org/v2.2.0" {
			t.Fatalf("got %+v", got)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("GitHub asked %d times", calls.Load())
	}
}

func TestReleaseNotesRefuseWhatIsNotATag(t *testing.T) {
	r, calls := newReleaseServer(t, http.StatusOK)
	for _, tag := range []string{"latest", "2.2.0", "v2.2", "v2.2.0-rc1", "..%2Fv2.2.0"} {
		if rec := notesRequest(r, tag); rec.Code != http.StatusNotFound {
			t.Errorf("%s: status %d", tag, rec.Code)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("GitHub asked %d times", calls.Load())
	}
}

func TestReleaseNotesKeepABoundedMemory(t *testing.T) {
	r, _ := newReleaseServer(t, http.StatusOK)
	for i := range releaseKeys + 10 {
		notesRequest(r, fmt.Sprintf("v1.0.%d", i))
	}
	if len(r.entries) > releaseKeys {
		t.Fatalf("%d answers kept", len(r.entries))
	}
}
