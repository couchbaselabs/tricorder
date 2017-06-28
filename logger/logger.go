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

package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

type Logger struct {
	logFp *os.File
	level int
}

const (
	ERRORLEVEL = 0
	INFOLEVEL  = 1
	DEBUGLEVEL = 2
)
const (
	ERROR = " [Error] "
	INFO  = " [Info] "
	DEBUG = " [Debug] "
)

func (l *Logger) Init(file string, level int) {
	var fp *os.File
	if file != "" {
		var err error
		fp, err = os.Create(file)
		if err != nil {
			log.Fatalf("Unable to open log file %v", err)
		}
	} else {
		fp = os.Stdout
	}
	l.logFp = fp
	l.level = level

}

func (l *Logger) Close() {
	l.logFp.Close()
}

func (l *Logger) printf(prefix string, format string, v ...interface{}) {
	msg := fmt.Sprintf(time.Now().UTC().Format(time.RFC3339Nano)+prefix+format+"\n", v...)
	l.logFp.WriteString(msg)
}

func (l *Logger) Info(format string, v ...interface{}) {
	if l.level >= INFOLEVEL {
		l.printf(INFO, format, v...)
	}
}

func (l *Logger) Debug(format string, v ...interface{}) {
	if l.level >= DEBUGLEVEL {
		l.printf(DEBUG, format, v...)
	}
}

func (l *Logger) Error(format string, v ...interface{}) {
	if l.level >= ERRORLEVEL {
		l.printf(ERROR, format, v...)
	}
}

func (l *Logger) Output(message string) error {
	l.printf("", message)
	return nil
}
