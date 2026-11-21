package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func newReleaseServer(t *testing.T, status int, body string) (*LatestRelease, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return &LatestRelease{URL: srv.URL, Client: srv.Client()}, &calls
}

func TestLatestReleaseAnswersWithTheTag(t *testing.T) {
	l, _ := newReleaseServer(t, http.StatusOK, `{"tag_name":"v1.4.0"}`)
	rec := httptest.NewRecorder()
	LatestReleaseHandler(l)(rec, httptest.NewRequest(http.MethodGet, "/api/latest-release", nil))
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
	l, calls := newReleaseServer(t, http.StatusOK, `{"tag_name":"v1.4.0"}`)
	for range 5 {
		if _, _, err := l.Get(); err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("GitHub asked %d times", calls.Load())
	}
	l.checked = time.Now().Add(-releaseFresh - time.Second)
	if _, _, err := l.Get(); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("a stale answer was kept, GitHub asked %d times", calls.Load())
	}
}

func TestLatestReleaseRemembersAFailureForAShorterWhile(t *testing.T) {
	l, calls := newReleaseServer(t, http.StatusForbidden, `{"message":"rate limited"}`)
	rec := httptest.NewRecorder()
	LatestReleaseHandler(l)(rec, httptest.NewRequest(http.MethodGet, "/api/latest-release", nil))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status %d", rec.Code)
	}
	_, _, _ = l.Get()
	if calls.Load() != 1 {
		t.Fatalf("GitHub asked %d times", calls.Load())
	}
	l.checked = time.Now().Add(-releaseFailed - time.Second)
	_, _, _ = l.Get()
	if calls.Load() != 2 {
		t.Fatalf("a failure was kept too long, GitHub asked %d times", calls.Load())
	}
}

func TestLatestReleaseRefusesAnAnswerWithoutATag(t *testing.T) {
	l, _ := newReleaseServer(t, http.StatusOK, `{}`)
	if _, _, err := l.Get(); err == nil {
		t.Fatal("no error for an answer without a tag")
	}
}
