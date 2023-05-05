// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build !windows
// +build !windows

package cliconfig

import (
	"errors"
	"os"
	"os/user"
	"path/filepath"
)

func configFile() (string, error) {
	dir, err := homeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, ".terraformrc"), nil
}

func configDir() (string, error) {
	dir, err := homeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, ".terraform.d"), nil
}

func homeDir() (string, error) {
	// First prefer the HOME environmental variable
	if home := os.Getenv("HOME"); home != "" {
		// FIXME: homeDir gets called from globalPluginDirs during init, before
		// the logging is set up.  We should move meta initializtion outside of
		// init, but in the meantime we just need to silence this output.
		//log.Printf("[DEBUG] Detected home directory from env var: %s", home)

		return home, nil
	}

	// If that fails, try build-in module
	user, err := user.Current()
	if err != nil {
		return "", err
	}

	if user.HomeDir == "" {
		return "", errors.New("blank output")
	}

	return user.HomeDir, nil
}
