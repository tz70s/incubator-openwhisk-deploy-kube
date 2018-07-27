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
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
)

type RuncSuspendResumeOps struct {
	*Config
}

const runcCmd = "/usr/bin/docker-runc"

func CheckIfRuncExisted() bool {
	cmd := exec.Command("docker-runc")
	err := cmd.Run()
	if err != nil {
		log.Printf("runc doesn't exist.")
		return false
	}
	return true
}

func NewRuncSuspendResumeOps(cfg *Config) *RuncSuspendResumeOps {
	return &RuncSuspendResumeOps{
		Config: cfg,
	}
}

func (rops *RuncSuspendResumeOps) Resume(w http.ResponseWriter, r *http.Request) {
	var start time.Time
	if rops.TimeOps {
		start = time.Now()
	}

	vars := mux.Vars(r)
	container := vars["container"]
	cmd := exec.Command("docker-runc", "resume", container)
	err := cmd.Run()
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Unpausing %s failed with error: %v\n", container, err)
	} else {
		w.WriteHeader(204) // success!
	}

	if rops.TimeOps {
		end := time.Now()
		elapsed := end.Sub(start)
		fmt.Fprintf(os.Stdout, "Unpause took %s\n", elapsed.String())
	}
}

func (rops *RuncSuspendResumeOps) Suspend(w http.ResponseWriter, r *http.Request) {
	var start time.Time
	if rops.TimeOps {
		start = time.Now()
	}

	vars := mux.Vars(r)
	container := vars["container"]
	cmd := exec.Command("docker-runc", "pause", container)
	err := cmd.Run()
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Pausing %s failed with error: %v\n", container, err)
	} else {
		w.WriteHeader(204) // success!
	}

	if rops.TimeOps {
		end := time.Now()
		elapsed := end.Sub(start)
		fmt.Fprintf(os.Stdout, "Pause took %s\n", elapsed.String())
	}
}
