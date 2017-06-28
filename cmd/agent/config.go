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

type Config struct {
	Port            int             `yaml:"port"`
	InterfaceConfig InterfaceConfig `yaml:"interface"`
	logging         LoggingConfig   `yaml:"log"`
}

type InterfaceConfig struct {
	Device                 string `yaml:"device"`
	CaptureType            string `yaml:"type"`
	AfPacketTragetSizeInMB int    `yaml:"targetsize"`
	Port                   int    `yaml:"port"`
}

const (
	AF_PACKET = "afpacket"
	PF_RING   = "pfring"
)

type LoggingConfig struct {
	logLevel string `yaml:"level"`
	file     string `yaml:"file"`
}