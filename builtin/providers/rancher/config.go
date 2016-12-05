package rancher

import (
	"log"

	rancherClient "github.com/rancher/go-rancher/client"
	"github.com/raphink/go-rancher/catalog"
)

type Config struct {
	*rancherClient.RancherClient
	APIURL    string
	AccessKey string
	SecretKey string
}

// Create creates a generic Rancher client
func (c *Config) CreateClient() error {
	client, err := rancherClient.NewRancherClient(&rancherClient.ClientOpts{
		Url:       c.APIURL,
		AccessKey: c.AccessKey,
		SecretKey: c.SecretKey,
	})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Rancher Client configured for url: %s", c.APIURL)

	c.RancherClient = client

	return nil
}

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

func (c *Config) RegistryClient(id string) (*rancherClient.RancherClient, error) {
	reg, err := c.Registry.ById(id)
	if err != nil {
		return nil, err
	}

	return c.EnvironmentClient(reg.AccountId)
}

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
