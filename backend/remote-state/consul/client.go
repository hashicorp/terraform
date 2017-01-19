package consul

import (
	"crypto/md5"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/terraform/state/remote"
)

// RemoteClient is a remote client that stores data in Consul.
type RemoteClient struct {
	Client *consulapi.Client
	Path   string
}

func (c *RemoteClient) Get() (*remote.Payload, error) {
	pair, _, err := c.Client.KV().Get(c.Path, nil)
	if err != nil {
		return nil, err
	}
	if pair == nil {
		return nil, nil
	}

	md5 := md5.Sum(pair.Value)
	return &remote.Payload{
		Data: pair.Value,
		MD5:  md5[:],
	}, nil
}

func (c *RemoteClient) Put(data []byte) error {
	kv := c.Client.KV()
	_, err := kv.Put(&consulapi.KVPair{
		Key:   c.Path,
		Value: data,
	}, nil)
	return err
}

func (c *RemoteClient) Delete() error {
	kv := c.Client.KV()
	_, err := kv.Delete(c.Path, nil)
	return err
}
