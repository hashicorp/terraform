// Package projectconfigs contains the parser and models for representing
// Terraform project configuration files.
package projectconfigs

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// ProjectConfigFilenameNative is the name of the configuration file expressing
	// project configuration in HCL native syntax.
	ProjectConfigFilenameNative = ".terraform-project.hcl"

	// ProjectConfigFilenameJSON is the name of the configuration file expressing
	// project configuration in HCL JSON.
	ProjectConfigFilenameJSON = ".terraform-project.hcl.json"
)

// FindProjectRoot looks in the given directory and all of the parent
// directories of it in turn until it finds one that is a Terraform
// project root.
func FindProjectRoot(startDir string) (string, error) {
	var err error
	startDirRaw := startDir
	startDir, err = filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("invalid start directory %q: %s", startDirRaw, err)
	}

	info, err := os.Stat(startDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("start directory %q does not exist", startDirRaw)
		}
		return "", fmt.Errorf("invalid start directory %q: %s", startDirRaw, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("invalid start directory %q: not a directory", startDirRaw)
	}

	currentDir := startDir
	for {
		nativeSyntaxFile := filepath.Join(currentDir, ProjectConfigFilenameNative)
		jsonSyntaxFile := filepath.Join(currentDir, ProjectConfigFilenameJSON)

		_, err := os.Stat(nativeSyntaxFile)
		if !os.IsNotExist(err) {
			return currentDir, nil
		}
		_, err = os.Stat(jsonSyntaxFile)
		if !os.IsNotExist(err) {
			return currentDir, nil
		}

		parentDir, _ := filepath.Split(currentDir)
		parentDir = filepath.Clean(parentDir) // trim trailing slash on non-root, and any other normalization

		// If we're not making progress then we've hit a root directory and
		// so we'll give up.
		if parentDir == currentDir {
			return "", fmt.Errorf("no parent directory of %s contains either a %s or a %s file", startDirRaw, ProjectConfigFilenameNative, ProjectConfigFilenameJSON)
		}
		currentDir = parentDir
	}
}
