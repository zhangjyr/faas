// Copyright (c) Alex Ellis 2017. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

	"github.com/openfaas/faas/watchdog/types"
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

func makeSpecializeHandler(ics *Scheduler) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			function := r.Header.Get("X-FUNCTION")
			if len(function) == 0 {
				function = strings.TrimPrefix(r.URL.Path, "/_/specialize/")
			}
			if len(function) == 0 {
				w.WriteHeader(http.StatusBadRequest)
			}

			err := ics.specialize(ics.serving, function)
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

// Add by Tianium
var ErrSpecialization = errors.New("Scheduler: Failed to specialize environment")

type FaasEnvironment struct {
	cmd         *exec.Cmd
	address     string
	endpoint    string
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
		execStderr, err := execCmd.StderrPipe()
		if err != nil {
			log.Fatal(err)
			continue
		}
		execStdout, err := execCmd.StdoutPipe()
		if err != nil {
			log.Fatal(err)
			continue
		}
		err = execCmd.Start()
		if err != nil {
			log.Fatal(err)
			continue
		}

		prefix := []byte(fmt.Sprintf("Environment %d: ", port))
		go ics.pipeFE(prefix, execStderr, os.Stderr)
		go ics.pipeFE(prefix, execStdout, os.Stdout)

		ics.faasEnvs[port] = &FaasEnvironment{
			cmd:     execCmd,
			address: fmt.Sprintf(":%d", port),
			endpoint: fmt.Sprintf("http://localhost:%d", port),
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
		log.Printf("Error on proxy %d: unregistered.\n", port)
		return -1
	}

	log.Printf("Environment is ready: (Pid %d, Address \"%s\").\n", fe.cmd.Process.Pid, fe.address)

	if !ics.Proxy.IsListening() {
		go func() {
			err := ics.Proxy.ListenAndProxy(port, fe.address, ics.makeOnProxyHandler())
			if err != nil && err != proxy.ErrServerListening {
				log.Printf("Error on proxy %d: %v\n", port, err)
			}
		}()
	}

	return fe.cmd.Process.Pid
}

func (ics *Scheduler) makeOnProxyHandler() func(int) {
	return func(port int) {
		ics.serving = port
		ics.Profiler("proxy")
		log.Printf("Proxing %d\n", port)
	}
}

func (ics *Scheduler) pipeFE(prefix []byte, src io.ReadCloser, dst io.Writer) {
	//directional copy (64k buffer)
	buff := make([]byte, 0xffff)
	for {
		n, readErr := src.Read(buff)
		if readErr != nil {
			if readErr != io.EOF {
				log.Fatal(readErr)
				src.Close()
				return
			} else if n == 0 {
				src.Close()
				return
			}

			// Pass down to to transfer rest bytes
		}
		b := buff[:n]

		//write out result
		n, writeErr := dst.Write(append(prefix, b...))
		if writeErr != nil {
			log.Fatal(writeErr)
		}

		// EOF and we're done
		if readErr != nil {
			src.Close()
			return
		}
	}
}

func (ics *Scheduler) specialize(port int, functionName string) error {
	log.Printf("Specializing environment %d\n", port)
	url := ics.faasEnvs[port].endpoint + "/v2/specialize"

	loadFunction := types.FunctionLoadRequest{
		FilePath:         ics.Config.faasBasePath,
		FunctionName:     functionName,
	}
	body, err := json.Marshal(loadFunction)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err == nil && resp.StatusCode < 300 {
		// Success
		resp.Body.Close()
		return nil
	}

	if err != nil {
		log.Printf("Failed to specialize environment %d: %v\n", port, err)
	} else {
		log.Printf("Failed to specialize environment %d: %d\n", port, resp.StatusCode)
		err = ErrSpecialization
	}
	return err
}
