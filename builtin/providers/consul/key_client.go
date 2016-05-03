package consul

import (
	"fmt"
	"log"

	consulapi "github.com/hashicorp/consul/api"
)

// keyClient is a wrapper around the upstream Consul client that is
// specialized for Terraform's manipulations of the key/value store.
type keyClient struct {
	client *consulapi.KV
	qOpts  *consulapi.QueryOptions
	wOpts  *consulapi.WriteOptions
}

func newKeyClient(realClient *consulapi.KV, dc, token string) *keyClient {
	qOpts := &consulapi.QueryOptions{Datacenter: dc, Token: token}
	wOpts := &consulapi.WriteOptions{Datacenter: dc, Token: token}

	return &keyClient{
		client: realClient,
		qOpts:  qOpts,
		wOpts:  wOpts,
	}
}

func (c *keyClient) Get(path string) (string, error) {
	log.Printf(
		"[DEBUG] Reading key '%s' in %s",
		path, c.qOpts.Datacenter,
	)
	pair, _, err := c.client.Get(path, c.qOpts)
	if err != nil {
		return "", fmt.Errorf("Failed to read Consul key '%s': %s", path, err)
	}
	value := ""
	if pair != nil {
		value = string(pair.Value)
	}
	return value, nil
}

func (c *keyClient) GetUnderPrefix(pathPrefix string) (map[string]string, error) {
	log.Printf(
		"[DEBUG] Listing keys under '%s' in %s",
		pathPrefix, c.qOpts.Datacenter,
	)
	pairs, _, err := c.client.List(pathPrefix, c.qOpts)
	if err != nil {
		return nil, fmt.Errorf(
			"Failed to list Consul keys under prefix '%s': %s", pathPrefix, err,
		)
	}
	value := map[string]string{}
	for _, pair := range pairs {
		subKey := pair.Key[len(pathPrefix):]
		value[subKey] = string(pair.Value)
	}
	return value, nil
}

func (c *keyClient) Put(path, value string) error {
	log.Printf(
		"[DEBUG] Setting key '%s' to '%v' in %s",
		path, value, c.wOpts.Datacenter,
	)
	pair := consulapi.KVPair{Key: path, Value: []byte(value)}
	if _, err := c.client.Put(&pair, c.wOpts); err != nil {
		return fmt.Errorf("Failed to write Consul key '%s': %s", path, err)
	}
	return nil
}

func (c *keyClient) Delete(path string) error {
	log.Printf(
		"[DEBUG] Deleting key '%s' in %s",
		path, c.wOpts.Datacenter,
	)
	if _, err := c.client.Delete(path, c.wOpts); err != nil {
		return fmt.Errorf("Failed to delete Consul key '%s': %s", path, err)
	}
	return nil
}

func (c *keyClient) DeleteUnderPrefix(pathPrefix string) error {
	log.Printf(
		"[DEBUG] Deleting all keys under prefix '%s' in %s",
		pathPrefix, c.wOpts.Datacenter,
	)
	if _, err := c.client.DeleteTree(pathPrefix, c.wOpts); err != nil {
		return fmt.Errorf("Failed to delete Consul keys under '%s': %s", pathPrefix, err)
	}
	return nil
}
