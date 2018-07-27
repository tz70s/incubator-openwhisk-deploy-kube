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
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func handleRequests(config *Config, logForwardHandler *LogForwardHandler, suspendResumeOps SuspendResumeOps) {
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("/logs/{container}", logForwardHandler.ForwardLogsFromUserAction)
	myRouter.HandleFunc("/suspend/{container}", suspendResumeOps.Suspend)
	myRouter.HandleFunc("/resume/{container}", suspendResumeOps.Resume)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.InvokerAgentPort), myRouter))
}

func main() {
	config := NewConfigFromEnv()

	logForwardHandler := NewLogForwardHandler(config)
	var suspendResumeOps SuspendResumeOps
	if CheckIfRuncExisted() {
		log.Println("Runc existed, use runc for optimization ...")
		suspendResumeOps = NewRuncSuspendResumeOps(config)
	} else {
		log.Println("Runc doesn't existed, use docker command instead ..")
		suspendResumeOps = NewDockerSuspendResumeOps(config)
	}
	handleRequests(config, logForwardHandler, suspendResumeOps)
}
