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
