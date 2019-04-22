package main

import (
	"net/http"
	"time"
	"strconv"
)

// Handler is the entry point for this fission function
func Handler(w http.ResponseWriter, r *http.Request) {
	strN := r.URL.Query().Get("n")
	if strN == "" {
		strN = "1"
	}
	n, parseErr := strconv.Atoi(strN)
	if parseErr != nil {
		n = 200
	}
	time.Sleep(time.Duration(n) * time.Millisecond)
	w.Write([]byte(string(n)))
}
