package remote

import (
	"crypto/md5"
	"fmt"
	"strings"

	etcdapi "github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

func etcdFactory(conf map[string]string) (Client, error) {
	path, ok := conf["path"]
	if !ok {
		return nil, fmt.Errorf("missing 'path' configuration")
	}

	endpoints, ok := conf["endpoints"]
	if !ok || endpoints == "" {
		return nil, fmt.Errorf("missing 'endpoints' configuration")
	}

	config := etcdapi.Config{
		Endpoints: strings.Split(endpoints, " "),
	}
	if username, ok := conf["username"]; ok && username != "" {
		config.Username = username
	}
	if password, ok := conf["password"]; ok && password != "" {
		config.Password = password
	}

	client, err := etcdapi.New(config)
	if err != nil {
		return nil, err
	}

	return &EtcdClient{
		Client: client,
		Path:   path,
	}, nil
}

// EtcdClient is a remote client that stores data in etcd.
type EtcdClient struct {
	Client etcdapi.Client
	Path   string
}

func (c *EtcdClient) Get() (*Payload, error) {
	resp, err := etcdapi.NewKeysAPI(c.Client).Get(context.Background(), c.Path, &etcdapi.GetOptions{Quorum: true})
	if err != nil {
		if err, ok := err.(etcdapi.Error); ok && err.Code == etcdapi.ErrorCodeKeyNotFound {
			return nil, nil
		}
		return nil, err
	}
	if resp.Node.Dir {
		return nil, fmt.Errorf("path is a directory")
	}

	data := []byte(resp.Node.Value)
	md5 := md5.Sum(data)
	return &Payload{
		Data: data,
		MD5:  md5[:],
	}, nil
}

func (c *EtcdClient) Put(data []byte) error {
	_, err := etcdapi.NewKeysAPI(c.Client).Set(context.Background(), c.Path, string(data), nil)
	return err
}

func (c *EtcdClient) Delete() error {
	_, err := etcdapi.NewKeysAPI(c.Client).Delete(context.Background(), c.Path, nil)
	return err
}
