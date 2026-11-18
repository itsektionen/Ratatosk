package app

import (
	"encoding/json"
	"net/http"

	"github.com/itsektionen/ratatosk/internal/auth"
	"github.com/itsektionen/ratatosk/internal/config"
	"github.com/itsektionen/ratatosk/internal/httpx"
	"github.com/itsektionen/ratatosk/internal/sites"
)

// boot is everything the UI needs to draw its first screen. It is embedded in
// the HTML so the page needs no requests of its own to start
type boot struct {
	auth.Me
	BaseDomain string     `json:"baseDomain"`
	Sites      []siteView `json:"sites"`
}

func loadBoot(r *http.Request, cfg config.Config, a *auth.Auth, store *sites.Store) (boot, error) {
	me := a.Me(r.Context())
	var user auth.User // the zero user can't edit anything
	if me.User != nil {
		user = *me.User
	}
	views, err := siteViews(r, store, user)
	return boot{Me: me, BaseDomain: cfg.BaseDomain, Sites: views}, err
}

// bootHandler is the same data as a request, for when the HTML wasn't served by us (vite dev)
func bootHandler(cfg config.Config, a *auth.Auth, store *sites.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := loadBoot(r, cfg, a, store)
		if err != nil {
			httpx.InternalError(w, "load boot", err)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		httpx.WriteJSON(w, b)
	}
}

// bootJSON is for web.Handler. The result is safe to put inside a <script> tag
func bootJSON(cfg config.Config, a *auth.Auth, store *sites.Store) func(*http.Request) ([]byte, error) {
	return func(r *http.Request) ([]byte, error) {
		b, err := loadBoot(r, cfg, a, store)
		if err != nil {
			return nil, err
		}
		return json.Marshal(b) // escapes <, > and &
	}
}
