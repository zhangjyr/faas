// Copyright (c) Alex Ellis 2017. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package config

import (
	"testing"
	"time"
)

type EnvBucket struct {
	Items map[string]string
}

func NewEnvBucket() EnvBucket {
	return EnvBucket{
		Items: make(map[string]string),
	}
}

func (e EnvBucket) Getenv(key string) string {
	return e.Items[key]
}

func (e EnvBucket) Setenv(key string, value string) {
	e.Items[key] = value
}

func TestRead_CombineOutput_DefaultTrue(t *testing.T) {
	defaults := NewEnvBucket()
	readConfig := ReadConfig{}

	config := readConfig.Read(defaults)
	want := true
	if config.CombineOutput != want {
		t.Logf("combineOutput error, want: %v, got: %v", want, config.CombineOutput)
		t.Fail()
	}
}

func TestRead_CombineOutput_OverrideFalse(t *testing.T) {
	defaults := NewEnvBucket()
	readConfig := ReadConfig{}
	defaults.Setenv("combine_output", "false")

	config := readConfig.Read(defaults)
	want := false
	if config.CombineOutput != want {
		t.Logf("combineOutput error, want: %v, got: %v", want, config.CombineOutput)
		t.Fail()
	}
}

func TestRead_CgiHeaders_OverrideFalse(t *testing.T) {
	defaults := NewEnvBucket()
	readConfig := ReadConfig{}
	defaults.Setenv("cgi_headers", "false")

	config := readConfig.Read(defaults)

	if config.CgiHeaders != false {
		t.Logf("cgiHeaders should have been false (via env)")
		t.Fail()
	}
}

func TestRead_CgiHeaders_DefaultIsTrueConfig(t *testing.T) {
	defaults := NewEnvBucket()
	readConfig := ReadConfig{}

	config := readConfig.Read(defaults)

	if config.CgiHeaders != true {
		t.Logf("cgiHeaders should have been true (unspecified)")
		t.Fail()
	}
}

func TestRead_WriteDebug_DefaultIsFalseConfig(t *testing.T) {
	defaults := NewEnvBucket()
	readConfig := ReadConfig{}

	config := readConfig.Read(defaults)

	if config.WriteDebug != false {
		t.Logf("writeDebug should have been false (unspecified)")
		t.Fail()
	}
}

func TestRead_WriteDebug_TrueOverrideConfig(t *testing.T) {
	defaults := NewEnvBucket()
	readConfig := ReadConfig{}
	defaults.Setenv("write_debug", "true")

	config := readConfig.Read(defaults)

	if config.WriteDebug != true {
		t.Logf("writeDebug should have been true (specified)")
		t.Fail()
	}
}

func TestRead_WriteDebug_FlaseConfig(t *testing.T) {
	defaults := NewEnvBucket()
	readConfig := ReadConfig{}
	defaults.Setenv("write_debug", "false")

	config := readConfig.Read(defaults)

	if config.WriteDebug != false {
		t.Logf("writeDebug should have been false (specified)")
		t.Fail()
	}
}

func TestRead_SuppressLockConfig(t *testing.T) {
	defaults := NewEnvBucket()
	readConfig := ReadConfig{}
	defaults.Setenv("suppress_lock", "true")

	config := readConfig.Read(defaults)

	if config.SuppressLock != true {
		t.Logf("suppress_lock envVariable incorrect, got: %s.\n", config.FaasProcess)
		t.Fail()
	}
}

func TestRead_ContentTypeConfig(t *testing.T) {
	defaults := NewEnvBucket()
	readConfig := ReadConfig{}
	defaults.Setenv("content_type", "application/json")

	config := readConfig.Read(defaults)

	if config.ContentType != "application/json" {
		t.Logf("content_type envVariable incorrect, got: %s.\n", config.ContentType)
		t.Fail()
	}
}

