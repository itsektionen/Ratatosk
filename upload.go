package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
)

func uploadHandler(cfg Config) http.HandlerFunc {
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
		if !dnsRe.MatchString(subdomain) {
			http.Error(w, "invalid subdomain", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		if err := Deploy(cfg, subdomain, file, header); err != nil {
			if ue, ok := errors.AsType[*UserError](err); ok {
				http.Error(w, ue.Msg, http.StatusUnprocessableEntity)
			} else {
				log.Printf("deploy %q failed: %v", subdomain, err)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}

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
