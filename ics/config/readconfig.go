// Copyright (c) Alex Ellis 2017. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package config

import (
	"strconv"
	"time"
)

// HasEnv provides interface for os.Getenv
type HasEnv interface {
	Getenv(key string) string
}

// ReadConfig constitutes config from env variables
type ReadConfig struct {
}

func isBoolValueSet(val string) bool {
	return len(val) > 0
}

func parseBoolValue(val string) bool {
	if val == "true" {
		return true
	}
	return false
}

func parseIntOrDurationValue(val string, fallback time.Duration) time.Duration {
	if len(val) > 0 {
		parsedVal, parseErr := strconv.Atoi(val)
		if parseErr == nil && parsedVal >= 0 {
			return time.Duration(parsedVal) * time.Second
		}
	}

	duration, durationErr := time.ParseDuration(val)
	if durationErr != nil {
		return fallback
	}
	return duration
}

func parseIntValue(val string, fallback int) int {
	if len(val) > 0 {
		parsedVal, parseErr := strconv.Atoi(val)
		if parseErr == nil && parsedVal >= 0 {
			return parsedVal
		}
	}

	return fallback
}

// Read fetches config from environmental variables.
func (ReadConfig) Read(hasEnv HasEnv) *WatchdogConfig {
	cfg := &WatchdogConfig{
		WriteDebug:    false,
		CgiHeaders:    true,
		CombineOutput: true,
		FaasBasePath:  ".",
	}

	cfg.FaasProcess = hasEnv.Getenv("fprocess")
	cfg.Instances = parseIntValue(hasEnv.Getenv("instances"), 2)

	cfg.ReadTimeout = parseIntOrDurationValue(hasEnv.Getenv("read_timeout"), time.Second*5)
	cfg.WriteTimeout = parseIntOrDurationValue(hasEnv.Getenv("write_timeout"), time.Second*5)

	cfg.ExecTimeout = parseIntOrDurationValue(hasEnv.Getenv("exec_timeout"), time.Second*0)
	cfg.Port = parseIntValue(hasEnv.Getenv("port"), 8080)
	cfg.AdminPort = parseIntValue(hasEnv.Getenv("admin_port"), 8079)

	writeDebugEnv := hasEnv.Getenv("write_debug")
	if isBoolValueSet(writeDebugEnv) {
		cfg.WriteDebug = parseBoolValue(writeDebugEnv)
	}

	cgiHeadersEnv := hasEnv.Getenv("cgi_headers")
	if isBoolValueSet(cgiHeadersEnv) {
		cfg.CgiHeaders = parseBoolValue(cgiHeadersEnv)
	}

	cfg.MarshalRequest = parseBoolValue(hasEnv.Getenv("marshal_request"))
	cfg.DebugHeaders = parseBoolValue(hasEnv.Getenv("debug_headers"))

	cfg.SuppressLock = parseBoolValue(hasEnv.Getenv("suppress_lock"))

	cfg.ContentType = hasEnv.Getenv("content_type")

	if isBoolValueSet(hasEnv.Getenv("combine_output")) {
		cfg.CombineOutput = parseBoolValue(hasEnv.Getenv("combine_output"))
	}

	// Add by Tianium
	cfg.Profile = hasEnv.Getenv("profile")

	cfg.FaasBasePath = hasEnv.Getenv("faasBasePath")

	cfg.Faas = hasEnv.Getenv("faas")

	return cfg
}

// WatchdogConfig for the process.
type WatchdogConfig struct {

	// HTTP read timeout
	ReadTimeout time.Duration

	// HTTP write timeout
	WriteTimeout time.Duration

	// faasProcess is the process to exec
	FaasProcess string

	// duration until the faasProcess will be killed
	ExecTimeout time.Duration

	// writeDebug write console stdout statements to the container
	WriteDebug bool

	// marshal header and body via JSON
	MarshalRequest bool

	// cgiHeaders will make environmental variables available with all the HTTP headers.
	CgiHeaders bool

	// prints out all incoming and out-going HTTP headers
	DebugHeaders bool

	// Don't write a lock file to /tmp/
	SuppressLock bool

	// contentType forces a specific pre-defined value for all responses
	ContentType string

	// port for HTTP server
	Port int

	// port for management
	AdminPort int

	// combineOutput combines stderr and stdout in response
	CombineOutput bool

	// Add by Tianium: path of profile
	Profile string

	// faas instances
	Instances int

	// base path for faas module
	FaasBasePath string

	// start faas
	Faas string
}
