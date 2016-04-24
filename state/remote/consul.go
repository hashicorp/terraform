package remote

import (
	"crypto/md5"
	"fmt"
	"strings"

	consulapi "github.com/hashicorp/consul/api"
)

func consulFactory(conf map[string]string) (Client, error) {
	path, ok := conf["path"]
	if !ok {
		return nil, fmt.Errorf("missing 'path' configuration")
	}

	config := consulapi.DefaultConfig()
	if token, ok := conf["access_token"]; ok && token != "" {
		config.Token = token
	}
	if addr, ok := conf["address"]; ok && addr != "" {
		config.Address = addr
	}
	if scheme, ok := conf["scheme"]; ok && scheme != "" {
		config.Scheme = scheme
	}
	if auth, ok := conf["http_auth"]; ok && auth != "" {
		var username, password string
		if strings.Contains(auth, ":") {
			split := strings.SplitN(auth, ":", 2)
			username = split[0]
			password = split[1]
		} else {
			username = auth
		}
		config.HttpAuth = &consulapi.HttpBasicAuth{
			Username: username,
			Password: password,
		}
	}

	client, err := consulapi.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &ConsulClient{
		Client: client,
		Path:   path,
	}, nil
}

// ConsulClient is a remote client that stores data in Consul.
type ConsulClient struct {
	Client *consulapi.Client
	Path   string
}

func (c *ConsulClient) Get() (*Payload, error) {
	pair, _, err := c.Client.KV().Get(c.Path, nil)
	if err != nil {
		return nil, err
	}
	if pair == nil {
		return nil, nil
	}

	md5 := md5.Sum(pair.Value)
	return &Payload{
		Data: pair.Value,
		MD5:  md5[:],
	}, nil
}

func (c *ConsulClient) Put(data []byte) error {
	kv := c.Client.KV()
	_, err := kv.Put(&consulapi.KVPair{
		Key:   c.Path,
		Value: data,
	}, nil)
	return err
}

func (c *ConsulClient) Delete() error {
	kv := c.Client.KV()
	_, err := kv.Delete(c.Path, nil)
	return err
}
