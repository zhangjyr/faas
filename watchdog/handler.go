// Copyright (c) Alex Ellis 2017. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"strconv"

	"github.com/openfaas/faas/watchdog/proxy"
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

// Add by Tianium
type FaasEnvironment struct {
	cmd         *exec.Cmd
	address     string
}

type Scheduler struct{
	Config       *WatchdogConfig
	Proxy        *proxy.Server
	Profiler     ProfilerFunc

	faasEnvs    map[int]*FaasEnvironment
	mu          sync.Mutex
	serving     int
}

func (ics *Scheduler) LaunchFEs() {
	ics.faasEnvs = make(map[int]*FaasEnvironment, ics.Config.instances)
	for i := 0; i < ics.Config.instances; i++ {
		port := ics.Config.port + i + 1
		// Pass i as id, and port to environments.
		faasProcess := fmt.Sprintf(ics.Config.faasProcess, port)
		log.Printf("Launching \"%s\"\n", faasProcess)
		parts := strings.Split(faasProcess, " ")

		execCmd := exec.Command(parts[0], parts[1:]...)
		err := execCmd.Start()
		if err != nil {
			log.Fatal(err)
			continue
		}

		ics.faasEnvs[port] = &FaasEnvironment{
			cmd:     execCmd,
			address: fmt.Sprintf(":%d", port),
		}
	}
}

func (ics *Scheduler) RegisterFE(strPort string) int {
	port, parseErr := strconv.Atoi(strPort)
	if parseErr != nil {
		return -1
	}

	fe, registered := ics.faasEnvs[port]
	if !registered {
		return -1
	}

	log.Printf("Environment is ready: (Pid %d, Address \"%s\").\n", fe.cmd.Process.Pid, fe.address)

	if !ics.Proxy.IsListening() {
		go func() {
			err := ics.Proxy.ListenAndProxy(fe.address)
			if err != nil {
				log.Printf("Error on proxy %d: %v", port, err)
			} else {
				ics.serving = port
				ics.Profiler("proxy")
			}
		}()
	}

	return fe.cmd.Process.Pid
}
