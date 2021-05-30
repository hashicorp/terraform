package main

import (
	"fmt"
	"io/ioutil"

	"github.com/hashicorp/hcl"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/plugin/discovery"
)

var zeroThirteen = discovery.ConstraintStr(">= 0.13.0").MustParse()

type Config struct {
	Terraform TerraformConfig           `hcl:"terraform"`
	Providers map[string]ProviderConfig `hcl:"providers"`
}

type TerraformConfig struct {
	Version discovery.VersionStr `hcl:"version"`
}

type ProviderConfig struct {
	Versions []string `hcl:"versions"`
	Source   string   `hcl:"source"`
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

	var v discovery.Version
	var err error
	if v, err = c.Terraform.Version.Parse(); err != nil {
		return fmt.Errorf("terraform.version: %s", err)
	}

	if !zeroThirteen.Allows(v) {
		return fmt.Errorf("this version of terraform-bundle can only build bundles for Terraform v0.13 and later; build terraform-bundle from a release tag (such as v0.12.*) to construct bundles for earlier versions")
	}

	if c.Providers == nil {
		c.Providers = map[string]ProviderConfig{}
	}

	for k, cs := range c.Providers {
		if cs.Source != "" {
			_, diags := addrs.ParseProviderSourceString(cs.Source)
			if diags.HasErrors() {
				return fmt.Errorf("providers.%s: %s", k, diags.Err().Error())
			}
		}
		if len(cs.Versions) > 0 {
			for _, c := range cs.Versions {
				if _, err := getproviders.ParseVersionConstraints(c); err != nil {
					return fmt.Errorf("providers.%s: %s", k, err)
				}
			}
		} else {
			return fmt.Errorf("provider.%s: required \"versions\" argument not found", k)
		}
	}

	return nil
}
