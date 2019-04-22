package main

import (
	"fmt"
	"net/http"
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
		n = 1
	}

	seq := 0
	for i := 0; i < n; i++ {
		seq++
	}

	msg := fmt.Sprintf("Ret %d", seq)

	w.Write([]byte(msg))
}
