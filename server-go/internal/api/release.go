package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// latestReleaseURL answers with CastleParcel's newest published release.
const latestReleaseURL = "https://api.github.com/repos/junkerderprovinz/castleparcel/releases/latest"

// The page may only talk to its own server, so the server asks GitHub on its
// behalf. Anyone can reach the route, so one answer serves every visitor for a
// while and GitHub hears from this server at most once per window.
const (
	releaseFresh  = time.Hour
	releaseFailed = 5 * time.Minute
)

// LatestRelease remembers the newest release tag GitHub named and when it asked.
type LatestRelease struct {
	URL    string
	Client *http.Client

	mu      sync.Mutex
	tag     string
	err     error
	checked time.Time
}

// NewLatestRelease asks CastleParcel's own repository.
func NewLatestRelease() *LatestRelease {
	return &LatestRelease{URL: latestReleaseURL, Client: &http.Client{Timeout: 10 * time.Second}}
}

// Get returns the newest tag and when GitHub was asked, from memory while the
// last answer is fresh. The lock is held during the request, so visitors who
// press at once share one call.
func (l *LatestRelease) Get() (string, time.Time, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	keep := releaseFresh
	if l.err != nil {
		keep = releaseFailed
	}
	if !l.checked.IsZero() && time.Since(l.checked) < keep {
		return l.tag, l.checked, l.err
	}
	// A visitor who leaves mid-check does not cancel the answer the next one gets.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	l.tag, l.err = l.fetch(ctx)
	l.checked = time.Now()
	if l.err != nil {
		log.Printf("latest release: %v", l.err)
	}
	return l.tag, l.checked, l.err
}

func (l *LatestRelease) fetch(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.URL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "CastleParcel")
	resp, err := l.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("asking GitHub: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub answered %s", resp.Status)
	}
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("reading GitHub's answer: %w", err)
	}
	if release.TagName == "" {
		return "", errors.New("GitHub named no release")
	}
	return release.TagName, nil
}

type latestReleaseResponse struct {
	Latest    string    `json:"latest"`
	CheckedAt time.Time `json:"checkedAt"`
}

// LatestReleaseHandler builds GET /api/latest-release, which the Info page's
// Check for updates asks. The page compares the tag with its own version.
func LatestReleaseHandler(l *LatestRelease) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		tag, at, err := l.Get()
		if err != nil {
			writeJSONError(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, latestReleaseResponse{Latest: tag, CheckedAt: at})
	}
}
