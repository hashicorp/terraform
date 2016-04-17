package openstack

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
)

type Config struct {
	Username         string
	UserID           string
	Password         string
	Token            string
	APIKey           string
	IdentityEndpoint string
	TenantID         string
	TenantName       string
	DomainID         string
	DomainName       string
	Insecure         bool
	EndpointType     string
	CACertFile       string

	osClient *gophercloud.ProviderClient
}

func (c *Config) loadAndValidate() error {

	if c.EndpointType != "internal" && c.EndpointType != "internalURL" &&
		c.EndpointType != "admin" && c.EndpointType != "adminURL" &&
		c.EndpointType != "public" && c.EndpointType != "publicURL" &&
		c.EndpointType != "" {
		return fmt.Errorf("Invalid endpoint type provided")
	}

	ao := gophercloud.AuthOptions{
		Username:         c.Username,
		UserID:           c.UserID,
		Password:         c.Password,
		TokenID:          c.Token,
		APIKey:           c.APIKey,
		IdentityEndpoint: c.IdentityEndpoint,
		TenantID:         c.TenantID,
		TenantName:       c.TenantName,
		DomainID:         c.DomainID,
		DomainName:       c.DomainName,
	}

	client, err := openstack.NewClient(ao.IdentityEndpoint)
	if err != nil {
		return err
	}

	if c.CACertFile != "" {

		caCert, err := ioutil.ReadFile(c.CACertFile)
		if err != nil {
			return err
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		config := &tls.Config{
			RootCAs: caCertPool,
		}

		transport := &http.Transport{TLSClientConfig: config}
		client.HTTPClient.Transport = transport
	}

	if c.Insecure {
		// Configure custom TLS settings.
		config := &tls.Config{InsecureSkipVerify: true}
		transport := &http.Transport{TLSClientConfig: config}
		client.HTTPClient.Transport = transport
	}

	err = openstack.Authenticate(client, ao)
	if err != nil {
		return err
	}

	c.osClient = client

	return nil
}

func (c *Config) blockStorageV1Client(region string) (*gophercloud.ServiceClient, error) {
	return openstack.NewBlockStorageV1(c.osClient, gophercloud.EndpointOpts{
		Region:       region,
		Availability: c.getEndpointType(),
	})
}

func (c *Config) computeV2Client(region string) (*gophercloud.ServiceClient, error) {
	return openstack.NewComputeV2(c.osClient, gophercloud.EndpointOpts{
		Region:       region,
		Availability: c.getEndpointType(),
	})
}

func (c *Config) networkingV2Client(region string) (*gophercloud.ServiceClient, error) {
	return openstack.NewNetworkV2(c.osClient, gophercloud.EndpointOpts{
		Region:       region,
		Availability: c.getEndpointType(),
	})
}

func (c *Config) objectStorageV1Client(region string) (*gophercloud.ServiceClient, error) {
	return openstack.NewObjectStorageV1(c.osClient, gophercloud.EndpointOpts{
		Region:       region,
		Availability: c.getEndpointType(),
	})
}

func (c *Config) getEndpointType() gophercloud.Availability {
	if c.EndpointType == "internal" || c.EndpointType == "internalURL" {
		return gophercloud.AvailabilityInternal
	}
	if c.EndpointType == "admin" || c.EndpointType == "adminURL" {
		return gophercloud.AvailabilityAdmin
	}
	return gophercloud.AvailabilityPublic
}