func TestRead_FprocessConfig(t *testing.T) {
	defaults := NewEnvBucket()
	readConfig := ReadConfig{}
	defaults.Setenv("fprocess", "cat")

	config := readConfig.Read(defaults)

	if config.FaasProcess != "cat" {
		t.Logf("fprocess envVariable incorrect, got: %s.\n", config.FaasProcess)
		t.Fail()
	}
}

func TestRead_EmptyTimeoutConfig(t *testing.T) {
	defaults := NewEnvBucket()
	readConfig := ReadConfig{}

	config := readConfig.Read(defaults)

	if (config.ReadTimeout) != time.Duration(5)*time.Second {
		t.Log("readTimeout incorrect")
		t.Fail()
	}
	if (config.WriteTimeout) != time.Duration(5)*time.Second {
		t.Log("writeTimeout incorrect")
		t.Fail()
	}
}

func TestRead_DefaultPortConfig(t *testing.T) {
	defaults := NewEnvBucket()

	readConfig := ReadConfig{}
	config := readConfig.Read(defaults)
	want := 8080
	if config.Port != want {
		t.Logf("port got: %d, want: %d\n", config.Port, want)
		t.Fail()
	}
}

func TestRead_PortConfig(t *testing.T) {
	defaults := NewEnvBucket()
	defaults.Setenv("port", "8081")

	readConfig := ReadConfig{}
	config := readConfig.Read(defaults)
	want := 8081
	if config.Port != want {
		t.Logf("port got: %d, want: %d\n", config.Port, want)
		t.Fail()
	}
}

func TestRead_ReadAndWriteTimeoutConfig(t *testing.T) {
	defaults := NewEnvBucket()
	defaults.Setenv("read_timeout", "10")
	defaults.Setenv("write_timeout", "60")

	readConfig := ReadConfig{}
	config := readConfig.Read(defaults)

	if (config.ReadTimeout) != time.Duration(10)*time.Second {
		t.Logf("readTimeout incorrect, got: %d\n", config.ReadTimeout)
		t.Fail()
	}
	if (config.WriteTimeout) != time.Duration(60)*time.Second {
		t.Logf("writeTimeout incorrect, got: %d\n", config.WriteTimeout)
		t.Fail()
	}
}

func TestRead_ReadAndWriteTimeoutDurationConfig(t *testing.T) {
	defaults := NewEnvBucket()
	defaults.Setenv("read_timeout", "20s")
	defaults.Setenv("write_timeout", "1m30s")

	readConfig := ReadConfig{}
	config := readConfig.Read(defaults)

	if (config.ReadTimeout) != time.Duration(20)*time.Second {
		t.Logf("readTimeout incorrect, got: %d\n", config.ReadTimeout)
		t.Fail()
	}
	if (config.WriteTimeout) != time.Duration(90)*time.Second {
		t.Logf("writeTimeout incorrect, got: %d\n", config.WriteTimeout)
		t.Fail()
	}
}

func TestRead_ExecTimeoutConfig(t *testing.T) {
	defaults := NewEnvBucket()
	defaults.Setenv("exec_timeout", "3s")

	readConfig := ReadConfig{}
	config := readConfig.Read(defaults)

	want := time.Duration(3) * time.Second
	if (config.ExecTimeout) != want {
		t.Logf("execTimeout incorrect, got: %d - want: %s\n", config.ExecTimeout, want)
		t.Fail()
	}
}

// Add by Tianium
func TestRead_ProfileConfig(t *testing.T) {
	defaults := NewEnvBucket()
	readConfig := ReadConfig{}

	config := readConfig.Read(defaults)
	if config.Profile != "" {
		t.Logf("default profile envVariable incorrect, got: \"%s\".\n", config.Profile)
		t.Fail()
	}

	defaults.Setenv("profile", "/profile.log")
	config = readConfig.Read(defaults)

	if config.Profile != "/profile.log" {
		t.Logf("profile envVariable incorrect, got: \"%s\".\n", config.Profile)
		t.Fail()
	}
}
