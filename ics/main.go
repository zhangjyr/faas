// Copyright (c) Alex Ellis 2017. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package main provides the OpenFaaS Classic Watchdog. The Classic Watchdog is a HTTP
// shim for serverless functions providing health-checking, graceful shutdowns,
// timeouts and a consistent logging experience.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/openfaas/faas/ics/config"
	"github.com/openfaas/faas/ics/types"
	"github.com/openfaas/faas/ics/scheduler"
)

var (
	versionFlag          bool
	acceptingConnections int32
)

func main() {
	flag.BoolVar(&versionFlag, "version", false, "Print the version and exit")

	flag.Parse()
	printVersion()

	if versionFlag {
		return
	}

	atomic.StoreInt32(&acceptingConnections, 0)

	osEnv := types.OsEnv{}
	readConfig := config.ReadConfig{}
	cfg := readConfig.Read(osEnv)

	if len(cfg.FaasProcess) == 0 {
		log.Panicln("Provide a valid process via fprocess environmental variable.")
		return
	}

	readTimeout := cfg.ReadTimeout
	writeTimeout := cfg.WriteTimeout

	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.AdminPort),
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		MaxHeaderBytes: 1 << 20, // Max header of 1MB
	}

	// debug := len(cfg.Profile) > 0
	ics, err := scheduler.NewScheduler(cfg, getProfiler(cfg))
	if err != nil {
		log.Panicln(err)
	}
	defer ics.Close()

	log.Printf("Read/write timeout: %s, %s. Port: %d\n", readTimeout, writeTimeout, cfg.AdminPort)
	http.HandleFunc("/_/health", makeHealthHandler())
	http.HandleFunc("/_/ready/", makeReadyHandler(ics))
	http.HandleFunc("/_/serve/", makeServeHandler(ics))
	http.HandleFunc("/_/share/", makeShareHandler(ics))
	http.HandleFunc("/_/unshare/", makeUnshareHandler(ics))
	http.HandleFunc("/_/swap/", makeSwapHandler(ics))
	http.HandleFunc("/_/promote/", makePromoteHandler(ics))

	shutdownTimeout := cfg.WriteTimeout
	idleConnsClosed := listenUntilShutdown(shutdownTimeout, s, cfg)

	ics.LaunchFEs()

	if len(cfg.Faas) > 0 {
		go ics.Serve(cfg.Faas)
	}

	<-idleConnsClosed
}

func markUnhealthy() error {
	atomic.StoreInt32(&acceptingConnections, 0)

	path := filepath.Join(os.TempDir(), ".lock")
	log.Printf("Removing lock-file : %s\n", path)
	removeErr := os.Remove(path)
	return removeErr
}

// listenUntilShutdown will listen for HTTP requests until SIGTERM
// is sent at which point the code will wait `shutdownTimeout` before
// closing off connections and a futher `shutdownTimeout` before
// exiting
func listenUntilShutdown(shutdownTimeout time.Duration, s *http.Server, cfg *config.WatchdogConfig) chan struct{} {

	idleConnsClosed := make(chan struct{})
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGTERM)

		<-sig

		log.Printf("SIGTERM received.. shutting down server in %s\n", shutdownTimeout.String())

		healthErr := markUnhealthy()

		if healthErr != nil {
			log.Printf("Unable to mark unhealthy during shutdown: %s\n", healthErr.Error())
		}

		<-time.Tick(shutdownTimeout)

		if err := s.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Printf("Error in Shutdown: %v", err)
		}

		log.Printf("No new connections allowed. Exiting in: %s\n", shutdownTimeout.String())

		<-time.Tick(shutdownTimeout)

		close(idleConnsClosed)
	}()

	// Run the HTTP server in a separate go-routine.
	go func() {
		// Add by Tianium
		profiler("admin", cfg)

		if err := s.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("Error ListenAndServe: %v", err)
			close(idleConnsClosed)
		}
	}()

	if cfg.SuppressLock == false {
		path, writeErr := createLockFile()

		if writeErr != nil {
			log.Panicf("Cannot write %s. To disable lock-file set env suppress_lock=true.\n Error: %s.\n", path, writeErr.Error())
		}
	} else {
		log.Println("Warning: \"suppress_lock\" is enabled. No automated health-checks will be in place for your function.")

		atomic.StoreInt32(&acceptingConnections, 1)
	}

	return idleConnsClosed
}

func printVersion() {
	sha := "unknown"
	if len(GitCommit) > 0 {
		sha = GitCommit
	}

	log.Printf("Version: %v\tSHA: %v\n", BuildVersion(), sha)
}

// Add by Tianium
func profiler(action string, cfg *config.WatchdogConfig) {
	if len(cfg.Profile) == 0 {
		return
	}

	file, openErr := os.OpenFile(cfg.Profile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0660)
	if openErr != nil {
		log.Printf("Warning: failed to open profile. Error: %s.\n", openErr.Error())
	}

	defer file.Close()

	now := time.Now()
	_, writeErr := file.WriteString(fmt.Sprintf("%s,%s,%d.%d\n", "scheduler", action, now.Unix(), now.Nanosecond()))
	if writeErr != nil {
		log.Printf("Warning: failed to profile action \"%s\". Error: %s.\n", action, writeErr)
	}
}

func getProfiler(cfg *config.WatchdogConfig) func(string) {
	return func(action string) {
		profiler(action, cfg)
	}
}
