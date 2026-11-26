// Package report is what CastleParcel tells the fortress and the dashboards
// about itself, in the shape the kit's status contract asks for: the uploads
// in progress, a storage quota close to full, and a log of the parcels, file
// requests and fortress events it still keeps.
package report

import (
	"context"
	"database/sql"
	_ "embed"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/junkerderprovinz/parleyport/kit/status"

	"github.com/junkerderprovinz/castleparcel/server-go/internal/store"
)

// Version is the version of CastleParcel, the same as in package.json.
const Version = "2.2.2"

// logoSVG is CastleParcel's mark, drawn smaller than src/assets has it so it
// fits the size the contract allows an icon.
//
//go:embed logo.svg
var logoSVG []byte

// Icon is the mark as the members of a fortress show it.
var Icon, _ = status.IconFromSVG(logoSVG)

// nearlyFull is the share of STORAGE_QUOTA from which the owner should look.
const nearlyFull = 0.9

// Source reads the report and the log from CastleParcel's database.
type Source struct {
	DB *sql.DB
	// TmpDir holds the uploads in progress, whose size so far is their
	// progress.
	TmpDir   string
	Quota    int64
	App      status.App
	Instance func() status.Instance
	Now      func() time.Time
}

func (s Source) Report(_ context.Context, lang string) (status.Report, error) {
	tr := translator(lang)
	r := status.Report{App: s.App, Instance: s.Instance(), State: status.StateOK, SampledAt: s.Now().UTC()}
	uploads, err := store.UploadsInProgress(s.DB)
	if err != nil {
		return status.Report{}, err
	}
	for _, u := range uploads {
		key := "work.upload"
		if u.ForRequest {
			key = "work.requestUpload"
		}
		t := status.Task{
			ID:        "upload-" + u.Key,
			Kind:      status.KindTransfer,
			Title:     tr(key, map[string]string{"size": size(u.Size, lang)}),
			StartedAt: time.UnixMilli(u.CreatedAt).UTC(),
		}
		if fi, err := os.Stat(filepath.Join(s.TmpDir, u.ID)); err == nil && u.Size > 0 {
			p := min(float64(fi.Size())/float64(u.Size), 1)
			t.Progress = &p
		}
		r.Work = append(r.Work, t)
	}
	if s.Quota > 0 {
		used, err := store.QuotaUsage(s.DB, "")
		if err != nil {
			return status.Report{}, err
		}
		if share := float64(used) / float64(s.Quota); share > nearlyFull {
			severity := status.SeverityWarn
			if used >= s.Quota {
				severity = status.SeverityError
			}
			r.NeedsYou = append(r.NeedsYou, status.Need{
				ID: "storage", Severity: severity, Code: "storage.nearly_full",
				Text: tr("need.storage", map[string]string{"percent": strconv.Itoa(int(share * 100))}),
			})
		}
	}
	return r, nil
}

func (s Source) Log(_ context.Context, q status.LogQuery) (status.LogPage, error) {
	tr := translator(q.Lang)
	at, key := cursor(q.After)
	rows, err := store.LogAfter(s.DB, at, key, q.Limit)
	if err != nil {
		return status.LogPage{}, err
	}
	page := status.LogPage{Entries: []status.LogEntry{}, Next: q.After}
	for _, row := range rows {
		e := status.LogEntry{ID: row.Key, At: time.UnixMilli(row.At).UTC(), Level: status.SeverityInfo}
		values := map[string]string{"size": size(row.Size, q.Lang), "app": row.App, "device": row.Device}
		if row.Kind == store.LogFortress {
			e.Code = "fortress." + row.Code
			if row.Code == FortressRemovedHere {
				e.Level = status.SeverityWarn
			}
		} else {
			e.Code = row.Kind
		}
		e.Text = tr("log."+strings.TrimPrefix(e.Code, "fortress."), values)
		page.Entries = append(page.Entries, e)
		page.Next = strconv.FormatInt(row.At, 10) + "." + row.Key
	}
	return page, nil
}

// The codes of the fortress events in the log.
const (
	FortressJoined        = "joined"
	FortressLeft          = "left"
	FortressRemoved       = "removed"
	FortressRemovedHere   = "removed_here"
	FortressPhraseChanged = "phrase_changed"
)

// cursor reads a cursor Log handed out. One it never handed out starts at the
// oldest line.
func cursor(after string) (int64, string) {
	at, key, ok := strings.Cut(after, ".")
	n, err := strconv.ParseInt(at, 10, 64)
	if !ok || err != nil {
		return 0, ""
	}
	return n, key
}

// size writes n bytes the way the language writes decimals.
func size(n int64, lang string) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	v, u := float64(n), 0
	for v >= 1000 && u < len(units)-1 {
		v /= 1000
		u++
	}
	if u == 0 {
		return strconv.FormatInt(n, 10) + " " + units[0]
	}
	text := strconv.FormatFloat(v, 'f', 1, 64)
	if strings.Contains(decimalComma, " "+base(lang)+" ") {
		text = strings.Replace(text, ".", ",", 1)
	}
	return text + " " + units[u]
}

// decimalComma lists the languages that write a decimal comma.
const decimalComma = " bg ca cs da de el es et eu fi fr gl hr hu id is it lt lv nl no pl pt ro ru sk sl sr sv tr uk vi "

// base is the language a tag such as "pt-BR" or "nb" names.
func base(lang string) string {
	b := strings.ToLower(lang)
	if i := strings.IndexAny(b, "-_"); i >= 0 {
		b = b[:i]
	}
	if b == "nb" || b == "nn" {
		return "no"
	}
	return b
}

// translator returns the texts of lang, English for a language CastleParcel
// does not speak and for a text its table lacks.
func translator(lang string) func(key string, values map[string]string) string {
	table := texts[base(lang)]
	return func(key string, values map[string]string) string {
		text, ok := table[key]
		if !ok {
			text = texts["en"][key]
		}
		for name, v := range values {
			text = strings.ReplaceAll(text, "{"+name+"}", v)
		}
		return text
	}
}
