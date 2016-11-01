package remote

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/containers"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/objects"
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

	ao := gophercloud.AuthOptions{
		IdentityEndpoint: os.Getenv("OS_AUTH_URL"),
		Username:         os.Getenv("OS_USERNAME"),
		TenantName:       os.Getenv("OS_TENANT_NAME"),
		Password:         os.Getenv("OS_PASSWORD"),
		DomainName:       os.Getenv("OS_DOMAIN_NAME"),
		DomainID:         os.Getenv("OS_DOMAIN_ID"),
	}

	provider, err := openstack.NewClient(ao.IdentityEndpoint)
	if err != nil {
		return err
	}

	config := &tls.Config{}
	insecure := false
	if insecure_env := os.Getenv("OS_INSECURE"); insecure_env != "" {
		insecure, err = strconv.ParseBool(insecure_env)
		if err != nil {
			return err
		}
	}

	if insecure {
		log.Printf("[DEBUG] Insecure mode set")
		config.InsecureSkipVerify = true
	}

	transport := &http.Transport{Proxy: http.ProxyFromEnvironment, TLSClientConfig: config}
	provider.HTTPClient.Transport = transport

	err = openstack.Authenticate(provider, ao)
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

	// Extract any errors from result
	_, err := result.Extract()

	// 404 response is to be expected if the object doesn't already exist!
	if _, ok := err.(gophercloud.ErrDefault404); ok {
		log.Printf("[DEBUG] Container doesn't exist to download.")
		return nil, nil
	}

	bytes, err := result.ExtractContent()
	if err != nil {
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
	createOpts := objects.CreateOpts{
		Content: reader,
	}
	result := objects.Create(c.client, c.path, TFSTATE_NAME, createOpts)

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
