package server

import (
	"net"
	"net/http"
	"strings"
)

// Returns the request host, lowercased and without port
func Hostname(r *http.Request) string {
	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	// Fun fact: as far as DNS cares, example.com == example.com.
	return strings.ToLower(strings.TrimSuffix(host, "."))
}
