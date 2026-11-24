package fortress

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	kitfortress "github.com/junkerderprovinz/parleyport/kit/fortress"
	"github.com/junkerderprovinz/parleyport/kit/group"
	"github.com/junkerderprovinz/parleyport/kit/phrase"
	"github.com/junkerderprovinz/parleyport/kit/signin"
	"github.com/junkerderprovinz/parleyport/kit/status"
)

// callWait bounds a call this instance makes at another member for its user.
const callWait = 15 * time.Second

// Routes serves the fortress to the owner under /api/fortress. Who may reach
// them is the caller's to decide; account.Owner does.
func (s *Service) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/fortress", s.read)
	mux.HandleFunc("POST /api/fortress/join", s.join)
	mux.HandleFunc("DELETE /api/fortress", s.leave)
	mux.HandleFunc("DELETE /api/fortress/members/{id}", s.remove)
	mux.HandleFunc("PUT /api/fortress/permissions", s.permission)
	mux.HandleFunc("PUT /api/fortress/network", s.setNetwork)
	mux.HandleFunc("POST /api/fortress/phrase/show", s.showPhrase)
	mux.HandleFunc("POST /api/fortress/phrase", s.changePhrase)
	mux.HandleFunc("POST /api/fortress/open/{id}", s.open)
	mux.HandleFunc("GET /api/fortress/members/{id}/status", s.memberStatus)
	mux.HandleFunc("/api/fortress/", func(w http.ResponseWriter, _ *http.Request) { fail(w, http.StatusNotFound, "notFound") })
	return mux
}

// Direct answers the other members' calls on this network at
// group.DirectPath. The calls are sealed, so nothing of the sign-in counts.
func (s *Service) Direct() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f := s.current()
		if f == nil {
			http.NotFound(w, r)
			return
		}
		f.DirectHandler().ServeHTTP(w, r)
	})
}

// Tabs serves, at kitfortress.OpenPath, the tabs in which the owner opens
// another member's pages: the kit's bar above a sandboxed frame. The kit
// checks the session the tab was opened for, and sets the headers of every
// answer itself.
func (s *Service) Tabs() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f := s.current()
		if f == nil {
			http.NotFound(w, r)
			return
		}
		f.OpenHandler(func(r *http.Request) (string, bool) {
			name, _, ok := s.tabSession(r)
			return name, ok
		}, nil).ServeHTTP(w, r)
	})
}

// visitorCookie names a browser while CastleParcel is open, so that a tab
// opened there belongs to that browser alone, as it belongs to a sign-in
// once there is a password. visitorLife is as long as a tab lives at most.
const (
	visitorCookie = "cp_visitor"
	visitorLife   = 12 * time.Hour
)

// tabSession names the browser session of r without the secret it carries,
// and reports whether it still stands: the owner's sign-in, or the visitor
// cookie while CastleParcel is open.
func (s *Service) tabSession(r *http.Request) (string, func() bool, bool) {
	auth := s.cfg.Auth
	if auth.Anyone() {
		c, err := r.Cookie(visitorCookie)
		if err != nil || c.Value == "" {
			return "", nil, false
		}
		return sessionName(c.Value), auth.Anyone, true
	}
	v, ok := signin.From(r.Context())
	if !ok || v.Session() == "" {
		return "", nil, false
	}
	id := v.Session()
	return sessionName(id), func() bool { return auth.Alive(context.Background(), id) }, true
}

