// Copyright (c) Jingyuan Zhang. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package scheduler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
//	"math"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"strconv"

	"github.com/openfaas/faas/ics/config"
	"github.com/openfaas/faas/ics/types"
	"github.com/openfaas/faas/ics/proxy"
	"github.com/openfaas/faas/ics/monitor"
//	"github.com/openfaas/faas/ics/monitor/sampler"
//	"github.com/openfaas/faas/ics/monitor/model"
)

var ErrSpecialization = errors.New("Scheduler: Failed to specialize environment")
var ErrBusying = errors.New("Scheduler: Busying")
var ErrNotAvailable = errors.New("Scheduler: No available environment")
var ErrUnregistered = errors.New("Scheduler: Unregistered environment")
var ErrStatusTransition = errors.New("SchedulerStatus: Failed to transit")
var ErrStatusPendding = errors.New("SchedulerStatus: Pendding and wait for transition")

const NameCPUAnalyser = "cpu"
const NameLatencyReporter = "latency"

type FaasEnvironment struct {
	cmd         *exec.Cmd
	address     string
	endpoint    string
	ready       bool
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
	if status == STATUS_SERVING {
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
	Config       *config.WatchdogConfig
	Proxy        *proxy.Server
	Profiler     func(string)

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
	monitor     monitor.ResourceMonitor
}

func NewScheduler(cfg *config.WatchdogConfig, profiler func(string)) (*Scheduler, error) {
	debug := len(cfg.Profile) > 0
	ics := &Scheduler {
		Config: cfg,
		Profiler: profiler,
		Proxy: proxy.NewServer(cfg.Port, false),
		monitor: monitor.NewIntervalMonitor(nil),
	}

	// cpuAnalyser := monitor.NewLinearAnalyser(
	// 	sampler.CPUUsageSamplerInstance(),
	// 	sampler.NewRequestSampler(ics.Proxy))
	// cpuAnalyser.SetDebug(debug)
	// ics.monitor.AddAnalyser(NameCPUAnalyser, cpuAnalyser)

	latencyReporter := monitor.NewLatencyReporter(ics.Proxy.ServedFeed)
	latencyReporter.SetDebug(debug)
	ics.monitor.AddAnalyser(NameLatencyReporter, latencyReporter)

	ics.monitor.Start()
	go func() {
		for {
			select {
			// case serving := <-ics.Proxy.ServingFeed:
			// 	ics.servingHandler(serving.(int32))
			case err := <-ics.monitor.Error():
				log.Printf("Error while monitor resources: %v\n", err)
			}
		}
	}()

	return ics, nil
}

func (ics *Scheduler) LaunchFEs() {
	ics.mubusy.Lock()
	ics.busying = ics.Config.Instances
	ics.mubusy.Unlock()

	ics.faasEnvs = make(map[int]*FaasEnvironment, ics.Config.Instances)
	for i := 1; i <= ics.Config.Instances; i++ {
		go ics.launch(ics.Config.Port + i)
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
	fe.ready = true
	ics.done()

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
		reterr := ics.pendLocked(ics.Serve, functionName)
		ics.mustatus.Unlock()
		return <-reterr
	}

	defer ics.mustatus.Unlock()
	if err != nil {
		return err
	}

	ics.busy(false)
	err = ics.specialize(ics.serving, functionName)
	ics.done()

	if err == nil {
		ics.ServingFunction = functionName
		ics.status = status
	}
	ics.Profiler("serve")
	return err
}

func (ics *Scheduler) Share(functionName string) error {
	err := ics.busy(true)
	if err == ErrBusying {
		return ics.pend(ics.Share, functionName)
	} else if err != nil {
		return err
	}
	defer ics.done()

	ics.mustatus.Lock()
	defer ics.mustatus.Unlock()

	status, err := ics.status.share()
	if err != nil {
		return err
	}

	available := 0
	for port, meta := range ics.faasEnvs {
		if port != ics.serving && meta.ready {
			available = port
			break
		}
	}
	if available == 0 {
		return ErrNotAvailable
	}

	err = ics.specialize(available, functionName)
	if err == nil {
		ics.Proxy.Share(available, ics.faasEnvs[available].address)
		ics.SharingFunction = functionName
		ics.sharing = available
		ics.status = status
	}
	return err
}

func (ics *Scheduler) Close() {
	ics.Proxy.Close()
}

func (ics *Scheduler) unshare(functionName string) error {
	err := ics.busy(true)
	if err == ErrBusying {
		return ics.pend(ics.unshare, functionName)
	} else if err != nil {
		return err
	}
	// Done until relaunched
	// defer ics.done()

	ics.mustatus.Lock()
	defer ics.mustatus.Unlock()

	status, err := ics.status.unshare()
	if err != nil {
		return err
	}

	// Unshare
	port := ics.sharing
	ics.Proxy.Unshare()
	ics.SharingFunction = ""
	ics.sharing = 0
	ics.status = status

	// Relaunch
	err = ics.terminate(port)
	if err != nil {
		return err
	}

	go ics.launch(port)
	return nil
}

func (ics *Scheduler) Unshare() error {
	return ics.unshare("")
}

func (ics *Scheduler) promote(functionName string) error {
	err := ics.busy(true)
	if err == ErrBusying {
		return ics.pend(ics.promote, functionName)
	} else if err != nil {
		return err
	}
	// Done until relaunched
	// defer ics.done()

	ics.mustatus.Lock()
	defer ics.mustatus.Unlock()

	status, err := ics.status.promote()
	if err != nil {
		return err
	}

	// Promote
	port := ics.serving
	ics.Proxy.Promote()
	ics.ServingFunction = ics.SharingFunction
	ics.SharingFunction = ""
	ics.serving = ics.sharing
	ics.sharing = 0
	ics.status = status

	// Relaunch
	err = ics.terminate(port)
	if err != nil {
		return err
	}

	go ics.launch(port)
	return nil
}

func (ics *Scheduler) Promote() error {
	return ics.promote("")
}

func (ics *Scheduler) Swap(functionName string) error {
	err := ics.busy(true)
	if err == ErrBusying {
		return ics.pend(ics.Swap, functionName)
	} else if err != nil {
		return err
	}
	// Done until relaunched
	// defer ics.done()

	ics.mustatus.Lock()
	defer ics.mustatus.Unlock()

	status, err := ics.status.swap()
	if err != nil {
		return err
	}

	// Find available environment
	available := 0
	for port, meta := range ics.faasEnvs {
		if port != ics.serving && meta.ready {
			available = port
			break
		}
	}
	if available == 0 {
		return ErrNotAvailable
	}

	// Swap to GC
	if len(functionName) == 0 {
		functionName = ics.ServingFunction
	}

	// Specialize
	err = ics.specialize(available, functionName)
	if err != nil {
		return err
	}

	// Swap serving
	port := ics.serving
	ics.Proxy.Swap(available, ics.faasEnvs[available].address)
	ics.ServingFunction = functionName
	ics.serving = available
	ics.status = status

	// Relaunch
	err = ics.terminate(port)
	if err != nil {
		return err
	}

	go ics.launch(port)
	return nil
}

func (ics *Scheduler) makeOnProxyHandler() func(int) {
	return func(port int) {
		ics.mustatus.Lock()
		ics.serving = port
		ics.status, _ = ics.status.ready()
		pending := ics.tbsettleLocked()
		ics.mustatus.Unlock()

		if pending != nil {
			go pending()
		}

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

func (ics *Scheduler) launch(port int) {
	// Pass i as id, and port to environments.
	faasProcess := fmt.Sprintf(ics.Config.FaasProcess, port)
	log.Printf("Launching \"%s\"...\n", faasProcess)
	parts := strings.Split(faasProcess, " ")

	execCmd := exec.Command(parts[0], parts[1:]...)
	execStderr, err := execCmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
		return
	}
	execStdout, err := execCmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
		return
	}
	err = execCmd.Start()
	if err != nil {
		log.Fatal(err)
		return
	}

	prefix := []byte(fmt.Sprintf("Environment %d: ", port))
	go ics.pipeFE(prefix, execStderr, os.Stderr)
	go ics.pipeFE(prefix, execStdout, os.Stdout)

	ics.faasEnvs[port] = &FaasEnvironment{
		cmd:     execCmd,
		address: fmt.Sprintf(":%d", port),
		endpoint: fmt.Sprintf("http://localhost:%d", port),
		ready:   false,
	}
}

func (ics *Scheduler) terminate(port int) error {
	fe, registered := ics.faasEnvs[port]
	if !registered {
		log.Printf("Error on terminate %d: unregistered.\n", port)
		return ErrUnregistered
	}

	log.Printf("Terminating \"%d\"...\n", fe.cmd.Process.Pid)
	err := fe.cmd.Process.Kill()
	if err != nil {
		log.Printf("Error on termination: %v", err)
		return err
	}

	// Wait for environment to exit, and avoid zombie.
	fe.cmd.Process.Wait()
	delete(ics.faasEnvs, port)
	return nil
}

func (ics *Scheduler) specialize(port int, functionName string) error {
	log.Printf("Specializing environment %d\n", port)
	url := ics.faasEnvs[port].endpoint + "/v2/specialize"

	loadFunction := types.FunctionLoadRequest{
		FilePath:         ics.Config.FaasBasePath,
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

func (ics *Scheduler) busy(exclusive bool) error {
	ics.mubusy.Lock()
	if exclusive && ics.busying > 0 {
		ics.mubusy.Unlock()
		return ErrBusying
	}
	ics.busying += 1
	ics.mubusy.Unlock()
	return nil
}

func (ics *Scheduler) done() {
	ics.mubusy.Lock()
	ics.busying -= 1
	ics.mubusy.Unlock()

	if ics.busying == 0 {
		ics.settle()
	}
}

func (ics *Scheduler) pend(caller func(string) error, functionName string) error {
	ics.mustatus.Lock()
	defer ics.mustatus.Unlock()

	return <-ics.pendLocked(caller, functionName)
}

func (ics *Scheduler) pendLocked(caller func(string) error, functionName string) chan error {
	reterr := make(chan error)
	ics.pending = append(ics.pending, ics.makePendingCaller(caller, functionName, reterr))
	return reterr
}

func (ics *Scheduler) settle() {
	if len(ics.pending) == 0 {
		return
	}

	var pending func()
	ics.mustatus.Lock()
	pending = ics.tbsettleLocked()
	ics.mustatus.Unlock()

	if pending != nil {
		go pending()
	}
}

func (ics *Scheduler) tbsettleLocked() func() {
	if len(ics.pending) > 0 {
		pending := ics.pending[0]
		ics.pending = ics.pending[1:]
		return pending
	} else {
		return nil
	}
}

func (ics *Scheduler) servingHandler(requested int32) {
	// log.Printf("%d", requested)
	// if requested % 10 == 0 {
	// 	_, err := ics.monitor.GetAnalyser(NameCPUAnalyser).Query(float64(requested))
	// 	if err != nil && err != monitor.ErrOverestimate {
	// 		return
	// 	}
	// //
	// // 	// log.Printf("Estimate %d:%f", serving, expected)
	// // 	if expected > 0.4 {
	// // 		// 1 - 0.4 / expected = x / (10 + x)  => x = expected / 0.4 - 1
	// // 		throttle := (expected / 0.4 - 1) * 10
	// // 		log.Printf("Estimate %f,%f will be throttled", expected, throttle)
	// // 		for i := 0; i < int(math.Ceil(throttle)); i++ {
	// // 			ics.Proxy.Throttle <- true
	// // 		}
	// // 	}
	// }
}
