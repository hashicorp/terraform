package command

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	DefaultEnvDir  = "terraform.tfstate.d"
	DefaultEnvFile = "environment"
	DefaultEnvName = "default"
)

// CurrentEnv returns the name of the current environment.
// If there are no configured environments, or the listed environment no longer
// exists, CurrentEnv returns "default"
func CurrentEnv() (string, error) {
	contents, err := ioutil.ReadFile(filepath.Join(DefaultDataDir, DefaultEnvFile))
	if os.IsNotExist(err) {
		return DefaultEnvName, nil
	}
	if err != nil {
		return "", err
	}

	current := DefaultEnvName
	envFromFile := strings.TrimSpace(string(contents))

	envs, err := ListEnvs()
	if err != nil {
		return "", err
	}

	// ignore the env file value if it doesn't exist
	for _, env := range envs {
		if envFromFile == env {
			current = env
			break
		}
	}

	return current, nil
}

// EnvStatePath returns the path to the current environment's state file.
// If there are no configured environments, the "default" environment is used
// and the DefaultStateFileName is returned.
func EnvStatePath() (string, error) {
	currentEnv, err := CurrentEnv()
	if err != nil {
		return "", err
	}

	if currentEnv == DefaultEnvName {
		return DefaultStateFilename, nil
	}

	return filepath.Join(DefaultEnvDir, currentEnv, DefaultStateFilename), nil
}

// ListEnvs returns a list of all known environments, always starting with
// "default", and the rest lexically sorted.
func ListEnvs() ([]string, error) {
	entries, err := ioutil.ReadDir(DefaultEnvDir)
	// no error if there's no envs configured
	if os.IsNotExist(err) {
		return []string{DefaultEnvName}, nil
	}
	if err != nil {
		return nil, err
	}

	var envs []string
	for _, entry := range entries {
		if entry.IsDir() {
			envs = append(envs, filepath.Base(entry.Name()))
		}
	}

	sort.Strings(envs)

	// always start with "default"
	envs = append([]string{DefaultEnvName}, envs...)

	return envs, nil
}
