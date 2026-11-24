// Package fortress is CastleParcel's place in a fortress, the apps of one
// household joined under one phrase. The owner joins with the phrase and
// leaves, sees the other members, their status and log, takes one out,
// switches what the apps may do at each other and opens the pages of the
// members that offer them. CastleParcel offers no actions and no pages of its
// own; the members read its status and log.
//
// The membership, the network settings and the kit's own state live in the
// database, sealed with CastleParcel's key. An install that never joined has
// none of them and runs as before.
package fortress

import (
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	kitfortress "github.com/junkerderprovinz/parleyport/kit/fortress"
	"github.com/junkerderprovinz/parleyport/kit/group"
	"github.com/junkerderprovinz/parleyport/kit/lan"
	"github.com/junkerderprovinz/parleyport/kit/member"
	"github.com/junkerderprovinz/parleyport/kit/phrase"
	"github.com/junkerderprovinz/parleyport/kit/relay"
	"github.com/junkerderprovinz/parleyport/kit/signin"
	"github.com/junkerderprovinz/parleyport/kit/status"

	"github.com/junkerderprovinz/castleparcel/server-go/internal/report"
	"github.com/junkerderprovinz/castleparcel/server-go/internal/store"
)

var (
	errInFortress = errors.New("this CastleParcel is in a fortress already")
	errNoFortress = errors.New("this CastleParcel is in no fortress")
	errNoMember   = errors.New("no other member of the fortress has that id")
	// errClosed is a join the members turned away: one of them was removed,
	// and the fortress takes no new app until its phrase changes.
	errClosed = errors.New("the fortress takes no new app until its phrase changes")
)

// previousKept is how long the fortress still answers on the phrase before a
// change, so that the members away for it follow.
const previousKept = 30 * 24 * time.Hour

// The documents the fortress keeps in the database.
const (
	docMembership = "membership"
	docSettings   = "settings"
	docBook       = "book"
)

// membership is this instance's place in a fortress. Removed are the members
// taken out here since the phrase last changed.
type membership struct {
	Secret   []byte    `json:"secret"`
	Seed     []byte    `json:"seed"`
	Previous []byte    `json:"previous,omitempty"`
	Changed  time.Time `json:"changed,omitzero"`
	Removed  []Removed `json:"removed,omitempty"`
}

// Removed is a member taken out, by the names it went by.
type Removed struct {
	App    string `json:"app"`
	Device string `json:"device"`
}

// settings outlast a membership. Relay is "" for the project's relay, "off"
// or the address of an own one.
type settings struct {
	Relay  string `json:"relay,omitempty"`
	NoLAN  bool   `json:"noLAN,omitempty"`
	Kicked bool   `json:"kicked,omitempty"`
}

func (st settings) relay() (group.RelayMode, string) {
	switch st.Relay {
	case "":
		return group.RelayProject, ""
	case "off":
		return group.RelayOff, ""
	}
	return group.RelayOwn, st.Relay
}

// Config is what the fortress runs on.
type Config struct {
	DB   *sql.DB
	Seal signin.Sealer
	Auth *signin.Auth
	App  kitfortress.App
	// Self is what the other members see of this instance.
	Self group.Self
	// Host names the server CastleParcel runs on, such as Unraid's name.
	Host   string
	Status status.Source
	// Ports are where a sweep of the network looks for CastleParcel.
	Ports []int
	Log   *slog.Logger
}

// Service is CastleParcel's side of its fortress.
type Service struct {
	cfg Config
	// mu keeps joining, leaving, changing the phrase and the network one at
	// a time.
	mu sync.Mutex

	live     sync.Mutex
	fort     *kitfortress.Fortress
	network  *lan.Service
	forget   *time.Timer
	changing bool
	gen      uint64
}

// New returns the fortress of cfg, closed until Start.
func New(cfg Config) *Service {
	return &Service{cfg: cfg}
}

// Start opens the fortress this instance is in, if it is in one.
func (s *Service) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.openLocked()
}

// Close leaves the network and saves what the fortress learned.
func (s *Service) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closeLocked()
}

// Status is what the members and the dashboards read about this instance.
func (s *Service) Status() status.Source { return s.cfg.Status }

func (s *Service) current() *kitfortress.Fortress {
	s.live.Lock()
	defer s.live.Unlock()
	return s.fort
}

func (s *Service) generation() uint64 {
	s.live.Lock()
	defer s.live.Unlock()
	return s.gen
}

