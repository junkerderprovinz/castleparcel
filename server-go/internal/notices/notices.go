// Package notices sends what happens in CastleParcel to the services the
// owner set up, through the kit's notify. The channels live sealed in
// CONFIG_DIR, since they hold tokens and passwords of other services.
package notices

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/junkerderprovinz/parleyport/kit/notify"
	"github.com/junkerderprovinz/parleyport/kit/signin"
)

// The event kinds CastleParcel sends. A channel's events list names them.
const (
	RequestUpload = "request.upload"
	StorageFull   = "storage.full"
	PasswordReset = "signin.reset"
)

// Kinds are the event kinds in the order the card lists them.
var Kinds = []string{RequestUpload, StorageFull, PasswordReset}

var severity = map[string]notify.Severity{
	RequestUpload: notify.SeverityInfo,
	StorageFull:   notify.SeverityWarning,
	PasswordReset: notify.SeverityWarning,
}

const (
	fileName  = "notify.json"
	sealLabel = "cp/notify/v1"
	// sendFor bounds one event across every channel and its retries.
	sendFor = 2 * time.Minute
)

// Config is what the card loads and saves: the channels, and the language
// the messages go out in, which the card sends along from the page.
type Config struct {
	Channels []notify.Channel `json:"channels"`
	Lang     string           `json:"lang,omitempty"`
}

// Hub keeps the channels and sends to them.
type Hub struct {
	path string
	seal signin.Sealer
	log  *slog.Logger

	mu   sync.Mutex
	cfg  Config
	send *notify.Dispatcher
	// full is whether the last look at the store found it almost full.
	full bool
}

// Open reads the channels from dir, sealed under key. A file it cannot open
// stops the start, so a wrong key never loses the channels on the next save.
func Open(dir string, key []byte, log *slog.Logger) (*Hub, error) {
	seal, err := signin.NewSealer(key)
	if err != nil {
		return nil, err
	}
	if log == nil {
		log = slog.Default()
	}
	h := &Hub{path: filepath.Join(dir, fileName), seal: seal, log: log}
	raw, err := os.ReadFile(h.path)
	if errors.Is(err, os.ErrNotExist) {
		return h, h.use(Config{})
	}
	if err != nil {
		return nil, err
	}
	plain, err := seal.Open(sealLabel, raw)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", h.path, err)
	}
	var cfg Config
	if err := json.Unmarshal(plain, &cfg); err != nil {
		return nil, fmt.Errorf("read %s: %w", h.path, err)
	}
	return h, h.use(cfg)
}

func (h *Hub) use(cfg Config) error {
	d, err := notify.NewDispatcher(cfg.Channels, notify.Options{Source: "CastleParcel"})
	if err != nil {
		return err
	}
	h.cfg, h.send = cfg, d
	return nil
}

// Config returns the channels fit for a browser, every secret masked.
func (h *Hub) Config() Config {
	h.mu.Lock()
	defer h.mu.Unlock()
	return Config{Channels: notify.Redacted(h.cfg.Channels), Lang: h.cfg.Lang}
}

// Save takes the channels the card sends, with the secrets it never saw put
// back, and keeps them. Channels that cannot work come back as
// notify.Problems, and the old ones stay.
func (h *Hub) Save(next Config) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	next.Channels = notify.Merge(next.Channels, h.cfg.Channels)
	if next.Channels == nil {
		next.Channels = []notify.Channel{}
	}
	if err := notify.Validate(next.Channels); err != nil {
		return err
	}
	plain, err := json.Marshal(next)
	if err != nil {
		return err
	}
	sealed, err := h.seal.Seal(sealLabel, plain)
	if err != nil {
		return err
	}
	tmp := h.path + ".new"
	if err := os.WriteFile(tmp, sealed, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, h.path); err != nil {
		return err
	}
	return h.use(next)
}

// Test sends a test message to ch, with the stored secrets of the channel it
// stands for.
func (h *Hub) Test(ctx context.Context, ch notify.Channel) notify.Result {
	h.mu.Lock()
	stored, lang, d := h.cfg.Channels, h.cfg.Lang, h.send
	h.mu.Unlock()
	ch = notify.Merge([]notify.Channel{ch}, stored)[0]
	if err := notify.Validate([]notify.Channel{ch}); err != nil {
		return notify.Result{Channel: ch.ID, Kind: ch.Kind, Code: notify.CodeInvalid, Err: err}
	}
	t := Text(lang, "test")
	return d.Test(ctx, ch, notify.Event{Kind: "test", Severity: notify.SeveritySuccess, Title: t.Title, Body: t.Body})
}

// Send tells every channel that wants it about kind, in the background.
// params fill the {names} of the message.
func (h *Hub) Send(kind string, params map[string]string) {
	h.mu.Lock()
	lang, d := h.cfg.Lang, h.send
	h.mu.Unlock()
	t := Text(lang, kind)
	e := notify.Event{Kind: kind, Severity: severity[kind], Title: fill(t.Title, params), Body: fill(t.Body, params)}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), sendFor)
		defer cancel()
		for _, r := range d.Send(ctx, e) {
			if !r.OK() {
				h.log.Warn("notify: a channel did not take the message", "channel", r.Channel, "event", kind, "err", r.Err)
			}
		}
	}()
}

func fill(s string, params map[string]string) string {
	for name, v := range params {
		s = strings.ReplaceAll(s, "{"+name+"}", v)
	}
	return s
}

// Message is one event kind in one language: its name on the card and what
// the message says.
type Message struct {
	Label string `json:"label,omitempty"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

// texts.json is written from the client's language files by
// scripts/gen-notice-texts.mjs.
//
//go:embed texts.json
var textsJSON []byte

var texts = func() map[string]map[string]Message {
	var t map[string]map[string]Message
	if err := json.Unmarshal(textsJSON, &t); err != nil {
		panic(err)
	}
	return t
}()

// Text is kind's message in lang, in English where lang has none.
func Text(lang, kind string) Message {
	if m, ok := texts[lang][kind]; ok {
		return m
	}
	return texts["en"][kind]
}
