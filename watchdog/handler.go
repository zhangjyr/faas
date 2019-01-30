// Copyright (c) Alex Ellis 2017. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
)

func lockFilePresent() bool {
	path := filepath.Join(os.TempDir(), ".lock")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func createLockFile() (string, error) {
	path := filepath.Join(os.TempDir(), ".lock")
	log.Printf("Writing lock-file to: %s\n", path)
	writeErr := ioutil.WriteFile(path, []byte{}, 0660)

	atomic.StoreInt32(&acceptingConnections, 1)

	return path, writeErr
}

func makeHealthHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if atomic.LoadInt32(&acceptingConnections) == 0 || lockFilePresent() == false {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))

			break
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func makeReadyHandler(ics *Scheduler) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodGet:

			port := r.Header.Get("X-FE-PORT")
			if len(port) == 0 {
				port = strings.TrimPrefix(r.URL.Path, "/_/ready/")
			}
			if len(port) == 0 {
				w.WriteHeader(http.StatusBadRequest)
			} else if ics.RegisterFE(port) <= 0 {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
			}

			break
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func makeServeHandler(ics *Scheduler) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			function := r.Header.Get("X-FUNCTION")
			if len(function) == 0 {
				function = strings.TrimPrefix(r.URL.Path, "/_/serve/")
			}
			if len(function) == 0 {
				w.WriteHeader(http.StatusBadRequest)
			}

			err := ics.Serve(function)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
			}
			break
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func makeShareHandler(ics *Scheduler) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			function := r.Header.Get("X-FUNCTION")
			if len(function) == 0 {
				function = strings.TrimPrefix(r.URL.Path, "/_/share/")
			}
			if len(function) == 0 {
				w.WriteHeader(http.StatusBadRequest)
			}

			err := ics.Share(function)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
			}
			break
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func makeUnshareHandler(ics *Scheduler) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			err := ics.Unshare()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
			}
			break
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func makePromoteHandler(ics *Scheduler) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			err := ics.Promote()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
			}
			break
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func makeSwapHandler(ics *Scheduler) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			function := r.Header.Get("X-FUNCTION")
			if len(function) == 0 {
				function = strings.TrimPrefix(r.URL.Path, "/_/swap/")
			}

			err := ics.Swap(function)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
			}
			break
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}