// openLocked opens the fortress the membership names, with mu held.
func (s *Service) openLocked() {
	m, ok, err := s.membership()
	if err != nil {
		s.cfg.Log.Error("fortress: could not read the membership", "err", err)
		return
	}
	if !ok {
		return
	}
	if m.Previous != nil && time.Since(m.Changed) >= previousKept {
		m.Previous = nil
		if err := s.saveDoc(docMembership, m); err != nil {
			s.cfg.Log.Error("fortress: could not forget the phrase before the last change", "err", err)
		}
	}
	st := s.settings()
	id, err := member.FromSeed(member.KindInstance, m.Seed)
	if err != nil {
		s.cfg.Log.Error("fortress: the member key does not open", "err", err)
		return
	}
	network := s.startLAN(st)
	cfg := s.config(m.Secret, id, st, network)
	cfg.Previous = m.Previous
	cfg.Keeper = keeper{s}
	cfg.Store = sealedStore{s, docBook}
	gen := s.generation() + 1
	cfg.OnRemoved = func() { go s.removedFrom(gen) }
	f, err := kitfortress.Open(cfg)
	if err != nil {
		if network != nil {
			_ = network.Close()
		}
		s.cfg.Log.Error("fortress: could not open the fortress", "err", err)
		return
	}
	s.live.Lock()
	s.fort, s.network, s.gen = f, network, gen
	if m.Previous != nil {
		s.forget = time.AfterFunc(time.Until(m.Changed.Add(previousKept)), s.reopen)
	}
	s.live.Unlock()
}

// config is what this instance shows a fortress of secret and does for its
// members, the same while a join is only tried.
func (s *Service) config(secret []byte, id member.Identity, st settings, network *lan.Service) kitfortress.Config {
	mode, url := st.relay()
	return kitfortress.Config{
		App:      s.cfg.App,
		Secret:   secret,
		Identity: id,
		Self:     s.cfg.Self,
		Host:     s.cfg.Host,
		Relay:    mode,
		RelayURL: url,
		LAN:      network,
		Status:   s.cfg.Status,
		Uses:     []string{kitfortress.OpenAction},
	}
}

// startLAN starts announcing this instance on the network, unless the owner
// switched that off.
func (s *Service) startLAN(st settings) *lan.Service {
	if st.NoLAN {
		return nil
	}
	network := lan.New(lan.Config{Group: kitfortress.Multicast, Ports: s.cfg.Ports})
	network.Start()
	return network
}

// closeLocked leaves the network, with mu held, and reports whether this
// instance was in a fortress.
func (s *Service) closeLocked() bool {
	s.live.Lock()
	f, network := s.fort, s.network
	s.fort, s.network = nil, nil
	if s.forget != nil {
		s.forget.Stop()
		s.forget = nil
	}
	s.gen++
	s.live.Unlock()
	if f == nil {
		return false
	}
	if err := f.Close(); err != nil {
		s.cfg.Log.Error("fortress: could not save the fortress on closing it", "err", err)
	}
	if network != nil {
		_ = network.Close()
	}
	return true
}

func (s *Service) reopen() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closeLocked()
	s.openLocked()
}

// removedFrom leaves the fortress once another member took this instance out.
func (s *Service) removedFrom(gen uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.generation() != gen {
		return
	}
	s.cfg.Log.Warn("fortress: another member took this CastleParcel out, so it left the fortress")
	s.closeLocked()
	if err := s.forgetMembership(); err != nil {
		s.cfg.Log.Error("fortress: could not forget the membership", "err", err)
	}
	st := s.settings()
	st.Kicked = true
	if err := s.saveDoc(docSettings, st); err != nil {
		s.cfg.Log.Error("fortress: could not note that this CastleParcel was taken out", "err", err)
	}
	s.note(report.FortressRemovedHere, "", "")
}

func (s *Service) forgetMembership() error {
	return errors.Join(store.DropFortressDoc(s.cfg.DB, docMembership), store.DropFortressDoc(s.cfg.DB, docBook))
}

// keeper keeps the phrase another member hands over when the fortress
// changes it, and opens the fortress again under it unless this instance
// changes the phrase itself.
type keeper struct{ s *Service }

func (k keeper) Keep(secret []byte) error {
	m, ok, err := k.s.membership()
	if err != nil {
		return err
	}
	if !ok {
		return errNoFortress
	}
	m.Previous, m.Secret, m.Changed, m.Removed = m.Secret, secret, time.Now(), nil
	if err := k.s.saveDoc(docMembership, m); err != nil {
		return err
	}
	k.s.live.Lock()
	changing := k.s.changing
	k.s.live.Unlock()
	if !changing {
		go k.s.reopen()
	}
	return nil
}

