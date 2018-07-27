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
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// LogForwardInfo JSON structure expected as request body on /logs route
type LogForwardInfo struct {
	LastOffset             int64  `json:"lastOffset"`             // last offset read from this container's log
	SizeLimit              int    `json:"sizeLimit"`              // size limit on logs read in bytes
	SentinelledLogs        bool   `json:"sentinelledLogs"`        // does an action's log end with sentinel lines?
	EncodedLogLineMetadata string `json:"encodedLogLineMetadata"` // string to be injected in every log line
	EncodedActivation      string `json:"encodedActivation"`      // extra line to injected after all log lines are read
}

// String constants related to logging
const (
	logSentinelLine        = "XXX_THE_END_OF_A_WHISK_ACTIVATION_XXX"
	truncatedLogMessage    = "Logs were truncated because the total bytes size exceeds the limit of %d bytes."
	genericLogErrorMessage = "There was an issue while collecting your logs. Data might be missing."
)

type LogForwardHandler struct {
	*Config
	logSinkChannel chan string
}

func NewLogForwardHandler(config *Config) *LogForwardHandler {
	l := &LogForwardHandler{
		Config:         config,
		logSinkChannel: make(chan string),
	}
	go l.logWriter()
	return l
}

// ForwardLogsFromUserAction is a request handler for /logs/<container> route
// The container was given as part of the URL; gorilla makes it available in vars["container"]
// The JSON body of the request is expected to contain the fields specified by the
// LogForwardInfo struct defined above.
// If logs are successfully forwarded, the ending offset of the log file is returned
// to be used in a subsequent call to the /logs/<container> route.
func (l *LogForwardHandler) ForwardLogsFromUserAction(w http.ResponseWriter, r *http.Request) {
	var start time.Time
	if l.TimeOps {
		start = time.Now()
	}

	vars := mux.Vars(r)
	container := vars["container"]

	lfi, err := parseLogForwardInfo(r)
	if err != nil {
		// Return 400 status code if a parsing error occurred.
		l.reportLoggingError(w, 400, err.Error(), "")
		return
	}

	logFileOffset, err := l.logScan(container, lfi)
	if err != nil {
		l.reportLoggingError(w, 500, err.Error(), lfi.EncodedLogLineMetadata)
		l.logSinkChannel <- lfi.EncodedActivation // Write activation record before returning with error code.
		return
	}

	// Success; return updated logFileOffset to invoker
	w.WriteHeader(200)
	fmt.Fprintf(w, "%d", logFileOffset)

	if l.TimeOps {
		end := time.Now()
		elapsed := end.Sub(start)
		fmt.Fprintf(os.Stdout, "LogForward took %s\n", elapsed.String())
	}
}

func parseLogForwardInfo(r *http.Request) (*LogForwardInfo, error) {
	var lfi *LogForwardInfo
	b, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		errString := fmt.Sprintf("Error reading request body: %v", err)
		return nil, errors.New(errString)
	}
	err = json.Unmarshal(b, lfi)
	if err != nil {
		errString := fmt.Sprintf("Error unmarshalling request body: %v", err)
		return nil, errors.New(errString)
	}
	return lfi, nil
}

func (l *LogForwardHandler) logScan(container string, lfi *LogForwardInfo) (int64, error) {

	logFileName := l.ContainerDir + "/" + container + "/" + container + "-json.log"
	logFile, err := os.Open(logFileName)
	defer logFile.Close()
	if err != nil {
		errString := fmt.Sprintf("Error opening %s: %v", logFileName, err)
		return 0, errors.New(errString)
	}

	offset, err := logFile.Seek(lfi.LastOffset, 0)
	if offset != lfi.LastOffset || err != nil {
		errString := fmt.Sprintf("Unable to seek to %d in log file", lfi.LastOffset)
		return 0, errors.New(errString)
	}

	sentinelsLeft := 2
	scanner := bufio.NewScanner(logFile)
	bytesWritten := 0
	for sentinelsLeft > 0 && scanner.Scan() {
		logLine := scanner.Text()
		if lfi.SentinelledLogs && strings.Contains(logLine, logSentinelLine) {
			sentinelsLeft--
		} else {
			logLineLen := len(logLine)
			bytesWritten += logLineLen
			mungedLine := fmt.Sprintf("%s,%s}", logLine[:logLineLen-1], lfi.EncodedLogLineMetadata)
			l.logSinkChannel <- mungedLine
			if bytesWritten > lfi.SizeLimit {
				l.writeSyntheticLogLine(fmt.Sprintf(truncatedLogMessage, lfi.SizeLimit), lfi.EncodedLogLineMetadata)
				logFile.Seek(0, 2) // Seek to end of logfile to skip rest of output and prepare for next action invoke
				sentinelsLeft = 0  // Cause loop to exit now.
			}
		}
	}

	if lfi.SentinelledLogs && sentinelsLeft != 0 {
		errString := fmt.Sprintf("Failed to find expected sentinels in log file")
		return 0, errors.New(errString)
	}

	// Done copying log; write the activation record.
	l.logSinkChannel <- lfi.EncodedActivation

	// seek 0 bytes from current position to set logFileOffset to current fpos
	logFileOffset, err := logFile.Seek(0, 1)
	if err != nil {
		errString := fmt.Sprintf("Unable to determine current offset in log file: %v", err)
		return 0, errors.New(errString)
	}

	return logFileOffset, nil
}

// go routine that accepts log lines from the logSinkChannel and writes them to the logSink
func (l *LogForwardHandler) logWriter() {
	var sinkFile *os.File
	var sinkFileBytes int64
	var err error

	for {
		line := <-l.logSinkChannel

		if sinkFile == nil {
			timestamp := time.Now().UnixNano() / 1000000
			fname := fmt.Sprintf("%s/userlogs-%d.log", l.OutputLogDir, timestamp)
			sinkFile, err = os.Create(fname)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Unable to create log sink: %v\n", err)
				panic(err)
			}
			sinkFileBytes = 0
		}

		bytesWritten, err := fmt.Fprintln(sinkFile, line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to log sink: %v\n", err)
			sinkFile.Close()
			panic(err)
		}

		sinkFileBytes += int64(bytesWritten)
		if sinkFileBytes > l.LogSinkSize {
			sinkFile.Close()
			sinkFile = nil
		}
	}
}

func (l *LogForwardHandler) writeSyntheticLogLine(msg string, metadata string) {
	now := time.Now().UTC().Format(time.RFC3339)
	line := fmt.Sprintf("{\"log\":\"%s\", \"stream\":\"stderr\", \"time\":\"%s\",%s}", msg, now, metadata)
	l.logSinkChannel <- line
}

func (l *LogForwardHandler) reportLoggingError(w http.ResponseWriter, code int, msg string, metadata string) {
	w.WriteHeader(code)
	fmt.Fprint(w, msg)
	fmt.Fprintln(os.Stderr, msg)
	if metadata != "" {
		l.writeSyntheticLogLine(genericLogErrorMessage, metadata)
	}
}
