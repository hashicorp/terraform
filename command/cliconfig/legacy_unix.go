// +build !windows

package cliconfig

import (
	"errors"
	"os"
	"os/user"
	"path/filepath"
)

func legacyConfigFile() (string, error) {
	dir, err := homeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, ".terraformrc"), nil
}

func legacyConfigDir() (string, error) {
	dir, err := homeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, ".terraform.d"), nil
}

func unixHomeDir() (string, error) {
	// First prefer the HOME environmental variable
	if home := os.Getenv("HOME"); home != "" {
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
