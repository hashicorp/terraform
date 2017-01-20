// Copyright (C) 2015 Scaleway. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE.md file.

// Package utils contains logquiet
package utils

import (
	"fmt"
	"os"
)

// LogQuietStruct is a struct to store information about quiet state
type LogQuietStruct struct {
	quiet bool
}

var instanceQuiet LogQuietStruct

// Quiet enable or disable quiet
func Quiet(option bool) {
	instanceQuiet.quiet = option
}

// LogQuiet Displays info if quiet is activated
func LogQuiet(str string) {
	if !instanceQuiet.quiet {
		fmt.Fprintf(os.Stderr, "%s", str)
	}
}
