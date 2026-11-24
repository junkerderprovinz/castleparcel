package notices

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/junkerderprovinz/parleyport/kit/notify"
)

// maxBody bounds what the card sends, which is a handful of channels.
const maxBody = 256 << 10

// Handler serves the card's three calls: GET /api/notify for the channels,
// PUT /api/notify to keep them and POST /api/notify/test to try one. The
// router guards who may reach it.
func (h *Hub) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/notify", func(w http.ResponseWriter, _ *http.Request) {
		answer(w, http.StatusOK, h.Config())
	})
	mux.HandleFunc("PUT /api/notify", func(w http.ResponseWriter, r *http.Request) {
		var next Config
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody)).Decode(&next); err != nil {
			answer(w, http.StatusBadRequest, map[string]string{"error": "badRequest"})
			return
		}
		var problems notify.Problems
		switch err := h.Save(next); {
		case errors.As(err, &problems):
			answer(w, http.StatusBadRequest, map[string]any{"problems": problems})
		case err != nil:
			h.log.Error("notify: keep the channels", "err", err)
			answer(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		default:
			answer(w, http.StatusOK, map[string]bool{"ok": true})
		}
	})
	mux.HandleFunc("POST /api/notify/test", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Channel notify.Channel `json:"channel"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody)).Decode(&body); err != nil {
			answer(w, http.StatusBadRequest, map[string]string{"error": "badRequest"})
			return
		}
		if res := h.Test(r.Context(), body.Channel); !res.OK() {
			answer(w, http.StatusOK, map[string]any{"ok": false, "code": res.Code})
			return
		}
		answer(w, http.StatusOK, map[string]bool{"ok": true})
	})
	return mux
}

func answer(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// AfterUpload tells the channels once an upload to a file request went
// through next.
func (h *Hub) AfterUpload(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/uploads") && rec.status < 300 {
			h.Send(RequestUpload, nil)
		}
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// The share of STORAGE_QUOTA from which the store counts as almost full, and
// the one it has to fall under before a second warning.
const (
	fullAt  = 90
	clearAt = 80
)

// Storage warns once the store has used fullAt percent of quota, and not
// again until it fell under clearAt. Without a quota it says nothing.
func (h *Hub) Storage(used, quota int64) {
	if quota <= 0 {
		return
	}
	percent := used * 100 / quota
	h.mu.Lock()
	warn := percent >= fullAt && !h.full
	switch {
	case percent >= fullAt:
		h.full = true
	case percent < clearAt:
		h.full = false
	}
	h.mu.Unlock()
	if warn {
		h.Send(StorageFull, map[string]string{"percent": strconv.FormatInt(percent, 10) + " %"})
	}
}
