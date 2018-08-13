package handlers

import (
	"net/http"
	"strings"
)

// MakeURIPrefixStripper creates a HandlerFunc which will trim a given prefix
// from a URL before sending to the next handler.
func MakeURIPrefixStripper(next http.HandlerFunc, prefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL != nil && strings.HasPrefix(r.URL.Path, prefix) {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
		}
		next(w, r)
	}
}
