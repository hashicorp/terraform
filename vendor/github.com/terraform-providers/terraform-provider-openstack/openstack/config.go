package openstack

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/swauth"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/hashicorp/terraform/helper/pathorcontents"
	"github.com/hashicorp/terraform/terraform"
)

type Config struct {
	CACertFile        string
	ClientCertFile    string
	ClientKeyFile     string
	Cloud             string
	DefaultDomain     string
	DomainID          string
	DomainName        string
	EndpointType      string
	IdentityEndpoint  string
	Insecure          *bool
	Password          string
	ProjectDomainName string
	ProjectDomainID   string
	Region            string
	Swauth            bool
	TenantID          string
	TenantName        string
	Token             string
	UserDomainName    string
	UserDomainID      string
	Username          string
	UserID            string
	useOctavia        bool

	OsClient *gophercloud.ProviderClient
}

func (c *Config) LoadAndValidate() error {
	// Make sure at least one of auth_url or cloud was specified.
	if c.IdentityEndpoint == "" && c.Cloud == "" {
		return fmt.Errorf("One of 'auth_url' or 'cloud' must be specified")
	}

	validEndpoint := false
	validEndpoints := []string{
		"internal", "internalURL",
		"admin", "adminURL",
		"public", "publicURL",
		"",
	}

	for _, endpoint := range validEndpoints {
		if c.EndpointType == endpoint {
			validEndpoint = true
		}
	}

	if !validEndpoint {
		return fmt.Errorf("Invalid endpoint type provided")
	}

	clientOpts := new(clientconfig.ClientOpts)

	// If a cloud entry was given, base AuthOptions on a clouds.yaml file.
	if c.Cloud != "" {
		clientOpts.Cloud = c.Cloud

		cloud, err := clientconfig.GetCloudFromYAML(clientOpts)
		if err != nil {
			return err
		}

		if c.Region == "" && cloud.RegionName != "" {
			c.Region = cloud.RegionName
		}

		if c.CACertFile == "" && cloud.CACertFile != "" {
			c.CACertFile = cloud.CACertFile
		}

		if c.ClientCertFile == "" && cloud.ClientCertFile != "" {
			c.ClientCertFile = cloud.ClientCertFile
		}

		if c.ClientKeyFile == "" && cloud.ClientKeyFile != "" {
			c.ClientKeyFile = cloud.ClientKeyFile
		}

		if c.Insecure == nil && cloud.Verify != nil {
			v := (!*cloud.Verify)
			c.Insecure = &v
		}
	} else {
		authInfo := &clientconfig.AuthInfo{
			AuthURL:           c.IdentityEndpoint,
			DefaultDomain:     c.DefaultDomain,
			DomainID:          c.DomainID,
			DomainName:        c.DomainName,
			Password:          c.Password,
			ProjectDomainID:   c.ProjectDomainID,
			ProjectDomainName: c.ProjectDomainName,
			ProjectID:         c.TenantID,
			ProjectName:       c.TenantName,
			Token:             c.Token,
			UserDomainID:      c.UserDomainID,
			UserDomainName:    c.UserDomainName,
			Username:          c.Username,
			UserID:            c.UserID,
		}
		clientOpts.AuthInfo = authInfo
	}

	ao, err := clientconfig.AuthOptions(clientOpts)
	if err != nil {
		return err
	}

	client, err := openstack.NewClient(ao.IdentityEndpoint)
	if err != nil {
		return err
	}

	// Set UserAgent
	client.UserAgent.Prepend(terraform.UserAgentString())

	config := &tls.Config{}
	if c.CACertFile != "" {
		caCert, _, err := pathorcontents.Read(c.CACertFile)
		if err != nil {
			return fmt.Errorf("Error reading CA Cert: %s", err)
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(caCert))
		config.RootCAs = caCertPool
	}

	if c.Insecure == nil {
		config.InsecureSkipVerify = false
	} else {
		config.InsecureSkipVerify = *c.Insecure
	}

	if c.ClientCertFile != "" && c.ClientKeyFile != "" {
		clientCert, _, err := pathorcontents.Read(c.ClientCertFile)
		if err != nil {
			return fmt.Errorf("Error reading Client Cert: %s", err)
		}
		clientKey, _, err := pathorcontents.Read(c.ClientKeyFile)
		if err != nil {
			return fmt.Errorf("Error reading Client Key: %s", err)
		}

		cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
		if err != nil {
			return err
		}

		config.Certificates = []tls.Certificate{cert}
		config.BuildNameToCertificate()
	}

	// if OS_DEBUG is set, log the requests and responses
	var osDebug bool
	if os.Getenv("OS_DEBUG") != "" {
		osDebug = true
	}

	transport := &http.Transport{Proxy: http.ProxyFromEnvironment, TLSClientConfig: config}
	client.HTTPClient = http.Client{
		Transport: &LogRoundTripper{
			Rt:      transport,
			OsDebug: osDebug,
		},
	}

	// If using Swift Authentication, there's no need to validate authentication normally.
	if !c.Swauth {
		err = openstack.Authenticate(client, *ao)
		if err != nil {
			return err
		}
	}

	c.OsClient = client

	return nil
}