// Join joins the fortress of secret, unless its members turn this instance
// away until the phrase changes. In a fortress already, it takes the new
// phrase a member said the fortress changed to, as the same member.
func (s *Service) Join(secret []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, in, err := s.membership()
	if err != nil {
		return err
	}
	if in {
		if f := s.current(); f == nil || !f.PhraseChanged() {
			return errInFortress
		}
		s.closeLocked()
		m.Previous, m.Secret, m.Changed, m.Removed = m.Secret, secret, time.Now(), nil
		err := s.saveDoc(docMembership, m)
		s.openLocked()
		return err
	}
	id, err := member.NewIdentity(member.KindInstance)
	if err != nil {
		return err
	}
	st := s.settings()
	if !s.takenIn(secret, id, st) {
		s.cfg.Log.Warn("fortress: the members turned the join away until the phrase changes, because one of them was removed")
		return errClosed
	}
	if err := s.saveDoc(docMembership, membership{Secret: secret, Seed: id.Seed()}); err != nil {
		return err
	}
	st.Kicked = false
	if err := s.saveDoc(docSettings, st); err != nil {
		return err
	}
	s.openLocked()
	s.note(report.FortressJoined, "", "")
	return nil
}

// joinWait bounds how long a join waits for the members to take this
// instance in or turn it away, and joinSettle is how long the relay has to
// list them once it connected.
var (
	joinWait   = 5 * time.Second
	joinSettle = 1500 * time.Millisecond
)

// takenIn opens the fortress of secret under id, keeping nothing, until its
// members answer, and reports whether they take this instance in.
func (s *Service) takenIn(secret []byte, id member.Identity, st settings) bool {
	network := s.startLAN(st)
	if network != nil {
		defer func() { _ = network.Close() }()
	}
	cfg := s.config(secret, id, st, network)
	cfg.Store = &memoryStore{}
	f, err := kitfortress.Open(cfg)
	if err != nil {
		s.cfg.Log.Error("fortress: could not try the join", "err", err)
		return false
	}
	defer f.Close()
	mode, _ := st.relay()
	return !turnedAway(f, mode != group.RelayOff, network != nil)
}

// turnedAway waits until a member of f turns this instance away or takes it
// in, until the relay lists nobody to do either, or until joinWait is over.
// Members on this network may announce themselves any moment, so only a
// fortress reached through the relay alone can tell that nobody is there.
func turnedAway(f *kitfortress.Fortress, relayed, onLAN bool) bool {
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	deadline := time.After(joinWait)
	var connected time.Time
	for {
		if f.ShutOut() {
			return true
		}
		members := f.Members()
		if slices.ContainsFunc(members, func(m kitfortress.Member) bool { return m.State == group.StateApproved }) {
			return false
		}
		now := time.Now()
		if relayed && connected.IsZero() && f.RelayConnected() {
			connected = now
		}
		settled := !relayed || !connected.IsZero() && now.Sub(connected) >= joinSettle
		if !onLAN && settled && len(members) == 0 {
			return false
		}
		select {
		case <-deadline:
			return false
		case <-tick.C:
		}
	}
}

// Leave takes this instance out of the fortress. The members in reach drop it
// at once, the others when they hear it from them.
func (s *Service) Leave() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, in, err := s.membership(); err != nil || !in {
		return cmp.Or(err, errNoFortress)
	}
	if f := s.current(); f != nil {
		f.Leave()
	}
	s.closeLocked()
	if err := s.forgetMembership(); err != nil {
		return err
	}
	s.note(report.FortressLeft, "", "")
	return nil
}

// Remove takes another member out of the fortress, here and at every member
// in reach. Until the phrase changes the fortress takes no new app, so the
// member's names are kept for the page to say so.
func (s *Service) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f := s.current()
	if f == nil {
		return errNoFortress
	}
	i := slices.IndexFunc(f.Members(), func(m kitfortress.Member) bool { return m.ID == id })
	if i < 0 {
		return errNoMember
	}
	gone := f.Members()[i]
	if err := f.Remove(id); err != nil {
		return err
	}
	m, _, err := s.membership()
	if err != nil {
		return err
	}
	r := Removed{App: cmp.Or(gone.App.Name, gone.App.ID), Device: gone.Device}
	m.Removed = append(m.Removed, r)
	s.note(report.FortressRemoved, r.App, r.Device)
	return s.saveDoc(docMembership, m)
}

