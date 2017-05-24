package rackspace

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/rackspace"
)

// Config represents the options a user can provide to authenticate to a
// Rackspace endpoing
type Config struct {
	Username         string
	Password         string
	APIKey           string
	IdentityEndpoint string

	rsClient *gophercloud.ProviderClient
}

func (c *Config) loadAndValidate() error {
	ao := gophercloud.AuthOptions{
		Username:         c.Username,
		Password:         c.Password,
		APIKey:           c.APIKey,
		IdentityEndpoint: c.IdentityEndpoint,
	}

	client, err := rackspace.AuthenticatedClient(ao)
	if err != nil {
		return err
	}
	c.rsClient = client

	version := ""
	_, thisPath, _, _ := runtime.Caller(0)
	versionPath := filepath.Join(strings.Split(thisPath, "terraform")[0], "terraform", "version.go")
	versionFile, err := os.Open(versionPath)
	if err == nil {
		versionFileBytes, err := ioutil.ReadAll(versionFile)
		if err == nil {
			versionFileString := string(versionFileBytes)
			re, err := regexp.Compile(`[0-9]\.[0-9]\.[0-9]`)
			if err == nil {
				version = re.FindString(versionFileString)
			}
		}
	}
	client.UserAgent.Prepend("terraform/" + version)
	log.Printf("[DEBUG] user-agent: %s", client.UserAgent.Join())

	return nil
}

func (c *Config) blockStorageClient(region string) (*gophercloud.ServiceClient, error) {
	return rackspace.NewBlockStorageV1(c.rsClient, gophercloud.EndpointOpts{
		Region: region,
	})
}

func (c *Config) computeClient(region string) (*gophercloud.ServiceClient, error) {
	return rackspace.NewComputeV2(c.rsClient, gophercloud.EndpointOpts{
		Region: region,
	})
}

func (c *Config) networkingClient(region string) (*gophercloud.ServiceClient, error) {
	return rackspace.NewNetworkV2(c.rsClient, gophercloud.EndpointOpts{
		Region: region,
	})
}

func (c *Config) lbClient(region string) (*gophercloud.ServiceClient, error) {
	return rackspace.NewLBV1(c.rsClient, gophercloud.EndpointOpts{
		Region: region,
	})
}
