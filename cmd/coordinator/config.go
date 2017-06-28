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
	Capture  CaptureConfig  `yaml:"capture"`
	Agents   []string       `yaml:"agents"`
	Port     int            `yaml:"port"`
	History  ResultsHistory `yaml:"history"`
	RestPort int            `yaml:"restport"`
	logging  LoggingConfig  `yaml:"log"`
}

type CaptureConfig struct {
	Timeout  int `yaml:"timeout"`
	Period   int `yaml:"period"`
	Interval int `yaml:"interval"`
}

type ResultsHistory struct {
	FileName string `yaml:"file"`
	Period   int    `yaml:"period"`
}

type LoggingConfig struct {
	logLevel string `yaml:"level"`
	file     string `yaml:"file"`
}