func (c *Config) determineRegion(region string) string {
	// If a resource-level region was not specified, and a provider-level region was set,
	// use the provider-level region.
	if region == "" && c.Region != "" {
		region = c.Region
	}

	log.Printf("[DEBUG] OpenStack Region is: %s", region)
	return region
}

func (c *Config) blockStorageV1Client(region string) (*gophercloud.ServiceClient, error) {
	return openstack.NewBlockStorageV1(c.OsClient, gophercloud.EndpointOpts{
		Region:       c.determineRegion(region),
		Availability: c.getEndpointType(),
	})
}

func (c *Config) blockStorageV2Client(region string) (*gophercloud.ServiceClient, error) {
	return openstack.NewBlockStorageV2(c.OsClient, gophercloud.EndpointOpts{
		Region:       c.determineRegion(region),
		Availability: c.getEndpointType(),
	})
}

func (c *Config) blockStorageV3Client(region string) (*gophercloud.ServiceClient, error) {
	return openstack.NewBlockStorageV3(c.OsClient, gophercloud.EndpointOpts{
		Region:       c.determineRegion(region),
		Availability: c.getEndpointType(),
	})
}

func (c *Config) computeV2Client(region string) (*gophercloud.ServiceClient, error) {
	return openstack.NewComputeV2(c.OsClient, gophercloud.EndpointOpts{
		Region:       c.determineRegion(region),
		Availability: c.getEndpointType(),
	})
}

func (c *Config) dnsV2Client(region string) (*gophercloud.ServiceClient, error) {
	return openstack.NewDNSV2(c.OsClient, gophercloud.EndpointOpts{
		Region:       c.determineRegion(region),
		Availability: c.getEndpointType(),
	})
}

func (c *Config) identityV3Client(region string) (*gophercloud.ServiceClient, error) {
	return openstack.NewIdentityV3(c.OsClient, gophercloud.EndpointOpts{
		Region:       c.determineRegion(region),
		Availability: c.getEndpointType(),
	})
}

func (c *Config) imageV2Client(region string) (*gophercloud.ServiceClient, error) {
	return openstack.NewImageServiceV2(c.OsClient, gophercloud.EndpointOpts{
		Region:       c.determineRegion(region),
		Availability: c.getEndpointType(),
	})
}

func (c *Config) networkingV2Client(region string) (*gophercloud.ServiceClient, error) {
	return openstack.NewNetworkV2(c.OsClient, gophercloud.EndpointOpts{
		Region:       c.determineRegion(region),
		Availability: c.getEndpointType(),
	})
}

func (c *Config) objectStorageV1Client(region string) (*gophercloud.ServiceClient, error) {
	// If Swift Authentication is being used, return a swauth client.
	if c.Swauth {
		return swauth.NewObjectStorageV1(c.OsClient, swauth.AuthOpts{
			User: c.Username,
			Key:  c.Password,
		})
	}

	return openstack.NewObjectStorageV1(c.OsClient, gophercloud.EndpointOpts{
		Region:       c.determineRegion(region),
		Availability: c.getEndpointType(),
	})
}

func (c *Config) loadBalancerV2Client(region string) (*gophercloud.ServiceClient, error) {
	return openstack.NewLoadBalancerV2(c.OsClient, gophercloud.EndpointOpts{
		Region:       c.determineRegion(region),
		Availability: c.getEndpointType(),
	})
}

func (c *Config) databaseV1Client(region string) (*gophercloud.ServiceClient, error) {
	return openstack.NewDBV1(c.OsClient, gophercloud.EndpointOpts{
		Region:       c.determineRegion(region),
		Availability: c.getEndpointType(),
	})
}

func (c *Config) containerInfraV1Client(region string) (*gophercloud.ServiceClient, error) {
	return openstack.NewContainerInfraV1(c.OsClient, gophercloud.EndpointOpts{
		Region:       c.determineRegion(region),
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
