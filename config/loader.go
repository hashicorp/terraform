package config

import (
	"path/filepath"
)

// Load loads the Terraform configuration from a given file.
//
// This file can be any format that Terraform recognizes, and import any
// other format that Terraform recognizes.
func Load(path string) (*Config, error) {
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
// directory and merges them together.
func LoadDir(path string) (*Config, error) {
	matches, err := filepath.Glob(filepath.Join(path, "*.tf"))
	if err != nil {
		return nil, err
	}

	var result *Config
	for _, f := range matches {
		c, err := Load(f)
		if err != nil {
			return nil, err
		}

		if result != nil {
			result, err = mergeConfig(result, c)
			if err != nil {
				return nil, err
			}
		} else {
			result = c
		}
	}

	return result, nil
}
