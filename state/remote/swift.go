package remote

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"os"
	"strings"

	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
	"github.com/rackspace/gophercloud/openstack/objectstorage/v1/containers"
	"github.com/rackspace/gophercloud/openstack/objectstorage/v1/objects"
)

const TFSTATE_NAME = "tfstate.tf"

// SwiftClient implements the Client interface for an Openstack Swift server.
type SwiftClient struct {
	client *gophercloud.ServiceClient
	path   string
}

func swiftFactory(conf map[string]string) (Client, error) {
	client := &SwiftClient{}

	if err := client.validateConfig(conf); err != nil {
		return nil, err
	}

	return client, nil
}

func (c *SwiftClient) validateConfig(conf map[string]string) (err error) {
	if val := os.Getenv("OS_AUTH_URL"); val == "" {
		return fmt.Errorf("missing OS_AUTH_URL environment variable")
	}
	if val := os.Getenv("OS_USERNAME"); val == "" {
		return fmt.Errorf("missing OS_USERNAME environment variable")
	}
	if val := os.Getenv("OS_TENANT_NAME"); val == "" {
		return fmt.Errorf("missing OS_TENANT_NAME environment variable")
	}
	if val := os.Getenv("OS_PASSWORD"); val == "" {
		return fmt.Errorf("missing OS_PASSWORD environment variable")
	}
	path, ok := conf["path"]
	if !ok || path == "" {
		return fmt.Errorf("missing 'path' configuration")
	}

	provider, err := openstack.AuthenticatedClient(gophercloud.AuthOptions{
		IdentityEndpoint: os.Getenv("OS_AUTH_URL"),
		Username:         os.Getenv("OS_USERNAME"),
		TenantName:       os.Getenv("OS_TENANT_NAME"),
		Password:         os.Getenv("OS_PASSWORD"),
	})

	if err != nil {
		return err
	}

	c.path = path
	c.client, err = openstack.NewObjectStorageV1(provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})

	return err
}

func (c *SwiftClient) Get() (*Payload, error) {
	result := objects.Download(c.client, c.path, TFSTATE_NAME, nil)
	bytes, err := result.ExtractContent()

	if err != nil {
		if strings.Contains(err.Error(), "but got 404 instead") {
			return nil, nil
		}
		return nil, err
	}

	hash := md5.Sum(bytes)
	payload := &Payload{
		Data: bytes,
		MD5:  hash[:md5.Size],
	}

	return payload, nil
}

func (c *SwiftClient) Put(data []byte) error {
	if err := c.ensureContainerExists(); err != nil {
		return err
	}

	reader := bytes.NewReader(data)
	result := objects.Create(c.client, c.path, TFSTATE_NAME, reader, nil)

	return result.Err
}

func (c *SwiftClient) Delete() error {
	result := objects.Delete(c.client, c.path, TFSTATE_NAME, nil)
	return result.Err
}

func (c *SwiftClient) ensureContainerExists() error {
	result := containers.Create(c.client, c.path, nil)
	if result.Err != nil {
		return result.Err
	}

	return nil
}