// Phrase returns the words of the fortress's phrase, and the text of the code
// beside them, which names an own relay after the words.
func (s *Service) Phrase() (words, code string, err error) {
	m, in, err := s.membership()
	if err != nil || !in {
		return "", "", cmp.Or(err, errNoFortress)
	}
	if words, err = phrase.Encode(m.Secret); err != nil {
		return "", "", err
	}
	_, url := s.settings().relay()
	return words, relay.PairingCode(words, url), nil
}

// ChangePhrase gives the fortress a new phrase and returns its words and the
// members out of reach, which still hold the old one.
func (s *Service) ChangePhrase(ctx context.Context) ([]string, []Removed, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f := s.current()
	if f == nil {
		return nil, nil, errNoFortress
	}
	s.live.Lock()
	s.changing = true
	s.live.Unlock()
	took, err := f.ChangePhrase(ctx)
	s.live.Lock()
	s.changing = false
	s.live.Unlock()
	if err != nil {
		return nil, nil, err
	}
	left := []Removed{}
	for _, m := range f.Members() {
		if m.Kind == member.KindInstance && m.State == group.StateApproved && !slices.Contains(took, m.ID) {
			left = append(left, Removed{App: cmp.Or(m.App.Name, m.App.ID), Device: m.Device})
		}
	}
	s.closeLocked()
	s.openLocked()
	s.note(report.FortressPhraseChanged, "", "")
	words, _, err := s.Phrase()
	return strings.Fields(words), left, err
}

// SetNetwork changes how this instance reaches the other members and opens
// the fortress again on it.
func (s *Service) SetNetwork(relayAt string, onLAN bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.settings()
	st.Relay, st.NoLAN = relayAt, !onLAN
	if err := s.saveDoc(docSettings, st); err != nil {
		return err
	}
	if s.closeLocked() {
		s.openLocked()
	}
	return nil
}

func (s *Service) membership() (membership, bool, error) {
	var m membership
	ok, err := s.loadDoc(docMembership, &m)
	if err == nil && ok && (len(m.Secret) != phrase.SecretLen || len(m.Seed) == 0) {
		err = errors.New("fortress: the membership is incomplete")
	}
	return m, ok && err == nil, err
}

// settings are the network settings, the defaults where none were saved or
// they do not open.
func (s *Service) settings() settings {
	var st settings
	if _, err := s.loadDoc(docSettings, &st); err != nil {
		s.cfg.Log.Error("fortress: could not read the settings", "err", err)
	}
	return st
}

func (s *Service) note(code, app, device string) {
	e := store.FortressEvent{At: time.Now().UnixMilli(), Code: code, App: app, Device: device}
	if err := store.AddFortressEvent(s.cfg.DB, e); err != nil {
		s.cfg.Log.Error("fortress: could not note an event in the log", "code", code, "err", err)
	}
}

func sealDomain(name string) string { return "cp/fortress/" + name + "/v1" }

func (s *Service) loadRaw(name string) ([]byte, error) {
	sealed, err := store.FortressDoc(s.cfg.DB, name)
	if err != nil || sealed == nil {
		return nil, err
	}
	return s.cfg.Seal.Open(sealDomain(name), sealed)
}

func (s *Service) saveRaw(name string, raw []byte) error {
	sealed, err := s.cfg.Seal.Seal(sealDomain(name), raw)
	if err != nil {
		return err
	}
	return store.SaveFortressDoc(s.cfg.DB, name, sealed)
}

func (s *Service) loadDoc(name string, v any) (bool, error) {
	raw, err := s.loadRaw(name)
	if err != nil || raw == nil {
		return false, err
	}
	return true, json.Unmarshal(raw, v)
}

func (s *Service) saveDoc(name string, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return s.saveRaw(name, raw)
}

// sealedStore keeps what the kit saves for the fortress, sealed.
type sealedStore struct {
	s    *Service
	name string
}

func (d sealedStore) Load() ([]byte, error) { return d.s.loadRaw(d.name) }
func (d sealedStore) Save(raw []byte) error { return d.s.saveRaw(d.name, raw) }

// memoryStore keeps what the kit saves for a fortress that is only tried.
type memoryStore struct {
	mu  sync.Mutex
	raw []byte
}

func (m *memoryStore) Load() ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.raw), nil
}

func (m *memoryStore) Save(raw []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.raw = slices.Clone(raw)
	return nil
}
