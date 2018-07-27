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
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type DockerSuspendResumeOps struct {
	*Config
	client *http.Client
}

func NewDockerSuspendResumeOps(cfg *Config) *DockerSuspendResumeOps {

	client := NewDockerSockHttpClient(cfg)

	return &DockerSuspendResumeOps{
		Config: cfg,
		client: client,
	}
}

func NewDockerSockHttpClient(cfg *Config) *http.Client {
	// Open http client to DockerSock
	fd := func(proto, addr string) (conn net.Conn, err error) {
		return net.Dial("unix", cfg.DockerSock)
	}

	tr := &http.Transport{
		Dial: fd,
	}

	client := &http.Client{Transport: tr}
	return client
}

func (dOps *DockerSuspendResumeOps) Resume(w http.ResponseWriter, r *http.Request) {
	var start time.Time
	if dOps.TimeOps {
		start = time.Now()
	}

	vars := mux.Vars(r)
	container := vars["container"]
	dummy := strings.NewReader("")
	resp, err := dOps.client.Post("http://localhost/containers/"+container+"/unpause", "text/plain", dummy)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Unpausing %s failed with error: %v\n", container, err)
	} else if resp.StatusCode == 409 {
		w.WriteHeader(204)
		fmt.Fprintf(w, "%s is already unpaused. \n", container)
	} else if resp.StatusCode < 200 || resp.StatusCode > 299 {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Unpausing %s failed with status code: %d\n", container, resp.StatusCode)
	} else {
		w.WriteHeader(204) // success!
	}

	if dOps.TimeOps {
		end := time.Now()
		elapsed := end.Sub(start)
		fmt.Fprintf(os.Stdout, "Unpause took %s\n", elapsed.String())
	}
}

func (dOps *DockerSuspendResumeOps) Suspend(w http.ResponseWriter, r *http.Request) {
	var start time.Time
	if dOps.TimeOps {
		start = time.Now()
	}

	vars := mux.Vars(r)
	container := vars["container"]
	dummy := strings.NewReader("")
	resp, err := dOps.client.Post("http://localhost/containers/"+container+"/pause", "text/plain", dummy)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Pausing %s failed with error: %v\n", container, err)
	} else if resp.StatusCode == 409 {
		w.WriteHeader(204)
		fmt.Fprintf(w, "%s is already unpaused. \n", container)
	} else if resp.StatusCode < 200 || resp.StatusCode > 299 {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Pausing %s failed with status code: %d\n", container, resp.StatusCode)
	} else {
		w.WriteHeader(204) // success!
	}

	if dOps.TimeOps {
		end := time.Now()
		elapsed := end.Sub(start)
		fmt.Fprintf(os.Stdout, "Pause took %s\n", elapsed.String())
	}
}
