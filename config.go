package main

import (
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/go-libucl"
)

// TFConfig is the global base configuration that has the
// basic providers registered. Users of this configuration
// should copy it (call the Copy method) before using it so
// that it isn't corrupted.
var TFConfig terraform.Config

// Config is the structure of the configuration for the Terraform CLI.
//
// This is not the configuration for Terraform itself. That is in the
// "config" package.
type Config struct {
	Providers map[string]string
}

// Put the parse flags we use for libucl in a constant so we can get
// equally behaving parsing everywhere.
const libuclParseFlags = libucl.ParserKeyLowercase

// LoadConfig loads the CLI configuration from ".terraformrc" files.
func LoadConfig(path string) (*Config, error) {
	var obj *libucl.Object

	// Parse the file and get the root object.
	parser := libucl.NewParser(libuclParseFlags)
	err := parser.AddFile(path)
	if err == nil {
		obj = parser.Object()
		defer obj.Close()
	}
	defer parser.Close()

	// If there was an error parsing, return now.
	if err != nil {
		return nil, err
	}

	// Build up the result
	var result Config

	if err := obj.Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
