package notices

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/junkerderprovinz/parleyport/kit/notify"
)

var key = []byte(strings.Repeat("k", 32))

func webhook(id, url, token string) notify.Channel {
	return notify.Channel{ID: id, Kind: notify.KindWebhook, Enabled: true, On: notify.OnAlways, Webhook: &notify.Webhook{
		URL: url, Method: http.MethodPost, Headers: map[string]string{"Authorization": "Bearer " + token}, Body: `{"title":"%%title%%"}`,
	}}
}

func TestChannelsAreKeptSealed(t *testing.T) {
	dir := t.TempDir()
	h, err := Open(dir, key, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.Save(Config{Channels: []notify.Channel{webhook("hook", "https://hooks.example/in", "secret-token")}, Lang: "de"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "secret-token") || strings.Contains(string(raw), "hooks.example") {
		t.Errorf("the file shows the channel in the clear: %s", raw)
	}

	again, err := Open(dir, key, nil)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	got := again.Config()
	if len(got.Channels) != 1 || got.Lang != "de" || strings.Contains(got.Channels[0].Webhook.Headers["Authorization"], "secret-token") {
		t.Errorf("Config after a restart = %+v", got)
	}
}

func TestASavedRedactionKeepsTheStoredSecret(t *testing.T) {
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { auth = r.Header.Get("Authorization") }))
	defer srv.Close()
	h, _ := Open(t.TempDir(), key, nil)
	if err := h.Save(Config{Channels: []notify.Channel{webhook("hook", srv.URL, "secret-token")}}); err != nil {
		t.Fatal(err)
	}
	shown := h.Config()
	shown.Channels[0].Name = "Renamed"
	if err := h.Save(shown); err != nil {
		t.Fatalf("Save of the redacted config: %v", err)
	}
	if r := h.Test(context.Background(), shown.Channels[0]); !r.OK() {
		t.Fatalf("Test: %+v", r)
	}
	if auth != "Bearer secret-token" {
		t.Errorf("the test sent %q", auth)
	}
}

func TestABrokenChannelIsRefusedAndTheOldOnesStay(t *testing.T) {
	h, _ := Open(t.TempDir(), key, nil)
	_ = h.Save(Config{Channels: []notify.Channel{webhook("hook", "https://hooks.example/in", "t")}})
	err := h.Save(Config{Channels: []notify.Channel{webhook("hook", "not a url", "t")}})
	var problems notify.Problems
	if !errors.As(err, &problems) || len(problems) == 0 {
		t.Fatalf("Save of a broken channel = %v", err)
	}
	if got := h.Config(); got.Channels[0].Webhook.URL == "not a url" {
		t.Errorf("the broken channel was kept")
	}
}

func TestAnEventReachesTheChannelsThatWantItInTheSavedLanguage(t *testing.T) {
	got := make(chan string, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got <- r.URL.Path + " " + string(b)
	}))
	defer srv.Close()
	only := webhook("only", srv.URL+"/only", "t")
	only.Events = []string{StorageFull}
	h, _ := Open(t.TempDir(), key, nil)
	if err := h.Save(Config{Channels: []notify.Channel{webhook("all", srv.URL+"/all", "t"), only}, Lang: "de"}); err != nil {
		t.Fatal(err)
	}
	h.Send(RequestUpload, nil)
	select {
	case body := <-got:
		var sent struct{ Title string }
		_ = json.Unmarshal([]byte(body[strings.Index(body, " ")+1:]), &sent)
		if !strings.HasPrefix(body, "/all ") || sent.Title != Text("de", RequestUpload).Title {
			t.Errorf("sent %s", body)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("nothing arrived")
	}
	select {
	case body := <-got:
		t.Errorf("a channel that wants other events got %s", body)
	case <-time.After(300 * time.Millisecond):
	}
}

func TestEveryLanguageHasTheEventTextsAndFallsBackToEnglish(t *testing.T) {
	for _, kind := range Kinds {
		en := Text("en", kind)
		if en.Title == "" || en.Body == "" || en.Label == "" {
			t.Errorf("%s has no English text", kind)
		}
		if Text("xx", kind) != en {
			t.Errorf("%s does not fall back to English", kind)
		}
		if Text("de", kind) == en {
			t.Errorf("%s is not translated into German", kind)
		}
	}
}
