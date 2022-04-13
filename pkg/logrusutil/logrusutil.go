/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package logrusutil implements some helpers for using logrus
package logrusutil

import (
	"os"
	"strings"

	"github.com/jenkins-x/lighthouse/pkg/logrusutil/stackdriver"
	"github.com/sirupsen/logrus"
)

// DefaultFieldsFormatter wraps another logrus.Formatter, injecting
// DefaultFields into each Format() call, existing fields are preserved
// if they have the same key
type DefaultFieldsFormatter struct {
	WrappedFormatter logrus.Formatter
	DefaultFields    logrus.Fields
	PrintLineNumber  bool
}

// Init set Logrus formatter
// if DefaultFieldsFormatter.wrappedFormatter is nil &logrus.JSONFormatter{} will be used instead
func Init(formatter *DefaultFieldsFormatter) {
	if formatter == nil {
		return
	}
	if formatter.WrappedFormatter == nil {
		formatter.WrappedFormatter = CreateDefaultFormatter()
	}
	logrus.SetFormatter(formatter)
	logrus.SetReportCaller(formatter.PrintLineNumber)
}

// CreateDefaultFormatter creates a default JSON formatter
func CreateDefaultFormatter() logrus.Formatter {
	if os.Getenv("LOGRUS_FORMAT") == "text" {
		return &logrus.TextFormatter{
			ForceColors:      true,
			DisableTimestamp: true,
		}
	}

	if os.Getenv("LOGRUS_FORMAT") == "stackdriver" {
		f := &stackdriver.Formatter{
			Service: os.Getenv("LOGRUS_SERVICE"),
			Version: os.Getenv("LOGRUS_SERVICE_VERSION"),
		}
		ss := os.Getenv("LOGRUS_STACK_SPLIT")
		if ss != "" {
			f.StackSkip = strings.Split(ss, ",")
		}
		return f
	}

	jsonFormat := &logrus.JSONFormatter{}
	if os.Getenv("LOGRUS_JSON_PRETTY") == "true" {
		jsonFormat.PrettyPrint = true
	}
	return jsonFormat
}

// ComponentInit is a syntax sugar for easier Init
func ComponentInit(component string) {
	Init(
		&DefaultFieldsFormatter{
			PrintLineNumber: true,
			DefaultFields:   logrus.Fields{"component": component},
		},
	)
}

// Format implements logrus.Formatter's Format. We allocate a new Fields
// map in order to not modify the caller's Entry, as that is not a thread
// safe operation.
func (f *DefaultFieldsFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	data := make(logrus.Fields, len(entry.Data)+len(f.DefaultFields))
	for k, v := range f.DefaultFields {
		data[k] = v
	}
	for k, v := range entry.Data {
		data[k] = v
	}
	return f.WrappedFormatter.Format(&logrus.Entry{
		Logger:  entry.Logger,
		Data:    data,
		Time:    entry.Time,
		Level:   entry.Level,
		Message: entry.Message,
		Caller:  entry.Caller,
	})
}
