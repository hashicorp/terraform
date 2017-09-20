package main

import (
	"fmt"
	"io/ioutil"

	"github.com/hashicorp/hcl"
	"github.com/hashicorp/terraform/plugin/discovery"
)

type Config struct {
	Terraform TerraformConfig                      `hcl:"terraform"`
	Providers map[string][]discovery.ConstraintStr `hcl:"providers"`
}

type TerraformConfig struct {
	Version discovery.VersionStr `hcl:"version"`
}

func LoadConfig(src []byte, filename string) (*Config, error) {
	config := &Config{}
	err := hcl.Decode(config, string(src))
	if err != nil {
		return config, err
	}

	err = config.validate()
	return config, err
}

func LoadConfigFile(filename string) (*Config, error) {
	src, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return LoadConfig(src, filename)
}

func (c *Config) validate() error {
	if c.Terraform.Version == "" {
		return fmt.Errorf("terraform.version is required")
	}

	if _, err := c.Terraform.Version.Parse(); err != nil {
		return fmt.Errorf("terraform.version: %s", err)
	}

	if c.Providers == nil {
		c.Providers = map[string][]discovery.ConstraintStr{}
	}

	for k, cs := range c.Providers {
		for _, c := range cs {
			if _, err := c.Parse(); err != nil {
				return fmt.Errorf("providers.%s: %s", k, err)
			}
		}
	}

	return nil
}
