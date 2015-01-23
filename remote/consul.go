package remote

import (
	"crypto/md5"
	"fmt"

	consulapi "github.com/hashicorp/consul/api"
)

// ConsulRemoteClient implements the RemoteClient interface
// for an Consul compatible server.
type ConsulRemoteClient struct {
	client *consulapi.Client
	path   string // KV path
}

func NewConsulRemoteClient(conf map[string]string) (*ConsulRemoteClient, error) {
	client := &ConsulRemoteClient{}
	if err := client.validateConfig(conf); err != nil {
		return nil, err
	}
	return client, nil
}

func (c *ConsulRemoteClient) validateConfig(conf map[string]string) (err error) {
	config := consulapi.DefaultConfig()
	if token, ok := conf["access_token"]; ok && token != "" {
		config.Token = token
	}
	if addr, ok := conf["address"]; ok && addr != "" {
		config.Address = addr
	}
	path, ok := conf["path"]
	if !ok || path == "" {
		return fmt.Errorf("missing 'path' configuration")
	}
	c.path = path
	c.client, err = consulapi.NewClient(config)
	return err
}

func (c *ConsulRemoteClient) GetState() (*RemoteStatePayload, error) {
	kv := c.client.KV()
	pair, _, err := kv.Get(c.path, nil)
	if err != nil {
		return nil, err
	}
	if pair == nil {
		return nil, nil
	}

	// Create the payload
	payload := &RemoteStatePayload{
		State: pair.Value,
	}

	// Generate the MD5
	hash := md5.Sum(payload.State)
	payload.MD5 = hash[:md5.Size]
	return payload, nil
}

func (c *ConsulRemoteClient) PutState(state []byte, force bool) error {
	pair := &consulapi.KVPair{
		Key:   c.path,
		Value: state,
	}
	kv := c.client.KV()
	_, err := kv.Put(pair, nil)
	return err
}

func (c *ConsulRemoteClient) DeleteState() error {
	kv := c.client.KV()
	_, err := kv.Delete(c.path, nil)
	return err
}
