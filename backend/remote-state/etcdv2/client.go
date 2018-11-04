package etcdv2

import (
	"context"
	"crypto/md5"
	"fmt"

	etcdapi "github.com/coreos/etcd/client"
	"github.com/hashicorp/terraform/state/remote"
)

// EtcdClient is a remote client that stores data in etcd.
type EtcdClient struct {
	Client etcdapi.Client
	Path   string
}

func (c *EtcdClient) Get() (*remote.Payload, error) {
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
	return &remote.Payload{
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
