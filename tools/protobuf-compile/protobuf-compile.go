// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// validatePath validates and sanitizes file paths
func validatePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}
	
	// Clean the path to prevent path traversal
	cleanPath := filepath.Clean(path)
	
	// Check for suspicious patterns
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("path traversal detected: %s", path)
	}
	
	// Validate that path contains only expected characters
	validPath := regexp.MustCompile(`^[a-zA-Z0-9._/-]+$`)
	if !validPath.MatchString(cleanPath) {
		return "", fmt.Errorf("invalid characters in path: %s", path)
	}
	
	return cleanPath, nil
}

// validateProtocArgs validates protoc command arguments
func validateProtocArgs(args []string) error {
	validArg := regexp.MustCompile(`^[a-zA-Z0-9._/=-]+$`)
	
	for _, arg := range args {
		if arg == "" {
			continue
		}
		
		// Allow common protoc flags
		if strings.HasPrefix(arg, "--") {
			continue
		}
		
		if !validArg.MatchString(arg) {
			return fmt.Errorf("invalid argument: %s", arg)
		}
	}
	
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <protoc-args>\n", os.Args[0])
		os.Exit(1)
	}

	// Validate all arguments
	args := os.Args[1:]
	if err := validateProtocArgs(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Validate and clean any file paths in arguments
	var cleanArgs []string
	for _, arg := range args {
		if strings.HasSuffix(arg, ".proto") || strings.Contains(arg, "/") {
			cleanArg, err := validatePath(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error validating path %s: %v\n", arg, err)
				os.Exit(1)
			}
			cleanArgs = append(cleanArgs, cleanArg)
		} else {
			cleanArgs = append(cleanArgs, arg)
		}
	}

	// Use absolute path for protoc to avoid PATH manipulation
	protocPath, err := exec.LookPath("protoc")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding protoc: %v\n", err)
		os.Exit(1)
	}

	// Execute protoc with validated arguments
	cmd := exec.Command(protocPath, cleanArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running protoc: %v\n", err)
		os.Exit(1)
	}
}
