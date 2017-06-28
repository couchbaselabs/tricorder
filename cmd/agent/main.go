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
	pb "../../rpc"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"sync"
)

func loadConfig(configFile string, config *Config) {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Error while reading config: %v", err)
	}
	err = yaml.Unmarshal([]byte(data), &config)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
}

func main() {
	configFile := flag.String("config", "config.yml", "Config file for the tricorder agent")
	flag.Parse()
	agent := &Agent{
		config: &Config{},
		mutex:  &sync.Mutex{},
		logger: &logger.Logger{},
	}
	loadConfig(fmt.Sprint("./", *configFile), agent.config)

	if agent.config.logging.logLevel == "" || strings.EqualFold(agent.config.logging.logLevel, "info") {
		agent.logger.Init(agent.config.logging.file, 1)
	} else if strings.EqualFold(agent.config.logging.logLevel, "error") {
		agent.logger.Init(agent.config.logging.file, 0)
	} else if strings.EqualFold(agent.config.logging.logLevel, "debug") {
		agent.logger.Init(agent.config.logging.file, 2)
	}

	agent.logger.Info("Starting the agent at %v", agent.config.Port)

	lis, err := net.Listen("tcp", fmt.Sprint(":", agent.config.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()

	agent.Initialize()
	pb.RegisterAgentServiceServer(s, agent)
	// Register reflection service on gRPC server.
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}