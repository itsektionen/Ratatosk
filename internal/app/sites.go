package app

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/itsektionen/ratatosk/internal/auth"
	"github.com/itsektionen/ratatosk/internal/sites"
)

type siteView struct {
	Name      string  `json:"name"`
	URL       string  `json:"url"`
	Owner     *string `json:"owner"`
	OwnerRole string  `json:"ownerRole"` // empty if unowned
	CanEdit   bool    `json:"canEdit"`
}

// siteViews lists every site as user sees it
func siteViews(r *http.Request, store *sites.Store, user auth.User) ([]siteView, error) {
	list, err := store.List(r.Context())
	if err != nil {
		return nil, err
	}
	views := make([]siteView, 0, len(list))
	for _, s := range list {
		v := siteView{Name: s.Name, URL: siteURL(r, s.Name), CanEdit: user.CanEdit(s.OwnedBy(user.ID))}
		if s.Owned() {
			v.Owner = &s.OwnerName
			v.OwnerRole = s.OwnerRole
		}
		views = append(views, v)
	}
	return views, nil
}

func listSites(store *sites.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, _ := auth.FromContext(r.Context())
		views, err := siteViews(r, store, user)
		if err != nil {
			slog.Error("list sites", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(views)
	}
}

func deleteSite(store *sites.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if !sites.ValidName(name) {
			http.NotFound(w, r)
			return
		}
		site, err := store.Get(r.Context(), name)
		if errors.Is(err, sites.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			slog.Error("get site", "site", name, "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		user, _ := auth.FromContext(r.Context())
		if !user.CanEdit(site.OwnedBy(user.ID)) {
			http.Error(w, "you can't modify this site", http.StatusForbidden)
			return
		}
		if err := store.Delete(r.Context(), name); err != nil {
			slog.Error("delete site", "site", name, "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