func sessionName(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// view is what GET /api/fortress answers. It never carries the phrase.
type view struct {
	Joined bool `json:"joined"`
	// RemovedHere says another member took this instance out, and it has
	// not joined again since.
	RemovedHere bool `json:"removedHere"`
	// PhraseChanged says a member told this instance that the fortress has a
	// new phrase, which nobody in reach hands over.
	PhraseChanged bool             `json:"phraseChanged"`
	Device        string           `json:"device"`
	Relay         relayView        `json:"relay"`
	LAN           bool             `json:"lan"`
	Members       []memberView     `json:"members"`
	Permissions   []permissionView `json:"permissions"`
	Removed       []Removed        `json:"removed"`
}

type relayView struct {
	Mode      group.RelayMode `json:"mode"`
	URL       string          `json:"url"`
	Connected bool            `json:"connected"`
}

// memberView is another member. App and Name are empty for one known by its
// device alone, which never said which app it runs.
type memberView struct {
	ID         string `json:"id"`
	App        string `json:"app"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	Icon       string `json:"icon,omitempty"`
	Device     string `json:"device"`
	Deployment string `json:"deployment"`
	Host       string `json:"host,omitempty"`
	Online     bool   `json:"online"`
	// Relay means it is reached through the relay alone.
	Relay    bool  `json:"relay"`
	LastSeen int64 `json:"lastSeen,omitempty"`
	// Pages says whether its pages open from here.
	Pages bool `json:"pages"`
}

type permissionView struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Action string `json:"action"`
	On     bool   `json:"on"`
}

func (s *Service) view() view {
	st := s.settings()
	mode, url := st.relay()
	v := view{
		RemovedHere: st.Kicked, Device: s.cfg.Self.Name, LAN: !st.NoLAN,
		Relay:   relayView{Mode: mode, URL: url},
		Members: []memberView{}, Permissions: []permissionView{}, Removed: []Removed{},
	}
	m, in, _ := s.membership()
	f := s.current()
	if !in || f == nil {
		return v
	}
	v.Joined = true
	v.Removed = append(v.Removed, m.Removed...)
	v.PhraseChanged = f.PhraseChanged()
	v.Relay.Connected = f.RelayConnected()
	for _, m := range f.Members() {
		if m.State != group.StateApproved {
			continue
		}
		mv := memberView{
			ID: m.ID, Device: m.Device, Deployment: m.Deployment, Host: m.Host,
			Online: m.Online(), Relay: m.Relay && !m.Direct,
		}
		if m.App.ID != "" {
			mv.App, mv.Name, mv.Version, mv.Icon = m.App.ID, m.App.Name, m.App.Version, m.App.Icon
			mv.Pages = slices.Contains(m.Actions, kitfortress.OpenAction)
		}
		if !m.LastSeen.IsZero() {
			mv.LastSeen = m.LastSeen.Unix()
		}
		v.Members = append(v.Members, mv)
	}
	for _, p := range f.Permissions() {
		if p.From == s.cfg.App.ID || p.To == s.cfg.App.ID {
			v.Permissions = append(v.Permissions, permissionView{From: p.From, To: p.To, Action: p.Action, On: p.On})
		}
	}
	return v
}

func (s *Service) read(w http.ResponseWriter, r *http.Request) {
	if f := s.current(); f != nil {
		f.NoteAddress(r)
	}
	writeJSON(w, s.view())
}

// join joins the fortress of a phrase. Words that do not decode are refused
// with the code "phrase", the reason and the word at fault.
func (s *Service) join(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Phrase string `json:"phrase"`
	}
	if !readBody(w, r, &req) {
		return
	}
	secret, err := phrase.Decode(strings.TrimSpace(req.Phrase))
	if err != nil {
		var de *phrase.DecodeError
		reason := ""
		if errors.As(err, &de) {
			reason = string(de.Reason)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "phrase", "reason": reason})
		return
	}
	if err := s.Join(secret); err != nil {
		s.refuse(w, err)
		return
	}
	writeJSON(w, s.view())
}

func (s *Service) leave(w http.ResponseWriter, _ *http.Request) {
	if err := s.Leave(); err != nil {
		s.refuse(w, err)
		return
	}
	writeJSON(w, s.view())
}

func (s *Service) remove(w http.ResponseWriter, r *http.Request) {
	if err := s.Remove(r.PathValue("id")); err != nil {
		s.refuse(w, err)
		return
	}
	writeJSON(w, s.view())
}

func (s *Service) permission(w http.ResponseWriter, r *http.Request) {
	var req permissionView
	if !readBody(w, r, &req) {
		return
	}
	f := s.current()
	if f == nil {
		s.refuse(w, errNoFortress)
		return
	}
	if err := f.SetPermission(req.From, req.To, req.Action, req.On); err != nil {
		fail(w, http.StatusBadRequest, "badRequest")
		return
	}
	writeJSON(w, s.view())
}

// setNetwork takes the relay, as "project", "off" or "own" with the address
// of the own one, and whether the members are looked for on this network.
func (s *Service) setNetwork(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Relay string `json:"relay"`
		URL   string `json:"url"`
		LAN   bool   `json:"lan"`
	}
	if !readBody(w, r, &req) {
		return
	}
	at := ""
	switch req.Relay {
	case string(group.RelayProject):
	case string(group.RelayOff):
		at = "off"
	case string(group.RelayOwn):
		if !relayAddress(req.URL) {
			fail(w, http.StatusBadRequest, "relayAddress")
			return
		}
		at = strings.TrimSpace(req.URL)
	default:
		fail(w, http.StatusBadRequest, "badRequest")
		return
	}
	if err := s.SetNetwork(at, req.LAN); err != nil {
		s.refuse(w, err)
		return
	}
	writeJSON(w, s.view())
}

// relayAddress reports whether a is where an own relay could answer.
func relayAddress(a string) bool {
	u, err := url.Parse(strings.TrimSpace(a))
	return err == nil && u.Host != "" && u.User == nil && slices.Contains([]string{"http", "https", "ws", "wss"}, u.Scheme)
}

// showPhrase shows the phrase to an owner who signed in moments ago, since
// it lets any device into the fortress.
func (s *Service) showPhrase(w http.ResponseWriter, r *http.Request) {
	if !s.signedInMoments(w, r) {
		return
	}
	words, code, err := s.Phrase()
	if err != nil {
		s.refuse(w, err)
		return
	}
	writeJSON(w, map[string]any{"words": strings.Fields(words), "code": code})
}

// changePhrase gives the fortress a new phrase. The answer shows it, so it
// wants a fresh sign-in as showing it does.
func (s *Service) changePhrase(w http.ResponseWriter, r *http.Request) {
	if !s.signedInMoments(w, r) {
		return
	}
	words, left, err := s.ChangePhrase(r.Context())
	if err != nil {
		s.refuse(w, err)
		return
	}
	_, code, _ := s.Phrase()
	writeJSON(w, map[string]any{"words": words, "code": code, "left": left})
}

// signedInMoments answers a visitor who has to sign in again first, as the
// sign-in does, and reports whether the visitor signed in moments ago.
// While CastleParcel is open every visitor counts as such.
func (s *Service) signedInMoments(w http.ResponseWriter, r *http.Request) bool {
	err := s.cfg.Auth.Mint(r, func() error { return nil })
	switch {
	case errors.Is(err, signin.ErrNotFresh):
		s.cfg.Auth.SignInAgain(w, r)
		return false
	case err != nil:
		s.refuse(w, err)
		return false
	}
	return true
}

// open opens another member's pages in a tab of their own and answers its
// path. Without a password here the browser gets a visitor cookie for it,
// and setting a password closes the tab.
func (s *Service) open(w http.ResponseWriter, r *http.Request) {
	f := s.current()
	if f == nil {
		s.refuse(w, errNoFortress)
		return
	}
	name, valid, ok := s.tabSession(r)
	if !ok && s.cfg.Auth.Anyone() {
		c := &http.Cookie{
			Name: visitorCookie, Value: rand.Text(), Path: "/", HttpOnly: true, MaxAge: int(visitorLife / time.Second),
			SameSite: http.SameSiteLaxMode, Secure: r.TLS != nil || strings.HasPrefix(s.cfg.Self.Address, "https:"),
		}
		http.SetCookie(w, c)
		name, valid, ok = sessionName(c.Value), s.cfg.Auth.Anyone, true
	}
	if !ok {
		fail(w, http.StatusForbidden, "noSession")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), callWait)
	defer cancel()
	tab, err := f.Open(ctx, kitfortress.Visit{Member: r.PathValue("id"), Session: name, Valid: valid, Back: "/settings/fortress"})
	if err != nil {
		s.refuseOpen(w, err)
		return
	}
	writeJSON(w, map[string]string{"path": tab})
}

// refuseOpen answers why a member did not open or report, with the code
// GlimStone has words for.
func (s *Service) refuseOpen(w http.ResponseWriter, err error) {
	status, code := http.StatusBadGateway, "failed"
	var no *kitfortress.Refusal
	switch {
	case errors.Is(err, group.ErrNoSuchMember), errors.Is(err, group.ErrRemoved):
		status, code = http.StatusNotFound, "unreachable"
	case errors.Is(err, group.ErrUnreachable), errors.Is(err, context.DeadlineExceeded):
		code = "unreachable"
	case errors.Is(err, kitfortress.ErrNoSuchAction):
		status, code = http.StatusNotFound, "no_ui"
	case errors.Is(err, kitfortress.ErrPermissionOff), errors.Is(err, kitfortress.ErrNotUsed):
		status, code = http.StatusForbidden, "permission_off"
	case errors.As(err, &no) && no.Code == "ui_no_login":
		status, code = http.StatusForbidden, "ui_no_login"
	case errors.As(err, &no) && (no.Code == "busy" || no.Code == "jars_full"):
		status, code = http.StatusServiceUnavailable, "busy"
	default:
		s.cfg.Log.Warn("fortress: another member did not answer", "err", err)
	}
	fail(w, status, code)
}

// memberStatusLines is how many lines of a member's log its status window
// shows, and memberLogPages how many pages of the log are read for them.
const (
	memberStatusLines = 10
	memberLogPages    = 5
)

// memberStatus answers another member's status report and the end of its
// log, newest first, for a member whose pages do not open from here.
func (s *Service) memberStatus(w http.ResponseWriter, r *http.Request) {
	f := s.current()
	if f == nil {
		s.refuse(w, errNoFortress)
		return
	}
	id, lang := r.PathValue("id"), r.URL.Query().Get("lang")
	ctx, cancel := context.WithTimeout(r.Context(), callWait)
	defer cancel()
	rep, err := f.Status(ctx, id, lang)
	if err != nil {
		s.refuseOpen(w, err)
		return
	}
	var tail []status.LogEntry
	after := ""
	for range memberLogPages {
		page, err := f.Log(ctx, id, status.LogQuery{After: after, Limit: status.MaxLogLimit, Lang: lang})
		if err != nil {
			break
		}
		tail = append(tail, page.Entries...)
		tail = tail[max(0, len(tail)-memberStatusLines):]
		if len(page.Entries) < status.MaxLogLimit || page.Next == after {
			break
		}
		after = page.Next
	}
	slices.Reverse(tail)
	writeJSON(w, map[string]any{"report": rep, "log": tail})
}

// refuse answers err with the code the page words it by.
func (s *Service) refuse(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errInFortress):
		fail(w, http.StatusConflict, "inFortress")
	case errors.Is(err, errNoFortress):
		fail(w, http.StatusConflict, "noFortress")
	case errors.Is(err, errNoMember):
		fail(w, http.StatusNotFound, "noMember")
	case errors.Is(err, errClosed):
		fail(w, http.StatusConflict, group.ClosedUntilNewPhrase)
	default:
		s.cfg.Log.Error("fortress: a request failed", "err", err)
		fail(w, http.StatusInternalServerError, "internal")
	}
}

// maxBody bounds what a request of the fortress's page carries.
const maxBody = 16 << 10

func readBody(w http.ResponseWriter, r *http.Request, v any) bool {
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBody))
	if err != nil || json.Unmarshal(raw, v) != nil {
		fail(w, http.StatusBadRequest, "badRequest")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
}
