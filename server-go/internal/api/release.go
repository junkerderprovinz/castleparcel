package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

// releasesURL lists CastleParcel's published releases on GitHub.
const releasesURL = "https://api.github.com/repos/junkerderprovinz/castleparcel/releases"

// The page may only talk to its own server, so the server asks GitHub on its
// behalf. Anyone can reach these routes, so one answer serves every visitor
// for a while and GitHub hears from this server at most once per window and
// question.
const (
	releaseFresh  = time.Hour
	releaseFailed = 5 * time.Minute
	// More tags than releases exist only come from someone making them up.
	releaseKeys = 64
)

var releaseTag = regexp.MustCompile(`^v\d{1,4}\.\d{1,4}\.\d{1,4}$`)

// Release is what the Info page needs of one GitHub release.
type Release struct {
	Tag  string `json:"tag_name"`
	Body string `json:"body"`
	URL  string `json:"html_url"`
}

type releaseEntry struct {
	release Release
	err     error
	checked time.Time
}

// Releases remembers GitHub's answers about the newest release and about the
// notes of single tags, each with when it asked.
type Releases struct {
	URL    string
	Client *http.Client

	mu      sync.Mutex
	entries map[string]releaseEntry
}

// NewReleases asks CastleParcel's own repository.
func NewReleases() *Releases {
	return &Releases{URL: releasesURL, Client: &http.Client{Timeout: 10 * time.Second}}
}

// get answers from memory while the last answer to path is fresh. The lock is
// held during the request, so visitors who ask at once share one call.
func (r *Releases) get(path string) (Release, time.Time, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e, ok := r.entries[path]; ok {
		keep := releaseFresh
		if e.err != nil {
			keep = releaseFailed
		}
		if time.Since(e.checked) < keep {
			return e.release, e.checked, e.err
		}
	}
	if r.entries == nil || len(r.entries) >= releaseKeys {
		r.entries = map[string]releaseEntry{}
	}
	// A visitor who leaves mid-request does not cancel the answer the next one gets.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	release, err := r.fetch(ctx, path)
	if err != nil {
		log.Printf("release %s: %v", path, err)
	}
	e := releaseEntry{release: release, err: err, checked: time.Now()}
	r.entries[path] = e
	return e.release, e.checked, e.err
}

func (r *Releases) fetch(ctx context.Context, path string) (Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.URL+"/"+path, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "CastleParcel")
	resp, err := r.Client.Do(req)
	if err != nil {
		return Release{}, fmt.Errorf("asking GitHub: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("GitHub answered %s", resp.Status)
	}
	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return Release{}, fmt.Errorf("reading GitHub's answer: %w", err)
	}
	if release.Tag == "" {
		return Release{}, errors.New("GitHub named no release")
	}
	return release, nil
}

type latestReleaseResponse struct {
	Latest    string    `json:"latest"`
	CheckedAt time.Time `json:"checkedAt"`
}

// LatestReleaseHandler builds GET /api/latest-release, which the Info page's
// Check for updates asks. The page compares the tag with its own version.
func LatestReleaseHandler(r *Releases) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		release, at, err := r.get("latest")
		if err != nil {
			writeJSONError(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, latestReleaseResponse{Latest: release.Tag, CheckedAt: at})
	}
}

type releaseNotesResponse struct {
	Tag   string `json:"tag"`
	Notes string `json:"notes"`
	URL   string `json:"url"`
}

// ReleaseNotesHandler builds GET /api/release-notes/{tag}, the notes the Info
// page's What's new window shows.
func ReleaseNotesHandler(r *Releases) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		tag := chi.URLParam(req, "tag")
		if !releaseTag.MatchString(tag) {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}
		release, _, err := r.get("tags/" + tag)
		if err != nil {
			writeJSONError(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, releaseNotesResponse{Tag: release.Tag, Notes: release.Body, URL: release.URL})
	}
}
