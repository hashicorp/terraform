package aws

import (
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/mitchellh/goamz/aws"
)

type Config struct {
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	Region    string `mapstructure:"region"`
}

// AWSAuth returns a valid aws.Auth object for access to AWS services, or
// an error if the authentication couldn't be resolved.
//
// TODO(mitchellh): Test in some way.
func (c *Config) AWSAuth() (aws.Auth, error) {
	auth, err := aws.GetAuth(c.AccessKey, c.SecretKey)
	if err == nil {
		// Store the accesskey and secret that we got...
		c.AccessKey = auth.AccessKey
		c.SecretKey = auth.SecretKey
	}

	return auth, err
}

// IsValidRegion returns true if the configured region is a valid AWS
// region and false if it's not
func (c *Config) IsValidRegion() bool {
	var regions = [8]string{"us-east-1", "us-west-2", "us-west-1", "eu-west-1",
		"ap-southeast-1", "ap-southeast-2", "ap-northeast-1", "sa-east-1"}

	for _, valid := range regions {
		if c.Region == valid {
			return true
		}
	}
	return false
}

// AWSRegion returns the configured region.
//
// TODO(mitchellh): Test in some way.
func (c *Config) AWSRegion() (aws.Region, error) {
	if c.Region != "" {
		if c.IsValidRegion() {
			return aws.Regions[c.Region], nil
		} else {
			return aws.Region{}, fmt.Errorf("Not a valid region: %s", c.Region)
		}
	}

	if v := os.Getenv("AWS_REGION"); v != "" {
		return aws.Regions[v], nil
	}

	md, err := aws.GetMetaData("placement/availability-zone")
	if err != nil {
		return aws.Region{}, err
	}

	region := strings.TrimRightFunc(string(md), unicode.IsLetter)
	return aws.Regions[region], nil
}
