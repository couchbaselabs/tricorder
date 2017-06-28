/*
* Copyright (c) 2017 Couchbase, Inc.
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*    http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package main

import (
	"../../logger"
	"flag"
	"fmt"
	"github.com/codahale/hdrhistogram"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"strings"
)

func loadConfig(configFile string, config *Config) {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Error while reading config: %v", err)
	}
	err = yaml.Unmarshal([]byte(data), config)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
}

func main() {
	configFile := flag.String("config", "./config.yml", "Config file for the tricorder coordinator")
	flag.Parse()
	coordinator := &Coordinator{
		config:     &Config{},
		agentsInfo: make(map[string]*AgentInfo),
		histogram:  hdrhistogram.New(1, 5*1000*1000, 3), //max histogram value for latency 5 secs
		logger:     &logger.Logger{},
	}
	loadConfig(fmt.Sprint("./", *configFile), coordinator.config)
	if coordinator.config.logging.logLevel == "" || strings.EqualFold(coordinator.config.logging.logLevel, "info") {
		coordinator.logger.Init(coordinator.config.logging.file, 1)
	} else if strings.EqualFold(coordinator.config.logging.logLevel, "error") {
		coordinator.logger.Init(coordinator.config.logging.file, 0)
	} else if strings.EqualFold(coordinator.config.logging.logLevel, "debug") {
		coordinator.logger.Init(coordinator.config.logging.file, 2)
	}

	coordinator.Run()
}
