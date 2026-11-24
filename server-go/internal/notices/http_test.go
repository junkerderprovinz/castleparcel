package notices

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/junkerderprovinz/parleyport/kit/notify"
)

func do(h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(body)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(method, path, &buf))
	return w
}

func TestTheCardLoadsSavesAndTests(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1) }))
	defer srv.Close()
	hub, _ := Open(t.TempDir(), key, nil)
	h := hub.Handler()

	if w := do(h, "PUT", "/api/notify", Config{Channels: []notify.Channel{webhook("hook", srv.URL, "secret-token")}, Lang: "fr"}); w.Code != http.StatusOK {
		t.Fatalf("PUT: %d %s", w.Code, w.Body)
	}
	w := do(h, "GET", "/api/notify", nil)
	if w.Code != http.StatusOK || strings.Contains(w.Body.String(), "secret-token") || !strings.Contains(w.Body.String(), `"lang":"fr"`) {
		t.Errorf("GET: %d %s", w.Code, w.Body)
	}
	var shown Config
	_ = json.Unmarshal(w.Body.Bytes(), &shown)
	if w := do(h, "POST", "/api/notify/test", map[string]any{"channel": shown.Channels[0]}); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"ok":true`) || hits.Load() != 1 {
		t.Errorf("test: %d %s, %d hits", w.Code, w.Body, hits.Load())
	}
	w = do(h, "PUT", "/api/notify", Config{Channels: []notify.Channel{webhook("hook", "nope", "t")}})
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), `"problems"`) {
		t.Errorf("PUT of a broken channel: %d %s", w.Code, w.Body)
	}
}

func TestAnUploadToARequestIsTold(t *testing.T) {
	got := make(chan string, 2)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got <- string(b)
	}))
	defer srv.Close()
	hub, _ := Open(t.TempDir(), key, nil)
	_ = hub.Save(Config{Channels: []notify.Channel{webhook("hook", srv.URL, "t")}})
	status := http.StatusOK
	h := hub.AfterUpload(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/r/slug/uploads", nil))
	select {
	case body := <-got:
		if !strings.Contains(body, Text("en", RequestUpload).Title) {
			t.Errorf("sent %s", body)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("nothing was told")
	}
	status = http.StatusNotFound
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/r/slug/uploads", nil))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/api/r/slug/uploads", nil))
	select {
	case body := <-got:
		t.Errorf("a refused upload or a listing was told: %s", body)
	case <-time.After(300 * time.Millisecond):
	}
}

func TestAFillingStoreIsToldOnceUntilItEmptiesAgain(t *testing.T) {
	got := make(chan string, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got <- string(b)
	}))
	defer srv.Close()
	hub, _ := Open(t.TempDir(), key, nil)
	_ = hub.Save(Config{Channels: []notify.Channel{webhook("hook", srv.URL, "t")}})
	told := func() int {
		n := 0
		for {
			select {
			case <-got:
				n++
			case <-time.After(500 * time.Millisecond):
				return n
			}
		}
	}
	for _, used := range []int64{50, 91, 95} {
		hub.Storage(used, 100)
	}
	if n := told(); n != 1 {
		t.Errorf("filling up told %d times, want 1", n)
	}
	hub.Storage(70, 100)
	hub.Storage(92, 100)
	if n := told(); n != 1 {
		t.Errorf("filling up again told %d times, want 1", n)
	}
	hub.Storage(99, 0)
	if n := told(); n != 0 {
		t.Errorf("without a quota told %d times", n)
	}
}
