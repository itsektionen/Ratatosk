package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/itsektionen/ratatosk/internal/auth"
	"github.com/itsektionen/ratatosk/internal/config"
	"github.com/itsektionen/ratatosk/internal/sites"
)

func uploadHandler(cfg config.Config, store *sites.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Prevent stupid uploads
		r.Body = http.MaxBytesReader(w, r.Body, cfg.MaxUpload)
		if err := r.ParseMultipartForm(cfg.MaxUpload); err != nil {
			if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
				http.Error(w, "upload too large", http.StatusRequestEntityTooLarge)
			} else {
				http.Error(w, "bad form", http.StatusBadRequest)
			}
			return
		}

		subdomain := strings.ToLower(strings.TrimSpace(r.FormValue("subdomain")))
		if !sites.ValidName(subdomain) {
			http.Error(w, "invalid subdomain", http.StatusBadRequest)
			return
		}

		user, _ := auth.FromContext(r.Context())
		site, created, err := store.Claim(r.Context(), subdomain, user.ID)
		if err != nil {
			slog.Error("claim site", "site", subdomain, "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !created && !user.CanEdit(site.OwnedBy(user.ID)) {
			http.Error(w, "you can't modify this site", http.StatusForbidden)
			return
		}

		// Don't leave a claim behind if the site never got deployed
		deployed := false
		defer func() {
			if created && !deployed {
				store.Release(context.WithoutCancel(r.Context()), subdomain)
			}
		}()

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		if err := sites.Deploy(cfg, subdomain, file, header); err != nil {
			if ue, ok := errors.AsType[*sites.UserError](err); ok {
				http.Error(w, ue.Msg, http.StatusUnprocessableEntity)
			} else {
				slog.Error("deploy failed", "site", subdomain, "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}

		deployed = true
		fmt.Fprintln(w, siteURL(r, subdomain))
	}
}

func siteURL(r *http.Request, subdomain string) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + subdomain + "." + r.Host + "/"
}
