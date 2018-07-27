/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config defines configuration variables in invoker agent.
type Config struct {
	TimeOps          bool
	DockerSock       string
	ContainerDir     string
	OutputLogDir     string
	InvokerAgentPort int
	LogSinkSize      int64
}

// Default configuration variables.
const (
	defaultDockerSock       string = "/var/run/docker.sock"
	defaultContainerDir     string = "/containers"
	defaultOutputLogDir     string = "/action-logs"
	defaultInvokerAgentPort int    = 3233
	defaultLogSinkSize      int64  = 100 * 1024 * 1024
)

// NewConfigFromEnv generate config object with configuration variables from environment variables or default values.
func NewConfigFromEnv() *Config {
	var config = &Config{}

	config.getTimeOpsFromEnv()

	config.getPortFromEnv()

	config.getLogSinkSizeFromEnv()

	config.DockerSock = getEnvWithFallback("INVOKER_AGENT_DOCKER_SOCK", defaultDockerSock)

	config.ContainerDir = getEnvWithFallback("INVOKER_AGENT_CONTAINER_DIR", defaultContainerDir)

	config.OutputLogDir = getEnvWithFallback("INVOKER_AGENT_OUTPUT_LOG_DIR", defaultOutputLogDir)

	return config
}

func (c *Config) getTimeOpsFromEnv() {
	if value, ok := os.LookupEnv("INVOKER_AGENT_TIME_TRACE"); ok {
		if str := strings.ToLower(value); str == "true" {
			c.TimeOps = true
		} else {
			if str == "false" {
				c.TimeOps = false
			} else {
				fmt.Fprintf(os.Stderr, "Invalid INVOKER_AGENT_TIME_TRACE %s\n", value)
				panic(errors.New("invalid INVOKER_AGENT_TIME_TRACE variable, should be one of true or false"))
			}
		}
	} else {
		c.TimeOps = false
	}
}

func (c *Config) getPortFromEnv() {
	if value, ok := os.LookupEnv("INVOKER_AGENT_PORT"); ok {
		invokerAgentPort, err := strconv.Atoi(value)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid INVOKER_AGENT_PORT %s; error was %v\n", value, err)
			panic(err)
		}
		c.InvokerAgentPort = invokerAgentPort
	} else {
		c.InvokerAgentPort = defaultInvokerAgentPort
	}
}

func (c *Config) getLogSinkSizeFromEnv() {
	if value, ok := os.LookupEnv("INVOKER_AGENT_LOG_SINK_SIZE"); ok {
		logSinkSize, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid INVOKER_AGENT_LOG_SINK_SIZE %s; error was %v\n", value, err)
			panic(err)
		}
		c.LogSinkSize = logSinkSize
	} else {
		c.LogSinkSize = defaultLogSinkSize
	}
}

func getEnvWithFallback(envKey string, fallback string) string {
	if value, ok := os.LookupEnv(envKey); ok {
		return value
	}
	return fallback
}
