package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/hcl"
)

// ErrNoConfigsFound is the error returned by LoadDir if no
// Terraform configuration files were found in the given directory.
type ErrNoConfigsFound struct {
	Dir string
}

func (e ErrNoConfigsFound) Error() string {
	return fmt.Sprintf(
		"No Terraform configuration files found in directory: %s",
		e.Dir)
}

// LoadJSON loads a single Terraform configuration from a given JSON document.
//
// The document must be a complete Terraform configuration. This function will
// NOT try to load any additional modules so only the given document is loaded.
func LoadJSON(raw json.RawMessage) (*Config, error) {
	obj, err := hcl.Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf(
			"Error parsing JSON document as HCL: %s", err)
	}

	// Start building the result
	hclConfig := &hclConfigurable{
		Root: obj,
	}

	return hclConfig.Config()
}

// LoadFile loads the Terraform configuration from a given file.
//
// This file can be any format that Terraform recognizes, and import any
// other format that Terraform recognizes.
func LoadFile(path string) (*Config, error) {
	importTree, err := loadTree(path)
	if err != nil {
		return nil, err
	}

	configTree, err := importTree.ConfigTree()

	// Close the importTree now so that we can clear resources as quickly
	// as possible.
	importTree.Close()

	if err != nil {
		return nil, err
	}

	return configTree.Flatten()
}

// LoadDir loads all the Terraform configuration files in a single
// directory and appends them together.
//
// Special files known as "override files" can also be present, which
// are merged into the loaded configuration. That is, the non-override
// files are loaded first to create the configuration. Then, the overrides
// are merged into the configuration to create the final configuration.
//
// Files are loaded in lexical order.
func LoadDir(root string) (*Config, error) {
	files, overrides, err := dirFiles(root)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, &ErrNoConfigsFound{Dir: root}
	}

	// Determine the absolute path to the directory.
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	var result *Config

	// Sort the files and overrides so we have a deterministic order
	sort.Strings(files)
	sort.Strings(overrides)

	// Load all the regular files, append them to each other.
	for _, f := range files {
		c, err := LoadFile(f)
		if err != nil {
			return nil, err
		}

		if result != nil {
			result, err = Append(result, c)
			if err != nil {
				return nil, err
			}
		} else {
			result = c
		}
	}

	// Load all the overrides, and merge them into the config
	for _, f := range overrides {
		c, err := LoadFile(f)
		if err != nil {
			return nil, err
		}

		result, err = Merge(result, c)
		if err != nil {
			return nil, err
		}
	}

	// Mark the directory
	result.Dir = rootAbs

	return result, nil
}

// IsEmptyDir returns true if the directory given has no Terraform
// configuration files.
func IsEmptyDir(root string) (bool, error) {
	if _, err := os.Stat(root); err != nil && os.IsNotExist(err) {
		return true, nil
	}

	fs, os, err := dirFiles(root)
	if err != nil {
		return false, err
	}

	return len(fs) == 0 && len(os) == 0, nil
}

// Ext returns the Terraform configuration extension of the given
// path, or a blank string if it is an invalid function.
func ext(path string) string {
	if strings.HasSuffix(path, ".tf") {
		return ".tf"
	} else if strings.HasSuffix(path, ".tf.json") {
		return ".tf.json"
	} else {
		return ""
	}
}

func dirFiles(dir string) ([]string, []string, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, nil, err
	}
	if !fi.IsDir() {
		return nil, nil, fmt.Errorf(
			"configuration path must be a directory: %s",
			dir)
	}

	var files, overrides []string
	err = nil
	for err != io.EOF {
		var fis []os.FileInfo
		fis, err = f.Readdir(128)
		if err != nil && err != io.EOF {
			return nil, nil, err
		}

		for _, fi := range fis {
			// Ignore directories
			if fi.IsDir() {
				continue
			}

			// Only care about files that are valid to load
			name := fi.Name()
			extValue := ext(name)
			if extValue == "" || IsIgnoredFile(name) {
				continue
			}

			// Determine if we're dealing with an override
			nameNoExt := name[:len(name)-len(extValue)]
			override := nameNoExt == "override" ||
				strings.HasSuffix(nameNoExt, "_override")

			path := filepath.Join(dir, name)
			if override {
				overrides = append(overrides, path)
			} else {
				files = append(files, path)
			}
		}
	}

	return files, overrides, nil
}

// IsIgnoredFile returns true or false depending on whether the
// provided file name is a file that should be ignored.
func IsIgnoredFile(name string) bool {
	return strings.HasPrefix(name, ".") || // Unix-like hidden files
		strings.HasSuffix(name, "~") || // vim
		strings.HasPrefix(name, "#") && strings.HasSuffix(name, "#") // emacs
}
