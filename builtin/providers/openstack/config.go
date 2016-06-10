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
	DefaultDomain     string
	Username          string
	UserID            string
	UserDomainName    string
	UserDomainID      string
	Password          string
	Token             string
	APIKey            string
	IdentityEndpoint  string
	TenantID          string // The tenant_* keyword is deprecated as
	TenantName        string // of the 1.7.0 release in favor of project_*
	ProjectID         string
	ProjectName       string
	ProjectDomainID   string
	ProjectDomainName string
	DomainID          string
	DomainName        string
	Insecure          bool
	EndpointType      string
	CACertFile        string

	osClient *gophercloud.ProviderClient
}

func (c *Config) loadAndValidate() error {

	if c.EndpointType != "internal" && c.EndpointType != "internalURL" &&
		c.EndpointType != "admin" && c.EndpointType != "adminURL" &&
		c.EndpointType != "public" && c.EndpointType != "publicURL" &&
		c.EndpointType != "" {
		return fmt.Errorf("Invalid endpoint type provided")
	}

	// Check if using the old tenant notation or the project notation
	if (c.TenantID == "" && c.TenantName == "") == (c.ProjectID == "" && c.ProjectName == "") {
		return fmt.Errorf("Please provide either a tenant ID/name or a projet ID/name")
	} else if c.ProjectID != "" || c.ProjectName != "" {
		// If using ProjectID/Name, overwrite TenantID/Name because gophercloud doesn't support
		// ProjectID/Name yet.
		c.TenantID = c.ProjectID
		c.TenantName = c.ProjectName
	}

	ao := gophercloud.AuthOptions{
		DefaultDomain:     c.DefaultDomain,
		Username:          c.Username,
		UserID:            c.UserID,
		UserDomainID:      c.UserDomainID,
		UserDomainName:    c.UserDomainName,
		Password:          c.Password,
		TokenID:           c.Token,
		APIKey:            c.APIKey,
		IdentityEndpoint:  c.IdentityEndpoint,
		TenantID:          c.TenantID,
		TenantName:        c.TenantName,
		ProjectDomainID:   c.ProjectDomainID,
		ProjectDomainName: c.ProjectDomainName,
		DomainID:          c.DomainID,
		DomainName:        c.DomainName,
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
