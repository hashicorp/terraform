package rancher

import (
	"log"

	rancherClient "github.com/rancher/go-rancher/client"
	"github.com/raphink/go-rancher/catalog"
)

// Config is the configuration parameters for a Rancher API
type Config struct {
	APIURL    string
	AccessKey string
	SecretKey string
}

// GlobalClient creates a Rancher client scoped to the global API
func (c *Config) GlobalClient() (*rancherClient.RancherClient, error) {
	client, err := rancherClient.NewRancherClient(&rancherClient.ClientOpts{
		Url:       c.APIURL,
		AccessKey: c.AccessKey,
		SecretKey: c.SecretKey,
	})
	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] Rancher Client configured for url: %s", c.APIURL)

	return client, nil
}

// EnvironmentClient creates a Rancher client scoped to an Environment's API
func (c *Config) EnvironmentClient(env string) (*rancherClient.RancherClient, error) {

	url := c.APIURL + "/projects/" + env + "/schemas"
	client, err := rancherClient.NewRancherClient(&rancherClient.ClientOpts{
		Url:       url,
		AccessKey: c.AccessKey,
		SecretKey: c.SecretKey,
	})
	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] Rancher Client configured for url: %s", url)

	return client, nil
}

// RegistryClient creates a Rancher client scoped to a Registry's API
func (c *Config) RegistryClient(id string) (*rancherClient.RancherClient, error) {
	client, err := c.GlobalClient()
	if err != nil {
		return nil, err
	}
	reg, err := client.Registry.ById(id)
	if err != nil {
		return nil, err
	}

	return c.EnvironmentClient(reg.AccountId)
}

// CatalogClient creates a Rancher client scoped to a Catalog's API
func (c *Config) CatalogClient() (*catalog.RancherClient, error) {

	url := c.APIURL + "-catalog/schemas"
	client, err := catalog.NewRancherClient(&catalog.ClientOpts{
		Url:       url,
		AccessKey: c.AccessKey,
		SecretKey: c.SecretKey,
	})
	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] Rancher Catalog Client configured for url: %s", url)

	return client, nil
}
