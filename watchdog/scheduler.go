// Copyright (c) Jingyuan Zhang. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"strconv"

	"github.com/openfaas/faas/watchdog/types"
	"github.com/openfaas/faas/watchdog/proxy"
)

var ErrSpecialization = errors.New("Scheduler: Failed to specialize environment")
var ErrStatusTransition = errors.New("SchedulerStatus: Failed to transit")
var ErrStatusPendding = errors.New("SchedulerStatus: Pendding and wait for transition")

type FaasEnvironment struct {
	cmd         *exec.Cmd
	address     string
	endpoint    string
}

type SchedulerStatus int

const (
   STATUS_LAUNCHING SchedulerStatus = 0
   STATUS_READY     SchedulerStatus = 1
   STATUS_SERVING   SchedulerStatus = 2
   STATUS_SHARING   SchedulerStatus = 3
)

func (status SchedulerStatus) ready() (SchedulerStatus, error) {
	if status == STATUS_LAUNCHING {
		return STATUS_READY, nil
	} else {
		return status, ErrStatusTransition
	}
}

func (status SchedulerStatus) serve() (SchedulerStatus, error) {
	if status == STATUS_READY {
		return STATUS_SERVING, nil
	} else if status == STATUS_LAUNCHING {
		return status, ErrStatusPendding
	} else {
		return status, ErrStatusTransition
	}
}

func (status SchedulerStatus) swap() (SchedulerStatus, error) {
	if status == STATUS_SERVING {
		return STATUS_SERVING, nil
	} else {
		return status, ErrStatusTransition
	}
}

func (status SchedulerStatus) share() (SchedulerStatus, error) {
	if status == STATUS_SERVING || status == STATUS_SHARING {
		return STATUS_SHARING, nil
	} else {
		return status, ErrStatusTransition
	}
}

func (status SchedulerStatus) unshare() (SchedulerStatus, error) {
	if status == STATUS_SHARING {
		return STATUS_SERVING, nil
	} else {
		return status, ErrStatusTransition
	}
}

func (status SchedulerStatus) promote() (SchedulerStatus, error) {
	if status == STATUS_SHARING {
		return STATUS_SERVING, nil
	} else {
		return status, ErrStatusTransition
	}
}

type Scheduler struct{
	Config       *WatchdogConfig
	Proxy        *proxy.Server
	Profiler     ProfilerFunc

	ServingFunction string
	SharingFunction string

	faasEnvs    map[int]*FaasEnvironment
	mustatus    sync.Mutex
	mubusy      sync.Mutex
	status      SchedulerStatus
	busying     int
	serving     int
	sharing     int
	pending     []func()
}

func (ics *Scheduler) LaunchFEs() {
	ics.mubusy.Lock()
	ics.busying = ics.Config.instances
	ics.mubusy.Unlock()

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
	ics.mubusy.Lock()
	ics.busying -= 1
	ics.mubusy.Unlock()

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

func (ics *Scheduler) Serve(functionName string) error {
	ics.mustatus.Lock()

	status, err := ics.status.serve()
	if err == ErrStatusPendding {
		reterr := make(chan error)
		ics.pending = append(ics.pending, ics.makePendingCaller(ics.Serve, functionName, reterr))
		ics.mustatus.Unlock()
		return <-reterr
	}

	defer ics.mustatus.Unlock()
	if err != nil {
		return err
	}

	ics.mubusy.Lock()
	ics.busying += 1
	ics.mubusy.Unlock()
	err = ics.specialize(ics.serving, functionName)
	ics.mubusy.Lock()
	ics.busying -= 1
	ics.mubusy.Unlock()

	if err == nil {
		ics.ServingFunction = functionName
		ics.status = status
	}
	ics.Profiler("serve")
	return err
}

func (ics *Scheduler) makeOnProxyHandler() func(int) {
	return func(port int) {
		ics.mustatus.Lock()
		ics.serving = port
		ics.status, _ = ics.status.ready()
		ics.mustatus.Unlock()

		go func() {
			for len(ics.pending) > 0 {
				pending := ics.pending[0]
				ics.pending = ics.pending[1:]

				pending()
			}
		}()

		ics.Profiler("proxy")
		log.Printf("Proxing %d\n", port)
	}
}

func (ics *Scheduler) makePendingCaller(caller func(string) error, functionName string, err chan error ) func() {
	return func() {
		err <-caller(functionName)
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
