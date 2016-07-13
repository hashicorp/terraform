// Copyright (C) 2015 Scaleway. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE.md file.

package api

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

// Logger handles logging concerns for the Scaleway API SDK
type Logger interface {
	LogHTTP(*http.Request)
	Fatalf(format string, v ...interface{})
	Debugf(format string, v ...interface{})
	Infof(format string, v ...interface{})
	Warnf(format string, v ...interface{})
}

// NewDefaultLogger returns a logger which is configured for stdout
func NewDefaultLogger() Logger {
	return &defaultLogger{
		Logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

type defaultLogger struct {
	*log.Logger
}

func (l *defaultLogger) LogHTTP(r *http.Request) {
	l.Printf("%s %s\n", r.Method, r.URL.Path)
}
func (l *defaultLogger) Fatalf(format string, v ...interface{}) {
	l.Printf("[FATAL] %s\n", fmt.Sprintf(format, v))
	os.Exit(1)
}
func (l *defaultLogger) Debugf(format string, v ...interface{}) {
	l.Printf("[DEBUG] %s\n", fmt.Sprintf(format, v))
}
func (l *defaultLogger) Infof(format string, v ...interface{}) {
	l.Printf("[INFO ] %s\n", fmt.Sprintf(format, v))
}
func (l *defaultLogger) Warnf(format string, v ...interface{}) {
	l.Printf("[WARN ] %s\n", fmt.Sprintf(format, v))
}
