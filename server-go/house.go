package main

import (
	"cmp"
	"database/sql"
	"encoding/hex"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	kitfortress "github.com/junkerderprovinz/parleyport/kit/fortress"
	"github.com/junkerderprovinz/parleyport/kit/group"
	"github.com/junkerderprovinz/parleyport/kit/signin"
	"github.com/junkerderprovinz/parleyport/kit/status"

	"github.com/junkerderprovinz/castleparcel/server-go/internal/account"
	"github.com/junkerderprovinz/castleparcel/server-go/internal/config"
	"github.com/junkerderprovinz/castleparcel/server-go/internal/fortress"
	"github.com/junkerderprovinz/castleparcel/server-go/internal/report"
)

// newHouse is CastleParcel's fortress with the status it reports, closed
// until Start.
func newHouse(cfg config.Config, db *sql.DB, auth *signin.Auth, log *slog.Logger) (*fortress.Service, error) {
	seal, err := account.Seal(cfg, db)
	if err != nil {
		return nil, err
	}
	name := cfg.Branding().AppName
	// Unraid names its server to every container, and the apps of a fortress
	// on one server find each other by it.
	host := strings.TrimSpace(os.Getenv("HOST_HOSTNAME"))
	device := cmp.Or(host, name)
	instance := status.Instance{ID: hex.EncodeToString(seal.MAC("cp/fortress/instance/v1", nil)[:16]), Name: device}
	src := report.Source{
		DB: db, TmpDir: cfg.TmpDir, Quota: cfg.StorageQuota,
		App:      status.App{ID: "castleparcel", Name: name, Version: report.Version, UI: cfg.BaseURL, Icon: report.Icon},
		Instance: func() status.Instance { return instance },
		Now:      time.Now,
	}
	port, _ := strconv.Atoi(cfg.Port)
	return fortress.New(fortress.Config{
		DB: db, Seal: seal, Auth: auth,
		App:    kitfortress.App{ID: "castleparcel", Name: name, Version: report.Version, Icon: report.Icon},
		Self:   group.Self{Name: device, Deployment: "container", Address: cfg.BaseURL},
		Host:   host,
		Status: src,
		Ports:  []int{cmp.Or(port, 3000)},
		Log:    log,
	}), nil
}

// houseRoutes serves the fortress to the owner, the status to the owner, the
// API tokens and the dashboards, the calls of the other members and the tabs
// in which the owner opens their pages.
func houseRoutes(r chi.Router, cfg config.Config, auth *signin.Auth, house *fortress.Service) {
	routes := account.OwnPages(cfg, account.Admins(auth, cfg, house.Routes()))
	r.Handle("/api/fortress", routes)
	r.Handle("/api/fortress/*", routes)
	reports := status.Handler(house.Status(), account.StatusReader(cfg))
	r.Handle(status.StatusPath, reports)
	r.Handle(status.LogPath, reports)
	r.Handle(group.DirectPath, house.Direct())
	r.Handle(kitfortress.OpenPath+"*", house.Tabs())
}
